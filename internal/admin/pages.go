package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
)

const pageSlug = "pages"

const (
	pageActionDelete = "delete"
	pageActionRevert = "revert"
	pageActionRename = "rename"
)

func init() {
	register(screen{slug: pageSlug, label: "admin.pages", need: perms.EditArticles, serve: (*Handler).pages})
}

func (h *Handler) pages(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+pageSlug), "/")
	if rest == "" {
		if r.Method == http.MethodPost {
			return h.pageBatch(w, r, loc)
		}
		return h.pageList(w, r, loc, "")
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	if r.Method == http.MethodPost {
		return h.savePage(w, r, loc, id)
	}
	return h.pageForm(w, r, loc, id, "")
}

func (h *Handler) pageList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem string) error {
	ctx := r.Context()
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	category := r.URL.Query().Get("c")
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	found, total, err := h.deps.DB.AdminPages(ctx, siteID(ctx), query, category, perPage, (page-1)*perPage)
	if err != nil {
		return err
	}
	categories, err := h.deps.DB.AdminPageCategories(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.pages"), "page_list.html", map[string]any{
		"Pages":      found,
		"Categories": categories,
		"Query":      query,
		"Category":   category,
		"Page":       page,
		"Pages_":     (total + perPage - 1) / perPage,
		"Total":      total,
		"Error":      problem,
		"CSRF":       csrf.Issue(w, r),
		"Action":     Prefix + pageSlug + "/",
		"Base":       Prefix + pageSlug + "/",
	})
}

