// Package account answers the pages a visitor signs in, signs up and signs out on.
package account

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/mail"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

const (
	LoginPath  = "/-/login"
	LogoutPath = "/-/logout"

	homePath = "/"
)

type Deps struct {
	DB       *db.DB
	Sessions *session.Store
	Tokens   token.Generator
	Verifier Verifier
	Mail     mail.Sender
	Engine   renderer.Renderer
	Icons    roles.IconLoader
	Bundle   *i18n.Bundle
	Assets   *static.Assets
	Trust    *proxyheader.Trust
	Log      *slog.Logger
}

func (d Deps) clientIP(r *http.Request) *netip.Addr {
	if d.Trust == nil {
		return nil
	}
	if addr, ok := d.Trust.ClientIP(r); ok {
		return &addr
	}
	return nil
}

func (d Deps) logger() *slog.Logger {
	if d.Log == nil {
		return slog.Default()
	}
	return d.Log
}

func redirect(w http.ResponseWriter, location string, status int) {
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(status)
}

func destination(r *http.Request) string {
	to := r.URL.Query().Get("to")
	if to == "" || !strings.HasPrefix(to, "/") || strings.HasPrefix(to, "//") {
		return homePath
	}
	return to
}

func (d Deps) page(r *http.Request, loc *i18n.Localizer, current *db.Site, title, content string) (string, error) {
	theme, err := site.ThemeURLByID(r.Context(), d.DB, current.SystemThemeID)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	err = shell.New(loc, d.Assets).SystemPage(&out, shell.System{
		Title:     title,
		SiteTitle: current.Title,
		ThemeURL:  theme,
		BodyClass: "wikit-page",
		Content:   content,
	})
	return out.String(), err
}

func (d Deps) welcome(ctx context.Context, userID int64) error {
	current := site.FromContext(ctx)
	if current == nil {
		return d.DB.SendNotification(ctx, 0, db.NotifyWelcome, "{}", []int64{userID}, time.Now())
	}
	loc := d.Bundle.For(ctx)
	source := strings.TrimSpace(current.Welcome)
	if source == "" {
		source = loc.T("account.welcome-default", "site", current.Title)
	}
	meta := "{}"
	if body, err := d.welcomeBody(ctx, loc, current, source); err != nil {
		d.logger().Error("render welcome message", "site", current.Slug, "err", err)
	} else if encoded, err := json.Marshal(map[string]string{"message": body}); err == nil {
		meta = string(encoded)
	}
	return d.DB.SendNotification(ctx, current.ID, db.NotifyWelcome, meta, []int64{userID}, time.Now())
}

func (d Deps) welcomeBody(ctx context.Context, loc *i18n.Localizer, current *db.Site, source string) (string, error) {
	env := pagerender.Deps{DB: d.DB, Engine: d.Engine, Icons: d.Icons}.Env(ctx, loc, current, nil)
	info, err := env.PageInfo(nil)
	if err != nil {
		return "", err
	}
	html, err := env.HTML(source, info, env.Callbacks(nil, page.NewContext(nil, nil, nil, nil)), renderer.ModeSystem)
	if err != nil {
		return "", err
	}
	return html.Body, nil
}

func authIcon(s *db.Site) string {
	if s == nil || s.AuthIcon == "" {
		return ""
	}
	return "/local--files/" + s.AuthIcon
}

func siteID(ctx context.Context) int64 {
	if current := site.FromContext(ctx); current != nil {
		return current.ID
	}
	return 0
}
