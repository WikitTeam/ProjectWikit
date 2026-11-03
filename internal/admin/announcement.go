package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const announcementSlug = "announcements"

func init() {
	register(screen{slug: announcementSlug, label: "admin.announcements", need: perms.ManageSite, serve: (*Handler).announcements})
}

func (h *Handler) announcements(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		notFound(w)
		return nil
	}
	instance := instanceSuper(ctx)
	if rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+announcementSlug), "/"); rest != "" {
		return h.announcementDetail(w, r, loc, current, rest)
	}
	problem := ""
	var preview *announcementPreview
	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		title := strings.TrimSpace(r.PostFormValue("title"))
		body := strings.TrimSpace(r.PostFormValue("body"))
		wide := instance && r.PostFormValue("scope") == "instance"
		checked := announcementDigest(title, body, wide)
		switch {
		case body == "":
			problem = loc.T("admin.announcement-empty")
		case r.PostFormValue("send") == "" || r.PostFormValue("checked") != checked:
			if r.PostFormValue("send") != "" {
				problem = loc.T("admin.announcement-changed")
			}
			html, err := h.renderAnnouncement(r, loc, current, body)
			if err != nil {
				return err
			}
			preview = &announcementPreview{Title: title, HTML: html, Checked: checked}
		default:
			if err := h.sendAnnouncement(r, loc, current, title, body, wide); err != nil {
				return err
			}
			screen := "site"
			if wide {
				screen = "instance"
			}
			h.note(r, db.AdminCreated, announcementSlug, screen, title)
			redirect(w, Prefix+announcementSlug+"/?sent")
			return nil
		}
	}

	history, err := h.deps.DB.Announcements(ctx, current.ID)
	if err != nil {
		return err
	}
	sent := r.URL.Query().Has("sent")
	return h.page(w, r, loc, loc.T("admin.announcements"), "announcement.html", map[string]any{
		"Error":    problem,
		"Title":    r.PostFormValue("title"),
		"Body":     r.PostFormValue("body"),
		"Instance": instance,
		"Wide":     r.PostFormValue("scope") == "instance",
		"Preview":  preview,
		"Sent":     sent,
		"History":  history,
		"CSRF":     csrf.Issue(w, r),
		"Action":   Prefix + announcementSlug + "/",
		"Base":     Prefix + announcementSlug + "/",
	})
}

func (h *Handler) announcementDetail(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, current *db.Site, rest string) error {
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	found, message, err := h.deps.DB.Announcement(r.Context(), current.ID, id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.announcements"), "announcement_detail.html", map[string]any{
		"Announcement": found,
		"Message":      message,
		"Back":         Prefix + announcementSlug + "/",
	})
}

type announcementPreview struct {
	Title   string
	HTML    string
	Checked string
}

// Sending is only offered for exactly what was previewed. A field changed after
// the preview no longer matches this, so the screen asks for a fresh one.
func announcementDigest(title, body string, wide bool) string {
	scope := "site"
	if wide {
		scope = "instance"
	}
	joined, _ := json.Marshal([]string{scope, title, body})
	sum := sha256.Sum256(joined)
	return hex.EncodeToString(sum[:])
}

func (h *Handler) renderAnnouncement(r *http.Request, loc *i18n.Localizer, current *db.Site, body string) (string, error) {
	ctx := r.Context()
	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.Env(ctx, loc, current, auth.FromContext(ctx))
	info, err := env.PageInfo(nil)
	if err != nil {
		return "", err
	}
	out, err := env.HTML(body, info, env.Callbacks(nil, page.NewContext(nil, nil, nil, nil)), renderer.ModeSystem)
	return out.Body, err
}

func (h *Handler) sendAnnouncement(r *http.Request, loc *i18n.Localizer, current *db.Site, title, body string, wide bool) error {
	ctx := r.Context()
	author := auth.FromContext(ctx)
	html, err := h.renderAnnouncement(r, loc, current, body)
	if err != nil {
		return err
	}

	var scope *int64
	siteID := current.ID
	if !wide {
		scope = &siteID
	}
	audience, err := h.deps.DB.AnnouncementAudience(ctx, scope)
	if err != nil {
		return err
	}
	name := ""
	if author != nil {
		name = author.DisplayLabel()
	}
	meta, err := json.Marshal(map[string]any{"title": title, "message": html, "author": name, "instance": wide})
	if err != nil {
		return err
	}
	notifySite := siteID
	if wide {
		notifySite = 0
	}
	return h.deps.DB.SendNotification(ctx, notifySite, db.NotifyAnnouncement, string(meta), audience, time.Now().UTC())
}