func (h *Handler) pageForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, id int64, problem string) error {
	ctx := r.Context()
	article, err := h.deps.DB.ArticleByID(ctx, siteID(ctx), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	source, err := h.deps.DB.LatestSource(ctx, id)
	if err != nil {
		return err
	}
	one := 1
	log, err := h.deps.DB.ArticleLog(ctx, id, 0, &one)
	if err != nil {
		return err
	}
	head := 0
	if len(log) > 0 {
		head = log[0].RevNumber
	}
	return h.page(w, r, loc, article.FullName(), "page_edit.html", map[string]any{
		"Article": article,
		"Full":    article.FullName(),
		"Source":  source,
		"Head":    head,
		"Error":   problem,
		"CSRF":    csrf.Issue(w, r),
		"Action":  Prefix + pageSlug + "/" + strconv.FormatInt(id, 10),
		"Back":    Prefix + pageSlug + "/",
	})
}

func (h *Handler) savePage(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, id int64) error {
	if !h.verified(w, r) {
		return nil
	}
	ctx := r.Context()
	article, err := h.deps.DB.ArticleByID(ctx, siteID(ctx), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}

	body := map[string]any{"pageId": article.FullName()}
	if r.PostFormValue("action") == pageActionRename {
		wanted := strings.TrimSpace(r.PostFormValue("full_name"))
		if wanted == "" || wanted == article.FullName() {
			return h.pageForm(w, r, loc, id, loc.T("admin.page-no-name"))
		}
		body["pageId"] = wanted
	} else {
		source := r.PostFormValue("source")
		if strings.TrimSpace(source) == "" {
			return h.pageForm(w, r, loc, id, loc.T("admin.page-no-source"))
		}
		body["source"] = source
		body["title"] = strings.TrimSpace(r.PostFormValue("title"))
	}

	status, answer, err := h.callArticle(r, http.MethodPut, article.FullName(), "", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return h.pageForm(w, r, loc, id, apiProblem(loc, answer))
	}
	h.noteID(r, db.AdminChanged, pageSlug, id, article.FullName())
	redirect(w, Prefix+pageSlug+"/"+strconv.FormatInt(id, 10))
	return nil
}

type batchResult struct {
	Full    string
	Problem string
}

func (h *Handler) pageBatch(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if !h.verified(w, r) {
		return nil
	}
	ctx := r.Context()
	action := r.PostFormValue("action")
	steps := atoi(r.PostFormValue("steps"))
	if action == pageActionRevert && steps < 1 {
		steps = 1
	}

	var picked []db.AdminPageRow
	for _, raw := range r.PostForm["id"] {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}
		article, err := h.deps.DB.ArticleByID(ctx, siteID(ctx), id)
		if errors.Is(err, db.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		picked = append(picked, db.AdminPageRow{
			ID: article.ID, Category: article.Category, Name: article.Name, Title: article.Title,
		})
	}
	if len(picked) == 0 {
		return h.pageList(w, r, loc, loc.T("admin.page-none-picked"))
	}

	if r.PostFormValue("confirm") == "" {
		return h.page(w, r, loc, loc.T("admin.pages"), "page_confirm.html", map[string]any{
			"Picked": picked,
			"What":   action,
			"Steps":  steps,
			"CSRF":   csrf.Issue(w, r),
			"Action": Prefix + pageSlug + "/",
			"Back":   Prefix + pageSlug + "/",
		})
	}

	results := make([]batchResult, 0, len(picked))
	for _, one := range picked {
		results = append(results, batchResult{
			Full:    one.FullName(),
			Problem: h.applyOne(r, loc, one, action, steps),
		})
	}
	return h.page(w, r, loc, loc.T("admin.pages"), "page_done.html", map[string]any{
		"Results": results,
		"What":    action,
		"Back":    Prefix + pageSlug + "/",
	})
}

func (h *Handler) applyOne(r *http.Request, loc *i18n.Localizer, one db.AdminPageRow, action string, steps int) string {
	switch action {
	case pageActionDelete:
		status, answer, err := h.callArticle(r, http.MethodDelete, one.FullName(), "", nil)
		if err != nil {
			return err.Error()
		}
		if status != http.StatusOK {
			return apiProblem(loc, answer)
		}
		h.noteID(r, db.AdminDeleted, pageSlug, one.ID, one.FullName())
		return ""
	case pageActionRevert:
		target, err := h.revertTarget(r, one.ID, steps)
		if err != nil {
			return err.Error()
		}
		if target < 0 {
			return loc.T("admin.page-no-older")
		}
		status, answer, err := h.callArticle(r, http.MethodPut, one.FullName(), "log",
			map[string]any{"revNumber": target})
		if err != nil {
			return err.Error()
		}
		if status != http.StatusOK {
			return apiProblem(loc, answer)
		}
		h.noteID(r, db.AdminChanged, pageSlug, one.ID, one.FullName())
		return ""
	}
	return loc.T("admin.page-bad-action")
}

func (h *Handler) revertTarget(r *http.Request, id int64, steps int) (int, error) {
	limit := steps + 1
	log, err := h.deps.DB.ArticleLog(r.Context(), id, 0, &limit)
	if err != nil {
		return 0, err
	}
	if len(log) <= steps {
		return -1, nil
	}
	return log[steps].RevNumber, nil
}

// Driven in process because the article API is the only place that knows what
// a page write drags along. The context carries the admin into the history.
func (h *Handler) callArticle(r *http.Request, method, name, tail string, body any) (int, string, error) {
	if h.deps.Articles == nil {
		return 0, "", errors.New("admin: no article API to call")
	}
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return 0, "", err
		}
	}
	url := webapi.ArticlesPrefix + name
	if tail != "" {
		url += "/" + tail
	}
	ctx := csrf.Exempt(r.Context())
	sub, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	sub.Header.Set("Content-Type", "application/json")
	sub.Host = r.Host

	rec := &recorder{status: http.StatusOK}
	h.deps.Articles.ServeHTTP(rec, sub)
	return rec.status, rec.body.String(), nil
}

type recorder struct {
	status int
	body   bytes.Buffer
	header http.Header
}

func (rec *recorder) Header() http.Header {
	if rec.header == nil {
		rec.header = http.Header{}
	}
	return rec.header
}

func (rec *recorder) Write(p []byte) (int, error) { return rec.body.Write(p) }
func (rec *recorder) WriteHeader(status int)      { rec.status = status }

func apiProblem(loc *i18n.Localizer, answer string) string {
	var body struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(answer), &body) == nil && body.Error != "" {
		return body.Error
	}
	return loc.T("admin.page-failed")
}
