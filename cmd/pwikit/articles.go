package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/admin"
	"github.com/WikitTeam/ProjectWikit/internal/articlepage"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/ipban"
	"github.com/WikitTeam/ProjectWikit/internal/lang"
	"github.com/WikitTeam/ProjectWikit/internal/localitem"
	"github.com/WikitTeam/ProjectWikit/internal/mail"
	"github.com/WikitTeam/ProjectWikit/internal/mirror"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/token"
	"github.com/WikitTeam/ProjectWikit/internal/update"
	"github.com/WikitTeam/ProjectWikit/internal/userpage"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
	"github.com/WikitTeam/ProjectWikit/internal/wikidotclient"
	"github.com/WikitTeam/ProjectWikit/internal/wikidotsrc"
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
	systemMissing http.Handler
	unresolved    http.Handler
	close         func()
}

type limits struct {
	soft int64
	hard int64
}

func newPageStack(conn *db.DB, p *paths.Paths, assets fs.FS, next http.Handler, trust *proxyheader.Trust, size limits, sidecar, secret string, cfg config.File, board *update.Board, log *slog.Logger) (*pageStack, error) {
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
	board.Bundle = bundle

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
	verifier, err := claimVerifier(conn, p, cfg.Wikidot, bundle, log)
	if err != nil {
		return nil, err
	}
	accounts := account.Deps{
		DB: conn, Sessions: store, Engine: engine, Icons: icons, Bundle: bundle,
		Tokens:   token.Generator{Secret: secret},
		Verifier: verifier,
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
		Mail: mail.New(mailConfig(cfg.Mail)), Trust: trust, Updates: board, Log: log,
		Mirrors: mirror.NewRunner(conn, wikidotsrc.New(mirrorPerSite, mirrorTotal)),
	}, next)
	if err != nil {
		return nil, err
	}
	stack.adminPages = adminPages

	resolver := auth.NewResolver(store, conn, conn, log)
	negotiate := lang.Middleware(bundle)
	banned := ipban.Middleware(ipban.Deps{DB: conn, Trust: trust, Bundle: bundle, Assets: static.NewAssets(assets), Log: log})
	resolved := func(h http.Handler) http.Handler { return resolver.Middleware(negotiate(banned(h))) }
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
	stack.systemMissing = resolved(shell.SystemNotFound(bundle, static.NewAssets(assets)))
	return stack, nil
}

func claimVerifier(conn *db.DB, p *paths.Paths, file config.Wikidot, bundle *i18n.Bundle, log *slog.Logger) (account.Verifier, error) {
	username := envOr(envWikidotUser, file.Username)
	if username == "" {
		return account.NewVerifier(), nil
	}
	password := envOr(envWikidotPassword, file.Password)
	if password == "" {
		path := filepath.Join(p.Secrets(), wikidotPasswordFile)
		raw, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		password = strings.TrimSpace(string(raw))
	}
	if password == "" {
		return nil, fmt.Errorf("wikidot.username is set in pwikit.toml but no password is: set wikidot.password, %s, or put it in %s",
			envWikidotPassword, filepath.Join(p.Secrets(), wikidotPasswordFile))
	}
	if file.MessageBody != "" && !strings.Contains(file.MessageBody, "{code}") {
		return nil, errors.New("wikidot.message_body in pwikit.toml has no {code} in it")
	}
	client := wikidotclient.New(wikidotclient.DefaultBase, username, password)
	verifier := account.NewLocalVerifier(conn, client, bundle, log)
	verifier.Subject = file.MessageSubject
	verifier.Body = file.MessageBody
	return verifier, nil
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
