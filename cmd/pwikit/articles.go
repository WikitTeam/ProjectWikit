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
	"github.com/WikitTeam/ProjectWikit/internal/localitem"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

type pageStack struct {
	articles http.Handler
	code     http.Handler
	html     http.Handler
	theme    http.Handler
	close    func()
}

func newPageStack(conn *db.DB, p *paths.Paths, assets fs.FS, sidecar, secret, timezone string, log *slog.Logger) (*pageStack, error) {
	engine, closeEngine, err := newRenderer(sidecar)
	if err != nil {
		return nil, err
	}
	bundle, err := i18n.Load(p.Locales())
	if err != nil {
		closeEngine()
		return nil, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		closeEngine()
		return nil, err
	}
	icons := roles.FileIcons(p.Files())

	pages := articlepage.New(articlepage.Deps{
		DB:          conn,
		Engine:      engine,
		Bundle:      bundle,
		Icons:       icons,
		Assets:      static.NewAssets(assets),
		TimeZone:    location,
		GoogleTagID: envOr(envGoogleTag, ""),
		Log:         log,
	})
	items := localitem.Deps{DB: conn, Engine: engine, Bundle: bundle, Icons: icons, Log: log}

	stack := &pageStack{
		articles: pages,
		code:     localitem.NewCode(items),
		html:     localitem.NewHTML(items),
		theme:    localitem.NewTheme(items),
		close:    closeEngine,
	}

	// Without the key nothing can be verified, so every visitor stays
	// anonymous rather than being trusted on the cookie alone.
	if secret == "" {
		return stack, nil
	}
	resolver := auth.NewResolver(session.New(secret), conn, conn, log)
	stack.articles = resolver.Middleware(stack.articles)
	stack.code = resolver.Middleware(stack.code)
	stack.html = resolver.Middleware(stack.html)
	stack.theme = resolver.Middleware(stack.theme)
	return stack, nil
}
