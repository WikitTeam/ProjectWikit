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
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/admin"
	"github.com/WikitTeam/ProjectWikit/internal/compress"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/entry"
	"github.com/WikitTeam/ProjectWikit/internal/localitem"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/migrate"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/respheader"
	"github.com/WikitTeam/ProjectWikit/internal/routing"
	"github.com/WikitTeam/ProjectWikit/internal/seed"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/userpage"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
)

const (
	envDatabase     = "DATABASE_URL"
	envSecretKey    = "SECRET_KEY"
	envTimeZone     = "PWIKIT_TIMEZONE"
	envGoogleTag    = "GOOGLE_TAG_ID"
	envUploadLimit  = "MEDIA_UPLOAD_LIMIT"
	envMailHost     = "EMAIL_HOST"
	envMailPort     = "EMAIL_PORT"
	envMailUser     = "EMAIL_USERNAME"
	envMailPassword = "EMAIL_PASSWORD"
	envMailTLS      = "EMAIL_USE_TLS"
	envMailImplicit = "EMAIL_IMPLICIT_TLS"
	envMailEngine   = "EMAIL_ENGINE"
	envMailFrom     = "EMAIL_DEFAULT_FROM"
	envStorageLimit = "ABSOLUTE_MEDIA_UPLOAD_LIMIT"
	envTLS          = "PWIKIT_TLS"
	envTLSCert      = "PWIKIT_TLS_CERT"
	envTLSKey       = "PWIKIT_TLS_KEY"
	envTLSListen    = "PWIKIT_TLS_LISTEN"
	envACMEEmail    = "PWIKIT_ACME_EMAIL"
	envACMEDir      = "PWIKIT_ACME_DIRECTORY"
	defaultListen   = "127.0.0.1:8080"
	defaultTLSPlain = ":80"
	defaultTLSAddr  = ":443"
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
	case "render":
		return render(args[1:])
	case "migrate":
		return migrateCommand(args[1:])
	case "createsite":
		return createSite(args[1:])
	case "seed":
		return seedPages(args[1:])
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
  serve       start the HTTP server
  createsite  create the site this database serves
  seed        write the pages a new site starts with
  render      render wikitext read from stdin or a file
  migrate     apply or inspect the schema migrations
  modules     print the wikidot module list
  help        show this help
`)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := fs.String("listen", defaultListen, "listen address")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	trusted := fs.String("trusted-proxies", "", "trusted reverse proxy addresses or CIDRs, comma separated; empty trusts no X-Forwarded-* header")
	staticDir := fs.String("static-dir", "", "directory holding the frontend asset bundle")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	secret := fs.String("secret-key", os.Getenv(envSecretKey), "key the session cookie is signed with; without it every visitor is anonymous")
	sidecar := fs.String("sidecar", os.Getenv(envSidecar), "path to the ftml sidecar binary; without it the linked-in ftml is used")
	timezone := fs.String("timezone", envOr(envTimeZone, "UTC"), "time zone dates are shown in")
	noMigrate := fs.Bool("no-migrate", false, "start without applying pending schema migrations")
	uploadLimit := fs.String("upload-limit", envOr(envUploadLimit, "0"), "size the files still attached to pages may reach, such as 4GB; 0 for no ceiling")
	storageLimit := fs.String("storage-limit", envOr(envStorageLimit, "0"), "size every file on disk may reach, deleted ones counted; 0 for no ceiling")
	tlsMode := fs.String("tls", envOr(envTLS, string(entry.Off)), "off to serve plain HTTP behind a proxy, file to use a supplied certificate, auto to obtain one over ACME")
	tlsListen := fs.String("tls-listen", envOr(envTLSListen, defaultTLSAddr), "listen address for HTTPS")
	tlsCert := fs.String("tls-cert", os.Getenv(envTLSCert), "certificate chain in PEM form, for -tls=file")
	tlsKey := fs.String("tls-key", os.Getenv(envTLSKey), "private key in PEM form, for -tls=file")
	acmeEmail := fs.String("acme-email", os.Getenv(envACMEEmail), "address the certificate authority sends expiry warnings to")
	acmeDirectory := fs.String("acme-directory", os.Getenv(envACMEDir), "ACME directory URL; empty uses Let's Encrypt")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *database == "" {
		return errors.New("serve needs -database or " + envDatabase)
	}

	mode, err := entry.ParseMode(*tlsMode)
	if err != nil {
		return err
	}
	if mode != entry.Off && !given(fs, "listen") {
		*listen = defaultTLSPlain
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

	assets, err := assetFS(*staticDir)
	if err != nil {
		return err
	}

	if !*noMigrate {
		result, err := migrate.Run(context.Background(), *database)
		if err != nil {
			return err
		}
		if result.Adopted {
			log.Info("pwikit adopted the schema", "baseline", migrate.BaselineName)
		}
		for _, name := range result.Applied {
			log.Info("pwikit applied a migration", "name", name)
		}
	}
	conn, err := db.Open(context.Background(), *database)
	if err != nil {
		return err
	}
	defer conn.Close()

	notFound := http.NotFoundHandler()
	mediaHandler := site.NewHostRules(conn, listenPort(*listen), media.New(p.Files(), conn), notFound)
	resizedHandler := site.NewHostRules(conn, listenPort(*listen), media.NewResized(p.Files(), conn), notFound)

	soft, err := parseSize(*uploadLimit)
	if err != nil {
		return fmt.Errorf("parse -upload-limit: %w", err)
	}
	hard, err := parseSize(*storageLimit)
	if err != nil {
		return fmt.Errorf("parse -storage-limit: %w", err)
	}

	stack, err := newPageStack(conn, p, assets, notFound, trust, limits{soft: soft, hard: hard},
		*sidecar, *secret, *timezone, log)
	if err != nil {
		return err
	}
	defer stack.close()
	served := func(h http.Handler) http.Handler {
		return compress.New(respheader.VaryCookie(site.NewHostRules(conn, listenPort(*listen), h, notFound)))
	}
	articles := served(stack.articles)
	codeHandler := served(stack.code)
	htmlHandler := served(stack.html)
	themeHandler := served(stack.theme)
	moduleAPI := served(stack.moduleAPI)
	preview := served(stack.preview)
	profile := served(stack.profile)
	profileForm := served(stack.profileForm)
	reactivePages := served(stack.reactivePages)
	notifyAPI := served(stack.notifyAPI)
	subscribeAPI := served(stack.subscribeAPI)
	messageAPI := served(stack.messageAPI)
	userAPI := served(stack.userAPI)
	adminAPI := served(stack.adminAPI)
	login := served(stack.login)
	logout := served(stack.logout)
	signup := served(stack.signup)
	accept := served(stack.accept)
	reset := served(stack.reset)
	tickets := served(stack.tickets)
	emailLinks := served(stack.emailLinks)
	settings := served(stack.settings)
	adminPages := served(stack.adminPages)
	favesAPI := served(stack.favesAPI)
	ownRowsAPI := served(stack.ownRowsAPI)
	articleAPI := served(stack.articleAPI)
	allArticles := served(stack.allArticles)
	fileAPI := served(stack.fileAPI)

	// A system path nobody claims is a mistyped URL, not a page name, so these
	// two answer before the article handler sees them.
	goHandlers := map[string]http.Handler{
		"/-/":                           notFound,
		"/pw-api/":                      notFound,
		static.Prefix:                   static.New(assets, notFound),
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
		account.EmailPrefix:             emailLinks,
		account.SettingsPrefix:          settings,
		admin.Bare:                      adminPages,
		admin.Prefix:                    adminPages,
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

	mux, err := routing.New(goHandlers)
	if err != nil {
		return err
	}

	log.Info("pwikit serve", "listen", *listen, "root", p.Root(),
		"root_source", string(p.Source()), "static_dir", *staticDir)

	var hosts entry.Hosts
	if conn != nil {
		hosts = func(ctx context.Context, host string) error {
			known, err := conn.SiteHostExists(ctx, host)
			if err != nil {
				return err
			}
			if !known {
				return fmt.Errorf("host %q is not a site on this server", host)
			}
			return nil
		}
	} else if mode == entry.Auto {
		return errors.New("-tls=auto needs -database to know which hosts to obtain certificates for")
	}

	return entry.Serve(context.Background(), entry.Config{
		Mode:      mode,
		Plain:     *listen,
		Secure:    *tlsListen,
		CertFile:  *tlsCert,
		KeyFile:   *tlsKey,
		CacheDir:  p.Certs(),
		Email:     *acmeEmail,
		Directory: *acmeDirectory,
		Hosts:     hosts,
		Handler:   respheader.OriginPolicy(mux),
		Logger:    log,
	})
}

func given(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func listenPort(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return port
}

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

func seedPages(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	slug := fs.String("site", "", "slug of the site to write into; needed once a database holds more than one")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *database == "" {
		return errors.New("no database, pass -database or set " + envDatabase)
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, *database)
	if err != nil {
		return err
	}
	defer conn.Close()

	current, err := resolveSite(ctx, conn, *slug)
	if err != nil {
		return err
	}

	written, err := seed.Run(ctx, conn, current.ID)
	for _, name := range written {
		fmt.Println("wrote " + name)
	}
	if err != nil {
		return err
	}
	fmt.Printf("%d of %d pages written, the rest were already there\n", len(written), len(seed.Names()))
	return nil
}

var slugPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func createSite(args []string) error {
	fs := flag.NewFlagSet("createsite", flag.ContinueOnError)
	slug := fs.String("slug", "", "short name for the site, letters, digits, - and _")
	domain := fs.String("domain", "", "domain the pages are served on")
	mediaDomain := fs.String("media-domain", "", "domain the uploaded files are served on; defaults to -domain")
	title := fs.String("title", "", "site title")
	headline := fs.String("headline", "", "site subtitle")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	for name, value := range map[string]string{"slug": *slug, "domain": *domain, "title": *title, "headline": *headline} {
		if value == "" {
			return fmt.Errorf("no -%s", name)
		}
	}
	if !slugPattern.MatchString(*slug) {
		return fmt.Errorf("slug %q may only hold letters, digits, - and _", *slug)
	}
	if *mediaDomain == "" {
		mediaDomain = domain
	}
	if *database == "" {
		return errors.New("no database, pass -database or set " + envDatabase)
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, *database)
	if err != nil {
		return err
	}
	defer conn.Close()

	id, err := conn.CreateSite(ctx, db.NewSite{
		Slug: *slug, Title: *title, Headline: *headline,
		Domain: *domain, MediaDomain: *mediaDomain,
	})
	if err != nil {
		return err
	}
	fmt.Printf("created site %s (%d) on %s\n", *slug, id, *domain)
	return nil
}

func migrateCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	if sub != "status" && sub != "up" {
		fmt.Fprint(os.Stderr, `Usage: pwikit migrate <status|up> [-database <url>]

  status  print which schema migrations the database carries
  up      apply the migrations the database is missing
`)
		return errors.New("unknown migrate subcommand")
	}
	fs := flag.NewFlagSet("migrate "+sub, flag.ContinueOnError)
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *database == "" {
		return errors.New("no database, pass -database or set " + envDatabase)
	}
	if sub == "up" {
		return migrateUp(*database)
	}

	state, err := migrate.Status(context.Background(), *database)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tMIGRATION")
	for _, name := range state.Applied {
		fmt.Fprintf(w, "applied\t%s\n", name)
	}
	for _, name := range state.Unknown {
		fmt.Fprintf(w, "unknown\t%s\n", name)
	}
	for _, name := range state.Pending {
		status := "pending"
		if state.Adoptable && name == migrate.BaselineName {
			status = "existing"
		}
		fmt.Fprintf(w, "%s\t%s\n", status, name)
	}
	return w.Flush()
}

func migrateUp(dsn string) error {
	result, err := migrate.Run(context.Background(), dsn)
	if err != nil {
		return err
	}
	if result.Adopted {
		fmt.Printf("adopted %s\n", migrate.BaselineName)
	}
	for _, name := range result.Applied {
		fmt.Printf("applied %s\n", name)
	}
	if !result.Adopted && len(result.Applied) == 0 {
		fmt.Println("already up to date")
	}
	return nil
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

func resolveSite(ctx context.Context, conn *db.DB, slug string) (*db.Site, error) {
	if slug != "" {
		found, err := conn.SiteBySlug(ctx, slug)
		if errors.Is(err, db.ErrNotFound) {
			return nil, fmt.Errorf("no site with slug %q", slug)
		}
		return found, err
	}
	slugs, err := conn.SiteSlugs(ctx)
	if err != nil {
		return nil, err
	}
	switch len(slugs) {
	case 0:
		return nil, errors.New("this database holds no site, make one with createsite")
	case 1:
		return conn.SiteBySlug(ctx, slugs[0])
	}
	return nil, fmt.Errorf("this database holds %d sites, name one with -site", len(slugs))
}
