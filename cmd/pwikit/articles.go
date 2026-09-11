package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/admin"
	"github.com/WikitTeam/ProjectWikit/internal/articlepage"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/lang"
	"github.com/WikitTeam/ProjectWikit/internal/localitem"
	"github.com/WikitTeam/ProjectWikit/internal/mail"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/site"
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
	unresolved    http.Handler
	close         func()
}

type limits struct {
	soft int64
	hard int64
}

func newPageStack(conn *db.DB, p *paths.Paths, assets fs.FS, next http.Handler, trust *proxyheader.Trust, size limits, sidecar, secret string, cfg config.File, log *slog.Logger) (*pageStack, error) {
	engine, closeEngine, err := newRenderer(sidecar)
	if err != nil {
		return nil, err
	}
	bundle, err := i18n.Load(p.Locales())
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
		GoogleTagID: envOr(envGoogleTag, cfg.Analytics.GoogleTagID),
		Log:         log,
	})
	items := localitem.Deps{DB: conn, Engine: engine, Bundle: bundle, Icons: icons, Log: log}
	api := webapi.Deps{DB: conn, Trust: trust, Engine: engine, Bundle: bundle, Icons: icons,
		Tokens: token.Generator{Secret: secret},
		Files:  p.Files(), SoftLimit: size.soft, HardLimit: size.hard, Log: log}

	profiles := userpage.Deps{
		DB: conn, Engine: engine, Bundle: bundle, Icons: icons,
		Assets: static.NewAssets(assets), Files: p.Files(), Log: log,
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
		unresolved:    site.NewUnresolved(bundle, static.NewAssets(assets)),
	}
	store := session.New(secret)
	accounts := account.Deps{
		DB: conn, Sessions: store, Engine: engine, Icons: icons, Bundle: bundle,
		Tokens:   token.Generator{Secret: secret},
		Verifier: account.NewVerifier(),
		Mail:     mail.New(mailConfig(cfg.Mail)),
		Assets:   static.NewAssets(assets), Trust: trust, Log: log,
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
		DB: conn, Bundle: bundle, Assets: static.NewAssets(assets), Files: p.Files(),
		Tokens: token.Generator{Secret: secret}, Articles: stack.articleAPI,
		Mail: mail.New(mailConfig(cfg.Mail)), Trust: trust, Log: log,
	}, next)
	if err != nil {
		return nil, err
	}
	stack.adminPages = adminPages

	resolver := auth.NewResolver(store, conn, conn, log)
	negotiate := lang.Middleware(bundle)
	resolved := func(h http.Handler) http.Handler { return resolver.Middleware(negotiate(h)) }
	stack.login = resolved(stack.login)
	stack.logout = resolved(stack.logout)
	stack.signup = resolved(stack.signup)
	stack.accept = resolved(stack.accept)
	stack.reset = resolved(stack.reset)
	stack.tickets = resolved(stack.tickets)
	stack.emailLinks = resolved(stack.emailLinks)
	stack.settings = resolved(stack.settings)
	stack.adminPages = resolved(stack.adminPages)
	stack.articleAPI = resolved(stack.articleAPI)
	stack.allArticles = resolved(stack.allArticles)
	stack.fileAPI = resolved(stack.fileAPI)
	stack.code = resolved(stack.code)
	stack.html = resolved(stack.html)
	stack.theme = resolved(stack.theme)
	stack.moduleAPI = resolved(stack.moduleAPI)
	stack.preview = resolved(stack.preview)
	stack.profile = resolved(stack.profile)
	stack.profileForm = resolved(stack.profileForm)
	stack.reactivePages = resolved(stack.reactivePages)
	stack.notifyAPI = resolved(stack.notifyAPI)
	stack.subscribeAPI = resolved(stack.subscribeAPI)
	stack.messageAPI = resolved(stack.messageAPI)
	stack.userAPI = resolved(stack.userAPI)
	stack.adminAPI = resolved(stack.adminAPI)
	stack.favesAPI = resolved(stack.favesAPI)
	stack.ownRowsAPI = resolved(stack.ownRowsAPI)
	stack.articles = resolved(stack.articles)
	return stack, nil
}

func mailConfig(file config.Mail) mail.Config {
	if envOr(envMailEngine, file.Engine) == config.EngineConsole {
		return mail.Config{}
	}
	filePort := ""
	if file.Port > 0 {
		filePort = strconv.Itoa(file.Port)
	}
	port := envOr(envMailPort, filePort)
	if port == "" {
		port = defaultMailPort
	}
	return mail.Config{
		Host:     envOr(envMailHost, file.Host),
		Port:     port,
		Username: envOr(envMailUser, file.Username),
		Password: envOr(envMailPassword, file.Password),
		UseTLS:   switchSetting(envMailTLS, file.UseTLS) || port == implicitTLSPort,
		Implicit: switchSetting(envMailImplicit, file.ImplicitTLS) || port == implicitTLSPort,
		From:     envOr(envMailFrom, file.From),
	}
}

func switchSetting(env string, file *bool) bool {
	if value := os.Getenv(env); value != "" {
		return value == "true"
	}
	return file != nil && *file
}

const (
	implicitTLSPort = "465"
	defaultMailPort = "587"
)
