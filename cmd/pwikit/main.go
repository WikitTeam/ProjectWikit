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
	"strings"
	"text/tabwriter"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/compress"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/forward"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/respheader"
	"github.com/WikitTeam/ProjectWikit/internal/routing"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

const (
	envDatabase     = "DATABASE_URL"
	envUpstream     = "PWIKIT_UPSTREAM"
	envSecretKey    = "SECRET_KEY"
	envTimeZone     = "PWIKIT_TIMEZONE"
	envGoogleTag    = "GOOGLE_TAG_ID"
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
	if conn != nil {
		mediaHandler = site.NewHostRules(conn, listenPort(*listen), media.New(p.Files(), conn), proxy)
	}

	var articles http.Handler = proxy
	if conn != nil {
		pages, closeEngine, err := articleHandler(conn, p, assets, *sidecar, *secret, *timezone, log)
		if err != nil {
			return err
		}
		defer closeEngine()
		articles = compress.New(respheader.VaryCookie(site.NewHostRules(conn, listenPort(*listen), pages, proxy)))
	}

	goHandlers := map[string]http.Handler{
		// The bundle is the one route answered above the session layer, so it
		// is also the one that does not vary on the cookie.
		static.Prefix: static.New(assets, proxy),
		media.Prefix:  respheader.VaryCookie(mediaHandler),
		"/":           articles,
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

// listenPort feeds the host lookup its second candidate, the way Django reads
// SERVER_PORT.
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
	fmt.Fprintln(w, "PREFIX\tOWNER\tLABEL")
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
