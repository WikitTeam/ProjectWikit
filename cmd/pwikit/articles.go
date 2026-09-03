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
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/userpage"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
)

type pageStack struct {
	articles    http.Handler
	code        http.Handler
	html        http.Handler
	theme       http.Handler
	moduleAPI   http.Handler
	preview     http.Handler
	profile     http.Handler
	profileForm http.Handler
	articleAPI  http.Handler
	close       func()
}

func newPageStack(conn *db.DB, p *paths.Paths, assets fs.FS, upstream http.Handler, trust *proxyheader.Trust, sidecar, secret, timezone string, log *slog.Logger) (*pageStack, error) {
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
	api := webapi.Deps{DB: conn, Trust: trust, Engine: engine, Bundle: bundle, Icons: icons, Log: log}

	profiles := userpage.Deps{
		DB: conn, Engine: engine, Bundle: bundle, Icons: icons,
		Assets: static.NewAssets(assets), TimeZone: location, Files: p.Files(), Log: log,
	}

	stack := &pageStack{
		articles:    pages,
		code:        localitem.NewCode(items),
		html:        localitem.NewHTML(items),
		theme:       localitem.NewTheme(items),
		moduleAPI:   webapi.New(api, upstream),
		preview:     webapi.NewPreview(api),
		articleAPI:  webapi.NewArticles(api, upstream),
		profile:     userpage.New(profiles),
		profileForm: userpage.NewEdit(profiles),
		close:       closeEngine,
	}

	// Without the key nothing can be verified, so every visitor stays
	// anonymous rather than being trusted on the cookie alone.
	if secret == "" {
		return stack, nil
	}
	resolver := auth.NewResolver(session.New(secret), conn, conn, log)
	stack.articleAPI = resolver.Middleware(stack.articleAPI)
	stack.code = resolver.Middleware(stack.code)
	stack.html = resolver.Middleware(stack.html)
	stack.theme = resolver.Middleware(stack.theme)
	stack.moduleAPI = resolver.Middleware(stack.moduleAPI)
	stack.preview = resolver.Middleware(stack.preview)
	stack.profile = resolver.Middleware(stack.profile)
	stack.profileForm = resolver.Middleware(stack.profileForm)
	stack.articles = resolver.Middleware(stack.articles)
	return stack, nil
}
