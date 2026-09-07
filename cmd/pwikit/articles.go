package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/admin"
	"github.com/WikitTeam/ProjectWikit/internal/articlepage"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/localitem"
	"github.com/WikitTeam/ProjectWikit/internal/mail"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/token"
	"github.com/WikitTeam/ProjectWikit/internal/userpage"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
)

type pageStack struct {
	articles      http.Handler
	code          http.Handler
	html          http.Handler
	theme         http.Handler
	moduleAPI     http.Handler
	preview       http.Handler
	profile       http.Handler
	profileForm   http.Handler
	reactivePages http.Handler
	notifyAPI     http.Handler
	subscribeAPI  http.Handler
	messageAPI    http.Handler
	userAPI       http.Handler
	adminAPI      http.Handler
	login         http.Handler
	logout        http.Handler
	signup        http.Handler
	accept        http.Handler
	reset         http.Handler
	tickets       http.Handler
	emailLinks    http.Handler
	settings      http.Handler
	adminPages    http.Handler
	allArticles   http.Handler
	favesAPI      http.Handler
	ownRowsAPI    http.Handler
	articleAPI    http.Handler
	fileAPI       http.Handler
	close         func()
}

type limits struct {
	soft int64
	hard int64
}

func newPageStack(conn *db.DB, p *paths.Paths, assets fs.FS, next http.Handler, trust *proxyheader.Trust, size limits, sidecar, secret, timezone string, log *slog.Logger) (*pageStack, error) {
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
	api := webapi.Deps{DB: conn, Trust: trust, Engine: engine, Bundle: bundle, Icons: icons,
		Tokens: token.Generator{Secret: secret},
		Files:  p.Files(), SoftLimit: size.soft, HardLimit: size.hard, Log: log}

	profiles := userpage.Deps{
		DB: conn, Engine: engine, Bundle: bundle, Icons: icons,
		Assets: static.NewAssets(assets), TimeZone: location, Files: p.Files(), Log: log,
	}

	stack := &pageStack{
		articles:      pages,
		code:          localitem.NewCode(items),
		html:          localitem.NewHTML(items),
		theme:         localitem.NewTheme(items),
		moduleAPI:     webapi.New(api, next),
		preview:       webapi.NewPreview(api),
		articleAPI:    webapi.NewArticles(api, next),
		allArticles:   webapi.NewAllArticles(api, next),
		fileAPI:       webapi.NewFileItems(api, next),
		profile:       userpage.New(profiles),
		profileForm:   userpage.NewEdit(profiles),
		reactivePages: userpage.NewReactive(profiles),
		notifyAPI:     webapi.NewNotifications(api, next),
		subscribeAPI:  webapi.NewSubscriptions(api, next),
		messageAPI:    webapi.NewMessages(api, next),
		userAPI:       webapi.NewUsers(api, next),
		adminAPI:      webapi.NewAdmin(api, next),
		login:         next,
		logout:        next,
		signup:        next,
		accept:        next,
		reset:         next,
		tickets:       next,
		emailLinks:    next,
		settings:      next,
		adminPages:    next,
		favesAPI:      webapi.NewFavourites(api, next),
		ownRowsAPI:    webapi.NewOwnRows(api, next),
		close:         closeEngine,
	}

	// Without the key nothing can be verified, so every visitor stays
	if secret == "" {
		return stack, nil
	}
	store := session.New(secret)
	accounts := account.Deps{
		DB: conn, Sessions: store, Engine: engine, Icons: icons, Bundle: bundle,
		Tokens:   token.Generator{Secret: secret},
		Verifier: account.NewVerifier(),
		Mail:     mail.New(mailConfig()),
		Assets:   static.NewAssets(assets), TimeZone: location, Trust: trust, Log: log,
	}
	stack.login = account.NewLogin(accounts)
	stack.logout = account.NewLogout(accounts)
	stack.signup = account.NewSignup(accounts)
	stack.accept = account.NewAccept(accounts)
	stack.reset = account.NewReset(accounts)
	stack.tickets = account.NewTickets(accounts)
	stack.emailLinks = account.NewEmail(accounts)
	stack.settings = account.NewSettings(accounts)

	adminPages, err := admin.New(admin.Deps{
		DB: conn, Bundle: bundle, Assets: static.NewAssets(assets), Files: p.Files(), TimeZone: location,
		Tokens: token.Generator{Secret: secret}, Articles: stack.articleAPI,
		Mail: mail.New(mailConfig()), Log: log,
	}, next)
	if err != nil {
		return nil, err
	}
	stack.adminPages = adminPages

	resolver := auth.NewResolver(store, conn, conn, log)
	stack.login = resolver.Middleware(stack.login)
	stack.logout = resolver.Middleware(stack.logout)
	stack.signup = resolver.Middleware(stack.signup)
	stack.accept = resolver.Middleware(stack.accept)
	stack.reset = resolver.Middleware(stack.reset)
	stack.tickets = resolver.Middleware(stack.tickets)
	stack.emailLinks = resolver.Middleware(stack.emailLinks)
	stack.settings = resolver.Middleware(stack.settings)
	stack.adminPages = resolver.Middleware(stack.adminPages)
	stack.articleAPI = resolver.Middleware(stack.articleAPI)
	stack.allArticles = resolver.Middleware(stack.allArticles)
	stack.fileAPI = resolver.Middleware(stack.fileAPI)
	stack.code = resolver.Middleware(stack.code)
	stack.html = resolver.Middleware(stack.html)
	stack.theme = resolver.Middleware(stack.theme)
	stack.moduleAPI = resolver.Middleware(stack.moduleAPI)
	stack.preview = resolver.Middleware(stack.preview)
	stack.profile = resolver.Middleware(stack.profile)
	stack.profileForm = resolver.Middleware(stack.profileForm)
	stack.reactivePages = resolver.Middleware(stack.reactivePages)
	stack.notifyAPI = resolver.Middleware(stack.notifyAPI)
	stack.subscribeAPI = resolver.Middleware(stack.subscribeAPI)
	stack.messageAPI = resolver.Middleware(stack.messageAPI)
	stack.userAPI = resolver.Middleware(stack.userAPI)
	stack.adminAPI = resolver.Middleware(stack.adminAPI)
	stack.favesAPI = resolver.Middleware(stack.favesAPI)
	stack.ownRowsAPI = resolver.Middleware(stack.ownRowsAPI)
	stack.articles = resolver.Middleware(stack.articles)
	return stack, nil
}

func mailConfig() mail.Config {
	if os.Getenv(envMailEngine) == "console" {
		return mail.Config{}
	}
	port := envOr(envMailPort, "1025")
	return mail.Config{
		Host:     os.Getenv(envMailHost),
		Port:     port,
		Username: os.Getenv(envMailUser),
		Password: os.Getenv(envMailPassword),
		UseTLS:   os.Getenv(envMailTLS) == "true" || port == implicitTLSPort,
		Implicit: os.Getenv(envMailImplicit) == "true" || port == implicitTLSPort,
		From:     os.Getenv(envMailFrom),
	}
}

const implicitTLSPort = "465"
