package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/articlepage"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

func articleHandler(conn *db.DB, p *paths.Paths, assets fs.FS, sidecar, secret, timezone string, log *slog.Logger) (http.Handler, func(), error) {
	engine, closeEngine, err := newRenderer(sidecar)
	if err != nil {
		return nil, nil, err
	}
	bundle, err := i18n.Load(p.Locales())
	if err != nil {
		closeEngine()
		return nil, nil, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		closeEngine()
		return nil, nil, err
	}

	pages := articlepage.New(articlepage.Deps{
		DB:          conn,
		Engine:      engine,
		Bundle:      bundle,
		Icons:       roles.FileIcons(p.Files()),
		Assets:      static.NewAssets(assets),
		TimeZone:    location,
		GoogleTagID: envOr(envGoogleTag, ""),
	})

	// Without the key nothing can be verified, so every visitor stays
	// anonymous rather than being trusted on the cookie alone.
	if secret == "" {
		return pages, closeEngine, nil
	}
	resolver := auth.NewResolver(session.New(secret), conn, conn, log)
	return resolver.Middleware(pages), closeEngine, nil
}
