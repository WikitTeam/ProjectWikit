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
		return errors.New("缺少子命令")
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
		return fmt.Errorf("未知子命令 %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `用法: pwikit <子命令> [选项]

子命令:
  serve   启动 HTTP 服务
  routes  打印静态路由表
  modules 打印 wikidot 模块清单
  help    显示本帮助
`)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := fs.String("listen", defaultListen, "监听地址")
	upstream := fs.String("upstream", envOr(envUpstream, defaultUpstream), "转发未处理请求的上游地址")
	dataDir := fs.String("data-dir", "", "状态目录，默认取可执行文件所在目录")
	trusted := fs.String("trusted-proxies", "", "可信前置代理的地址或网段，逗号分隔；留空则不信任任何 X-Forwarded-* 头")
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
