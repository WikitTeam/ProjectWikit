package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	iofs "io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/admin"
	"github.com/WikitTeam/ProjectWikit/internal/archive"
	"github.com/WikitTeam/ProjectWikit/internal/backup"
	"github.com/WikitTeam/ProjectWikit/internal/compress"
	"github.com/WikitTeam/ProjectWikit/internal/config"
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
	"github.com/WikitTeam/ProjectWikit/internal/secretfile"
	"github.com/WikitTeam/ProjectWikit/internal/seed"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/userpage"
	"github.com/WikitTeam/ProjectWikit/internal/webapi"
	staticfiles "github.com/WikitTeam/ProjectWikit/static"
)

const (
	envDatabase     = "DATABASE_URL"
	envSecretKey    = "SECRET_KEY"
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
	sessionKeyFile  = "session-key"
)

func main() {
	if handled, err := runAsService(os.Args[1:]); handled || err != nil {
		if err != nil {
			fmt.Fprintln(os.Stderr, "pwikit: "+err.Error())
			os.Exit(1)
		}
		return
	}
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
		return serve(context.Background(), args[1:])
	case "modules":
		return printModules()
	case "render":
		return render(args[1:])
	case "migrate":
		return migrateCommand(args[1:])
	case "createsite":
		return createSite(args[1:])
	case "site":
		return siteCommand(args[1:])
	case "admin":
		return adminCommand(args[1:])
	case "backup":
		return backupCommand(args[1:])
	case "seed":
		return seedPages(args[1:])
	case "service":
		return serviceCommand(args[1:])
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
  site        list the sites in this database or point one at another domain
  admin       create an administrator or give an account every right
  backup      write, check, list or put back a backup
  seed        write the pages a new site starts with
  service     start pwikit whenever the machine boots
  render      render wikitext read from stdin or a file
  migrate     apply or inspect the schema migrations
  modules     print the wikidot module list
  help        show this help
`)
}

func serve(ctx context.Context, args []string) (err error) {
	o := newServeOptions()
	if err := o.fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	p, err := paths.New(*o.dataDir)
	if err != nil {
		return err
	}
	if err := p.EnsureBase(); err != nil {
		return err
	}
	if _, err := config.WriteTemplate(p.Config()); err != nil {
		return err
	}
	cfg, err := config.Load(p.Config())
	if err != nil {
		return err
	}
	mode, err := o.resolve(cfg)
	if err != nil {
		return err
	}
	if *o.secret == "" {
		if *o.secret, err = secretfile.Ensure(p.Secrets(), sessionKeyFile); err != nil {
			return err
		}
	}

	log, closeLog, err := openLog(*o.logFile)
	if err != nil {
		return err
	}
	defer closeLog()
	if cfg.Mail.Password != "" && config.ReadableByOthers(p.Config()) {
		log.Warn("pwikit.toml holds the mail password and other accounts on this machine can read it", "path", p.Config())
	}

	// Caught this early so a stop during a slow PostgreSQL start still stops PostgreSQL.
	ctx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	dsn := *o.database
	if dsn == "" {
		bundled, err := startBundled(ctx, p, log)
		if err != nil {
			return err
		}
		defer stopBundled(bundled)
		dsn = bundled.DSN()

		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-bundled.Exited():
				log.Error("PostgreSQL stopped while pwikit was serving")
				cancel()
			case <-ctx.Done():
			}
		}()
		defer func() {
			select {
			case <-bundled.Exited():
				if err == nil {
					err = bundled.ExitError()
				}
			default:
			}
		}()
	}

	trust, err := proxyheader.NewTrust(strings.Split(*o.trusted, ","))
	if err != nil {
		return err
	}

	assets, err := assetFS(*o.staticDir)
	if err != nil {
		return err
	}
	if assets == nil {
		log.Warn("this build carries no page assets and -static-dir is not set, so pages are served without styles and scripts")
	}

	// Asked before anything writes, so a server too old to hold the schema says
	// so instead of failing somewhere in the middle of a migration.
	found, err := backup.CheckServer(ctx, dsn)
	if err != nil {
		return err
	}
	log.Info("pwikit reached postgres", "version", backup.Describe(found))

	if !*o.noMigrate {
		result, err := migrate.Run(ctx, dsn)
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
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close()

	hostsBound, err := conn.SiteHosts(ctx)
	if err != nil {
		return err
	}
	if domain, ok := o.promote(hostsBound); ok {
		mode = entry.Auto
		log.Info("pwikit serves HTTPS because a site is bound to a public domain", "domain", domain)
	}
	if *o.dev {
		log.Info("pwikit is in development mode, and only this machine can reach it", "url", "http://"+*o.listen)
	}

	notFound := http.NotFoundHandler()
	mediaHandler := site.NewHostRules(conn, listenPort(*o.listen), media.New(p.Files(), conn), notFound)
	resizedHandler := site.NewHostRules(conn, listenPort(*o.listen), media.NewResized(p.Files(), conn), notFound)

	soft, err := parseSize(*o.uploadLimit)
	if err != nil {
		return fmt.Errorf("parse -upload-limit: %w", err)
	}
	hard, err := parseSize(*o.storageLimit)
	if err != nil {
		return fmt.Errorf("parse -storage-limit: %w", err)
	}

	stack, err := newPageStack(conn, p, assets, notFound, trust, limits{soft: soft, hard: hard},
		*o.sidecar, *o.secret, cfg, log)
	if err != nil {
		return err
	}
	defer stack.close()
	// Only the page handler answers a name a person typed, so only it explains
	// an unresolved one. The rest are reached from inside a page.
	served := func(h http.Handler) http.Handler {
		return compress.New(respheader.VaryCookie(site.NewHostRules(conn, listenPort(*o.listen), h, stack.unresolved)))
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
		site.ThemePrefix:                respheader.VaryCookie(site.NewHostRules(conn, listenPort(*o.listen), site.NewThemeFiles(p.Files()), notFound)),
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

	log.Info("pwikit serve", "listen", *o.listen, "root", p.Root(),
		"root_source", string(p.Source()), "static_dir", *o.staticDir, "assets_embedded", staticfiles.Embedded)

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

	return entry.Serve(ctx, entry.Config{
		Mode:      mode,
		Plain:     *o.listen,
		Secure:    *o.tlsListen,
		CertFile:  *o.tlsCert,
		KeyFile:   *o.tlsKey,
		CacheDir:  p.Certs(),
		Email:     *o.acmeEmail,
		Directory: *o.acmeDirectory,
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
		if staticfiles.Embedded {
			return staticfiles.Files, nil
		}
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
	archivePath := fs.String("archive", "", "wikitCLI backup to import instead of the starter pages")
	from := fs.String("from", "", "slug of the site inside the archive; needed when it holds more than one")
	forceTags := fs.Bool("force-tags", false, "create tags this site would otherwise refuse")
	noVotes := fs.Bool("no-votes", false, "leave the ratings behind")
	noFiles := fs.Bool("no-files", false, "leave the attachments behind")
	dataDir := fs.String("data-dir", "", "state directory attachments are copied into; defaults to the directory holding the executable")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	ctx := context.Background()
	dsn, release, err := resolveDatabase(ctx, *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close()

	current, err := resolveSite(ctx, conn, *slug)
	if err != nil {
		return err
	}

	if *archivePath != "" {
		files := ""
		if !*noFiles {
			p, err := paths.New(*dataDir)
			if err != nil {
				return err
			}
			if err := p.EnsureBase(); err != nil {
				return err
			}
			files = p.Files()
		}
		return importArchive(ctx, conn, current, *archivePath, *from, *forceTags, !*noVotes, files)
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
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
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
	for name, value := range map[string]string{"domain": *domain, "media-domain": *mediaDomain} {
		if !site.ValidHost(value) {
			return fmt.Errorf("-%s %q is not a host name; give the name a request arrives on, without a scheme or a path", name, value)
		}
	}
	ctx := context.Background()
	dsn, release, err := resolveDatabase(ctx, *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()
	if _, err := backup.CheckServer(ctx, dsn); err != nil {
		return err
	}
	// A database nothing has started on yet has no tables to put the site in.
	result, err := migrate.Run(ctx, dsn)
	if err != nil {
		return err
	}
	if result.Adopted {
		fmt.Printf("adopted %s\n", migrate.BaselineName)
	}
	for _, name := range result.Applied {
		fmt.Printf("applied %s\n", name)
	}
	conn, err := db.Open(ctx, dsn)
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
	announceHTTPS(*domain)
	return nil
}

func migrateCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	if sub != "status" && sub != "up" {
		fmt.Fprint(os.Stderr, `Usage: pwikit migrate <status|up> [-database <url>] [-data-dir <dir>]

  status  print which schema migrations the database carries
  up      apply the migrations the database is missing
`)
		return errors.New("unknown migrate subcommand")
	}
	fs := flag.NewFlagSet("migrate "+sub, flag.ContinueOnError)
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	ctx := context.Background()
	dsn, release, err := resolveDatabase(ctx, *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()
	if sub == "up" {
		return migrateUp(dsn)
	}

	state, err := migrate.Status(ctx, dsn)
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

func importArchive(ctx context.Context, conn *db.DB, current *db.Site, path, from string,
	forceTags, votes bool, files string) error {

	found, err := archive.Open(path)
	if err != nil {
		return err
	}
	defer found.Close()

	slugs, err := found.Sites()
	if err != nil {
		return err
	}
	switch {
	case len(slugs) == 0:
		return fmt.Errorf("no site in the archive at %q", path)
	case from == "" && len(slugs) > 1:
		return fmt.Errorf("the archive holds %d sites, name one with -from", len(slugs))
	case from == "":
		from = slugs[0]
	case !slices.Contains(slugs, from):
		return fmt.Errorf("the archive has no site %q", from)
	}

	fmt.Printf("importing %s into %s\n", from, current.Slug)
	result, err := archive.ImportPages(ctx, conn, current.ID, found, from, archive.Options{
		ForceTags: forceTags,
		Votes:     votes,
		Files:     files,
		Report:    func(line string) { fmt.Println(line) },
	})
	fmt.Printf("%d pages, %d already there, %d revisions, %d parents, %d files, %d accounts\n",
		result.Pages, result.Skipped, result.Revisions, result.Parents, result.Files, result.Users)
	fmt.Printf("%d forum categories, %d threads, %d posts\n",
		result.Categories, result.Threads, result.Posts)
	return err
}

func siteCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	if sub != "list" && sub != "rebind" {
		fmt.Fprint(os.Stderr, `Usage: pwikit site <list|rebind> [options]

  list    print the slug and the two domains of every site
  rebind  point a site at another domain, for when the stored one cannot be reached

Options for rebind:
  -slug          site to rebind
  -domain        domain the pages are served on
  -media-domain  domain the uploaded files are served on; defaults to -domain
`)
		return errors.New("unknown site subcommand")
	}
	fs := flag.NewFlagSet("site "+sub, flag.ContinueOnError)
	slug := fs.String("slug", "", "site to rebind")
	domain := fs.String("domain", "", "domain the pages are served on")
	mediaDomain := fs.String("media-domain", "", "domain the uploaded files are served on; defaults to -domain")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	ctx := context.Background()
	dsn, release, err := resolveDatabase(ctx, *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close()

	if sub == "list" {
		return listSites(ctx, conn)
	}
	return rebindSite(ctx, conn, *slug, *domain, *mediaDomain)
}

func listSites(ctx context.Context, conn *db.DB) error {
	slugs, err := conn.SiteSlugs(ctx)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tDOMAIN\tMEDIA DOMAIN")
	for _, slug := range slugs {
		s, err := conn.SiteBySlug(ctx, slug)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Slug, s.Domain, s.MediaDomain)
	}
	return w.Flush()
}

func rebindSite(ctx context.Context, conn *db.DB, slug, domain, mediaDomain string) error {
	if slug == "" || domain == "" {
		return errors.New("rebind needs -slug and -domain")
	}
	if mediaDomain == "" {
		mediaDomain = domain
	}
	for name, value := range map[string]string{"domain": domain, "media-domain": mediaDomain} {
		if !site.ValidHost(value) {
			return fmt.Errorf("-%s %q is not a host name; give the name a request arrives on, without a scheme or a path", name, value)
		}
	}
	before, err := conn.SiteBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("no site with slug %q", slug)
		}
		return err
	}
	if err := conn.SetSiteHosts(ctx, slug, domain, mediaDomain); err != nil {
		return err
	}
	fmt.Printf("%s: %s -> %s\n", slug, before.Domain, domain)
	fmt.Printf("%s: %s -> %s (media)\n", slug, before.MediaDomain, mediaDomain)
	announceHTTPS(domain)
	return nil
}

func announceHTTPS(domain string) {
	if site.PublicHost(domain) {
		fmt.Printf("https://%s answers once pwikit serve starts, or restarts if it is already running\n", domain)
	}
}
