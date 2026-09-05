// Package account answers the pages a visitor signs in, signs up and signs out on.
package account

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/mail"
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
	Bundle   *i18n.Bundle
	Assets   *static.Assets
	TimeZone *time.Location
	Log      *slog.Logger
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
	err = shell.New(loc, d.Assets, d.TimeZone).SystemPage(&out, shell.System{
		Title:     title,
		ThemeURL:  theme,
		BodyClass: "wikit-page",
		Content:   content,
	})
	return out.String(), err
}

func (d Deps) welcome(ctx context.Context, userID int64) error {
	return d.DB.SendNotification(ctx, db.NotifyWelcome, "{}", []int64{userID}, time.Now())
}

func authIcon(s *db.Site) string {
	if s == nil || s.AuthIcon == "" {
		return ""
	}
	return "/local--files/" + s.AuthIcon
}
