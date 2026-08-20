package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/forward"
	"github.com/WikitTeam/ProjectWikit/internal/modules"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/routing"
)

const (
	envUpstream     = "PWIKIT_UPSTREAM"
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
	mux, err := routing.New(routing.Table, proxy, nil)
	if err != nil {
		return err
	}

	log.Info("pwikit serve", "listen", *listen, "upstream", proxy.Target(), "root", p.Root(), "root_source", string(p.Source()))

	srv := &http.Server{
		Addr:              *listen,
		Handler:           mux,
		ReadHeaderTimeout: 20 * time.Second,
	}
	return srv.ListenAndServe()
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
	for _, info := range modules.All() {
		status := "pending"
		switch {
		case info.Removed:
			status = "removed"
		case info.Ported:
			status = "ported"
		}
		fmt.Fprintf(w, "%s\t%t\t%s\n", info.Name, info.HasContent, status)
	}
	return w.Flush()
}
