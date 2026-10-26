package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/mirror"
)

const mirrorSlug = "include-mirrors"

func init() {
	register(screen{slug: mirrorSlug, label: "admin.mirrors", all: true, serve: (*Handler).mirrors})
}

func (h *Handler) mirrors(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+mirrorSlug), "/")
	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		if r.PostFormValue("action") == "delete" && !instanceSuper(r.Context()) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return nil
		}
		return h.mirrorAction(w, r, loc, rest)
	}
	if rest != "" {
		return h.mirrorForm(w, r, loc, rest)
	}
	return h.mirrorList(w, r, loc, "")
}

func (h *Handler) mirrorAction(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	ctx := r.Context()
	if rest == "" {
		switch r.PostFormValue("action") {
		case "scan":
			sum, err := mirror.Scan(ctx, h.deps.DB, siteID(ctx))
			if err != nil {
				return err
			}
			h.note(r, db.AdminChanged, mirrorSlug, "", strconv.Itoa(sum.Refs))
			return h.mirrorList(w, r, loc, loc.T("admin.mirror-scanned",
				"pages", sum.Pages, "refs", sum.Refs, "instance", sum.Instance, "mirrored", sum.Mirrored, "missing", sum.Missing))
		case "fetch":
			id := siteID(ctx)
			targets, err := mirror.Missing(ctx, h.deps.DB, &id)
			if err != nil {
				return err
			}
			if h.deps.Mirrors != nil && len(targets) > 0 && h.deps.Mirrors.Start(targets) {
				h.note(r, db.AdminChanged, mirrorSlug, "", strconv.Itoa(len(targets)))
			}
		}
		redirect(w, Prefix+mirrorSlug+"/")
		return nil
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.Mirror(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	switch r.PostFormValue("action") {
	case "refetch":
		if h.deps.Mirrors != nil && h.deps.Mirrors.Start([]mirror.Target{{Wiki: row.Wiki, Page: row.Page}}) {
			h.noteID(r, db.AdminChanged, mirrorSlug, id, row.Wiki+":"+row.Page)
		}
		redirect(w, Prefix+mirrorSlug+"/"+rest)
	case "delete":
		if err := h.deps.DB.DeleteMirror(ctx, id); err != nil {
			return err
		}
		h.noteID(r, db.AdminDeleted, mirrorSlug, id, row.Wiki+":"+row.Page)
		redirect(w, Prefix+mirrorSlug+"/")
	default:
		notFound(w)
	}
	return nil
}

func (h *Handler) mirrorJob() mirror.Job {
	if h.deps.Mirrors == nil {
		return mirror.Job{}
	}
	return h.deps.Mirrors.Status()
}

func (h *Handler) mirrorList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, done string) error {
	ctx := r.Context()
	id := siteID(ctx)
	missing, err := h.deps.DB.MissingIncludes(ctx, &id)
	if err != nil {
		return err
	}
	mirrors, err := h.deps.DB.Mirrors(ctx, id)
	if err != nil {
		return err
	}
	local, err := h.deps.DB.InstanceSiteSlugs(ctx)
	if err != nil {
		return err
	}

	view := r.URL.Query().Get("view")
	if view != "all" && view != "used" {
		view = "missing"
	}
	used := 0
	for _, m := range mirrors {
		if m.Refs > 0 {
			used++
		}
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	matches := func(wiki, page string) bool {
		return query == "" || strings.Contains(":"+wiki+":"+page, query)
	}
	var shownMissing []db.MissingInclude
	for _, m := range missing {
		if matches(m.Wiki, m.Page) {
			shownMissing = append(shownMissing, m)
		}
	}
	var shownMirrors []db.Mirror
	for _, m := range mirrors {
		if matches(m.Wiki, m.Page) && (view != "used" || m.Refs > 0) {
			shownMirrors = append(shownMirrors, m)
		}
	}
	total := len(shownMissing)
	if view != "missing" {
		total = len(shownMirrors)
	}
	page := max(atoi(r.URL.Query().Get("page")), 1)
	from, to := min((page-1)*perPage, total), min(page*perPage, total)
	if view == "missing" {
		shownMissing = shownMissing[from:to]
	} else {
		shownMirrors = shownMirrors[from:to]
	}

	return h.page(w, r, loc, loc.T("admin.mirrors"), "mirror_list.html", map[string]any{
		"Job":          h.mirrorJob(),
		"View":         view,
		"Query":        query,
		"Missing":      shownMissing,
		"Mirrors":      shownMirrors,
		"MissingCount": len(missing),
		"MirrorCount":  len(mirrors),
		"UsedCount":    used,
		"Total":        total,
		"Page":         page,
		"Pages":        (total + perPage - 1) / perPage,
		"Shadowed": func(wiki string) bool {
			return local[wiki]
		},
		"Done":   done,
		"CSRF":   csrf.Issue(w, r),
		"Action": Prefix + mirrorSlug + "/",
	})
}

func (h *Handler) mirrorForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	ctx := r.Context()
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.Mirror(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	referrers, err := h.deps.DB.MirrorReferrers(ctx, siteID(ctx), row.Wiki, row.Page)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.mirrors"), "mirror_form.html", map[string]any{
		"Mirror":    row,
		"Referrers": referrers,
		"Job":       h.mirrorJob(),
		"MayDelete": instanceSuper(ctx),
		"CSRF":      csrf.Issue(w, r),
		"Action":    Prefix + mirrorSlug + "/" + rest,
		"Back":      Prefix + mirrorSlug + "/",
	})
}
