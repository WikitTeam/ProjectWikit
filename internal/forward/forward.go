package forward

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
)

const (
	headerFor   = "X-Forwarded-For"
	headerHost  = "X-Forwarded-Host"
	headerProto = "X-Forwarded-Proto"
)

type Proxy struct {
	target *url.URL
	trust  *proxyheader.Trust
	rp     *httputil.ReverseProxy
	log    *slog.Logger
}

var _ http.Handler = (*Proxy)(nil)

func New(target string, trust *proxyheader.Trust, log *slog.Logger) (*Proxy, error) {
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
	if trust == nil {
		trust, err = proxyheader.NewTrust(nil)
		if err != nil {
			return nil, err
		}
	}
	if log == nil {
		log = slog.Default()
	}

	p := &Proxy{target: u, trust: trust, log: log}
	p.rp = &httputil.ReverseProxy{Rewrite: p.rewrite, ErrorHandler: p.handleError}
	return p, nil
}

func (p *Proxy) Target() string { return p.target.String() }

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rp.ServeHTTP(w, r)
}

func (p *Proxy) rewrite(pr *httputil.ProxyRequest) {
	client, ok := p.trust.ClientIP(pr.In)
	scheme := p.trust.Scheme(pr.In)

	pr.Out.Header.Del(headerFor)
	pr.Out.Header.Del(headerHost)
	pr.Out.Header.Del(headerProto)
	if ok {
		pr.Out.Header.Set(headerFor, client.String())
	}
	pr.Out.Header.Set(headerHost, pr.In.Host)
	pr.Out.Header.Set(headerProto, scheme)

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
