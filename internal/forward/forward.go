package forward

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	target *url.URL
	rp     *httputil.ReverseProxy
	log    *slog.Logger
}

var _ http.Handler = (*Proxy)(nil)

func New(target string, log *slog.Logger) (*Proxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("上游地址 %q 解析失败: %w", target, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("上游地址 %q 的 scheme = %q，期望 http 或 https", target, u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("上游地址 %q 缺少 host", target)
	}
	if log == nil {
		log = slog.Default()
	}

	p := &Proxy{target: u, log: log}
	p.rp = &httputil.ReverseProxy{Rewrite: p.rewrite, ErrorHandler: p.handleError}
	return p, nil
}

func (p *Proxy) Target() string { return p.target.String() }

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rp.ServeHTTP(w, r)
}

func (p *Proxy) rewrite(pr *httputil.ProxyRequest) {
	pr.Out.Header.Del("X-Forwarded-For")
	pr.Out.Header.Del("X-Forwarded-Host")
	pr.Out.Header.Del("X-Forwarded-Proto")
	pr.SetXForwarded()

	pr.SetURL(p.target)
	pr.Out.Host = pr.In.Host
}

func (p *Proxy) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	p.log.Error("转发失败", "path", r.URL.Path, "upstream", p.target.String(), "err", err)
	http.Error(w, "无法连接上游 "+p.target.String(), http.StatusBadGateway)
}
