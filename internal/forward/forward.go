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
		return nil, fmt.Errorf("parse upstream %q: %w", target, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("upstream %q scheme = %q, want http or https", target, u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("upstream %q has no host", target)
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
	p.log.Error("forward failed", "path", r.URL.Path, "upstream", p.target.String(), "err", err)
	// The visitor gets the bare status phrase: the upstream address is
	// internal, and prose here would have to go through i18n in a package
	// that ships no user-facing text of its own.
	http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
}
