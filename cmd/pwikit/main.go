package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	iofs "io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/compress"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/forward"
	"github.com/WikitTeam/ProjectWikit/internal/localitem"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/respheader"
	"github.com/WikitTeam/ProjectWikit/internal/routing"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/userpage"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
)

const (
	envDatabase     = "DATABASE_URL"
	envUpstream     = "PWIKIT_UPSTREAM"
	envSecretKey    = "SECRET_KEY"
	envTimeZone     = "PWIKIT_TIMEZONE"
	envGoogleTag    = "GOOGLE_TAG_ID"
	envUploadLimit  = "MEDIA_UPLOAD_LIMIT"
	envMailHost     = "EMAIL_HOST"
	envMailPort     = "EMAIL_PORT"
	envMailUser     = "EMAIL_USERNAME"
	envMailPassword = "EMAIL_PASSWORD"
	envMailTLS      = "EMAIL_USE_TLS"
	envMailFrom     = "EMAIL_DEFAULT_FROM"
	envStorageLimit = "ABSOLUTE_MEDIA_UPLOAD_LIMIT"
	defaultUpstream = "http://127.0.0.1:8000"
	defaultListen   = "127.0.0.1:8080"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pwikit: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("missing subcommand")
	}
	switch args[0] {
	case "serve":
		return serve(args[1:])
	case "modules":
		return printModules()
	case "routes":
		return printRoutes()
	case "render":
		return render(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: pwikit <command> [options]

Commands:
  serve     start the HTTP server
  routes    print the static route table
  render    render wikitext read from stdin or a file
  modules   print the wikidot module list
  help      show this help
`)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := fs.String("listen", defaultListen, "listen address")
	upstream := fs.String("upstream", envOr(envUpstream, defaultUpstream), "upstream address for requests pwikit does not handle")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	trusted := fs.String("trusted-proxies", "", "trusted reverse proxy addresses or CIDRs, comma separated; empty trusts no X-Forwarded-* header")
	staticDir := fs.String("static-dir", "", "directory holding the frontend asset bundle; without it every asset request goes upstream")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string; without it every request needing the database goes upstream")
	secret := fs.String("secret-key", os.Getenv(envSecretKey), "key the session cookie is signed with; without it every visitor is anonymous")
	sidecar := fs.String("sidecar", os.Getenv(envSidecar), "path to the ftml sidecar binary; without it the linked-in ftml is used")
	timezone := fs.String("timezone", envOr(envTimeZone, "UTC"), "time zone dates are shown in")
	bareOrigin := fs.Bool("bare-origin", false, "drop the port from the Origin header before forwarding")
	uploadLimit := fs.String("upload-limit", envOr(envUploadLimit, "0"), "size the files still attached to pages may reach, such as 4GB; 0 for no ceiling")
	storageLimit := fs.String("storage-limit", envOr(envStorageLimit, "0"), "size every file on disk may reach, deleted ones counted; 0 for no ceiling")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	p, err := paths.New(*dataDir)
	if err != nil {
		return err
	}
	if err := p.EnsureBase(); err != nil {
		return err
	}

	log := slog.Default()

	trust, err := proxyheader.NewTrust(strings.Split(*trusted, ","))
	if err != nil {
		return err
	}

	proxy, err := forward.New(*upstream, trust, log)
	if err != nil {
		return err
	}
	proxy.BareOrigin = *bareOrigin
	assets, err := assetFS(*staticDir)
	if err != nil {
		return err
	}

	var conn *db.DB
	if *database != "" {
		conn, err = db.Open(context.Background(), *database)
		if err != nil {
			return err
		}
		defer conn.Close()
	}

	var mediaHandler http.Handler = proxy
	var resizedHandler http.Handler = proxy
	if conn != nil {
		mediaHandler = site.NewHostRules(conn, listenPort(*listen), media.New(p.Files(), conn), proxy)
		resizedHandler = site.NewHostRules(conn, listenPort(*listen), media.NewResized(p.Files(), conn), proxy)
	}

	soft, err := parseSize(*uploadLimit)
	if err != nil {
		return fmt.Errorf("parse -upload-limit: %w", err)
	}
	hard, err := parseSize(*storageLimit)
	if err != nil {
		return fmt.Errorf("parse -storage-limit: %w", err)
	}

	var articles, codeHandler, htmlHandler, themeHandler http.Handler = proxy, proxy, proxy, proxy
	var moduleAPI, preview, profile, articleAPI http.Handler = proxy, proxy, proxy, proxy
	var profileForm, fileAPI, reactivePages, notifyAPI, favesAPI, ownRowsAPI http.Handler = proxy, proxy, proxy, proxy, proxy, proxy
	var allArticles http.Handler = proxy
	var subscribeAPI, messageAPI, userAPI, adminAPI http.Handler = proxy, proxy, proxy, proxy
	var login, logout, signup, accept, reset, tickets http.Handler = proxy, proxy, proxy, proxy, proxy, proxy
	if conn != nil {
		stack, err := newPageStack(conn, p, assets, proxy, trust, limits{soft: soft, hard: hard},
			*sidecar, *secret, *timezone, log)
		if err != nil {
			return err
		}
		defer stack.close()
		served := func(h http.Handler) http.Handler {
			return compress.New(respheader.VaryCookie(site.NewHostRules(conn, listenPort(*listen), h, proxy)))
		}
		articles = served(stack.articles)
		codeHandler = served(stack.code)
		htmlHandler = served(stack.html)
		themeHandler = served(stack.theme)
		moduleAPI = served(stack.moduleAPI)
		preview = served(stack.preview)
		profile = served(stack.profile)
		profileForm = served(stack.profileForm)
		reactivePages = served(stack.reactivePages)
		notifyAPI = served(stack.notifyAPI)
		subscribeAPI = served(stack.subscribeAPI)
		messageAPI = served(stack.messageAPI)
		userAPI = served(stack.userAPI)
		adminAPI = served(stack.adminAPI)
		login = served(stack.login)
		logout = served(stack.logout)
		signup = served(stack.signup)
		accept = served(stack.accept)
		reset = served(stack.reset)
		tickets = served(stack.tickets)
		favesAPI = served(stack.favesAPI)
		ownRowsAPI = served(stack.ownRowsAPI)
		articleAPI = served(stack.articleAPI)
		allArticles = served(stack.allArticles)
		fileAPI = served(stack.fileAPI)
	}

	goHandlers := map[string]http.Handler{
		// The bundle is the one route answered above the session layer, so it
		// is also the one that does not vary on the cookie.
		static.Prefix:                   static.New(assets, proxy),
		site.ThemePrefix:                respheader.VaryCookie(site.NewThemeFiles(p.Files())),
		media.Prefix:                    respheader.VaryCookie(mediaHandler),
		media.ResizedPrefix:             respheader.VaryCookie(resizedHandler),
		localitem.CodePrefix:            codeHandler,
		localitem.HTMLPrefix:            htmlHandler,
		localitem.ThemePrefix:           themeHandler,
		webapi.ModulesPath:              moduleAPI,
		webapi.PreviewPath:              preview,
		userpage.Prefix:                 profile,
		userpage.EditPrefix:             profileForm,
		account.LoginPath:               login,
		account.LogoutPath:              logout,
		account.SignupPath:              signup,
		account.SignupPrefix:            signup,
		account.AcceptPrefix:            accept,
		account.ResetPath:               reset,
		account.ResetPrefix:             reset,
		account.ResetConfirmPath:        reset,
		account.TicketPath:              tickets,
		account.MembershipPath:          tickets,
		userpage.FavouritesPrefix:       reactivePages,
		webapi.NotificationsPath:        notifyAPI,
		webapi.SubscribePath:            subscribeAPI,
		webapi.MessagesPrefix:           messageAPI,
		webapi.UsersPath:                userAPI,
		webapi.UsersPrefix:              userAPI,
		webapi.AdminPrefix:              adminAPI,
		webapi.FavouritesPath:           favesAPI,
		webapi.RatingsPath:              ownRowsAPI,
		webapi.LikedPostsPath:           ownRowsAPI,
		userpage.RatingsPrefix:          reactivePages,
		userpage.NotificationsPrefix:    reactivePages,
		userpage.NotificationsSubPrefix: reactivePages,
		userpage.MessagesPrefix:         reactivePages,
		userpage.MessagesSubPrefix:      reactivePages,
		userpage.LikedPostsPrefix:       reactivePages,
		webapi.AllArticlesPath:          allArticles,
		webapi.ArticlesPrefix:           articleAPI,
		webapi.FilesPrefix:              fileAPI,
		"/":                             articles,
	}

	mux, err := routing.New(routing.Table, proxy, goHandlers)
	if err != nil {
		return err
	}

	log.Info("pwikit serve", "listen", *listen, "upstream", proxy.Target(), "root", p.Root(),
		"root_source", string(p.Source()), "static_dir", *staticDir, "database", *database != "")

	srv := &http.Server{
		Addr:              *listen,
		Handler:           respheader.OriginPolicy(mux),
		ReadHeaderTimeout: 20 * time.Second,
	}
	return srv.ListenAndServe()
}

func listenPort(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return port
}

// Until the build produces a bundle, an empty path means every asset request
// falls through to the upstream.
func assetFS(dir string) (iofs.FS, error) {
	if dir == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("static-dir %q is not a directory", dir)
	}
	return os.DirFS(dir), nil
}

var sizeUnits = map[string]int64{"B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30, "TB": 1 << 40}

func parseSize(spec string) (int64, error) {
	digits := strings.TrimLeft(spec, "0123456789")
	number, err := strconv.ParseInt(spec[:len(spec)-len(digits)], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("no leading number in %q", spec)
	}
	unit := strings.ToUpper(strings.TrimSpace(digits))
	if unit == "" {
		return number, nil
	}
	scale, ok := sizeUnits[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit %q", unit)
	}
	return number * scale, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func printRoutes() error {
	if err := routing.Validate(routing.Table); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ROUTE\tOWNER\tLABEL")
	for _, r := range routing.Table {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Prefix, r.Owner, r.Label)
	}
	return w.Flush()
}

func printModules() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tBODY\tSTATUS")
	for _, info := range module.All() {
		status := "pending"
		switch {
		case info.Removed:
			status = "removed"
		case module.Ported(info.Name):
			status = "ported"
		}
		fmt.Fprintf(w, "%s\t%t\t%s\n", info.Name, info.HasContent, status)
	}
	return w.Flush()
}
