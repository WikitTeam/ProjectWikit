package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/forward"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
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
  help    显示本帮助
`)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := fs.String("listen", defaultListen, "监听地址")
	upstream := fs.String("upstream", envOr(envUpstream, defaultUpstream), "转发未处理请求的上游地址")
	dataDir := fs.String("data-dir", "", "状态目录，默认取可执行文件所在目录")
	if err := fs.Parse(args); err != nil {
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

	proxy, err := forward.New(*upstream, log)
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
