package forward

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
)

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newProxy(t *testing.T, upstream *httptest.Server) *Proxy {
	t.Helper()
	p, err := New(upstream.URL, nil, quietLog())
	if err != nil {
		t.Fatalf("New(%q) err = %v，期望 nil", upstream.URL, err)
	}
	return p
}

func TestNewRejectsBadTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"空串", ""},
		{"缺 scheme", "127.0.0.1:8000"},
		{"scheme 不是 http", "ftp://127.0.0.1:8000"},
		{"缺 host", "http://"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.target, nil, quietLog()); err == nil {
				t.Errorf("New(%q) err = nil，期望非 nil", tt.target)
			}
		})
	}
}

func TestProxyPreservesHost(t *testing.T) {
	var got string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Host
	}))
	defer up.Close()

	front := httptest.NewServer(newProxy(t, up))
	defer front.Close()

	req, _ := http.NewRequest(http.MethodGet, front.URL+"/scp-173", nil)
	req.Host = "scpfoundation.example"
	resp, err := front.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() err = %v，期望 nil", err)
	}
	resp.Body.Close()

	if got != "scpfoundation.example" {
		t.Errorf("上游收到的 Host = %q，期望 %q", got, "scpfoundation.example")
	}
}

func TestProxyDropsClientForwardedHeaders(t *testing.T) {
	var xff, proto string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff = r.Header.Get("X-Forwarded-For")
		proto = r.Header.Get("X-Forwarded-Proto")
	}))
	defer up.Close()

	front := httptest.NewServer(newProxy(t, up))
	defer front.Close()

	req, _ := http.NewRequest(http.MethodGet, front.URL+"/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := front.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() err = %v，期望 nil", err)
	}
	resp.Body.Close()

	if strings.Contains(xff, "1.2.3.4") {
		t.Errorf("上游收到的 X-Forwarded-For = %q，期望不含客户端伪造的 1.2.3.4", xff)
	}
	if proto != "http" {
		t.Errorf("上游收到的 X-Forwarded-Proto = %q，期望 %q", proto, "http")
	}
}

func TestProxyPassesRequestURIThrough(t *testing.T) {
	var got string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RequestURI
	}))
	defer up.Close()

	front := httptest.NewServer(newProxy(t, up))
	defer front.Close()

	resp, err := front.Client().Get(front.URL + "/local--files/a%2Fb.png?size=200")
	if err != nil {
		t.Fatalf("Get() err = %v，期望 nil", err)
	}
	resp.Body.Close()

	if got != "/local--files/a%2Fb.png?size=200" {
		t.Errorf("上游收到的 RequestURI = %q，期望 %q", got, "/local--files/a%2Fb.png?size=200")
	}
}

func TestProxyPassesStatusAndBody(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Wikit", "1")
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, "内容")
	}))
	defer up.Close()

	front := httptest.NewServer(newProxy(t, up))
	defer front.Close()

	resp, err := front.Client().Get(front.URL + "/")
	if err != nil {
		t.Fatalf("Get() err = %v，期望 nil", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("StatusCode = %d，期望 %d", resp.StatusCode, http.StatusTeapot)
	}
	if resp.Header.Get("X-Wikit") != "1" {
		t.Errorf("X-Wikit = %q，期望 %q", resp.Header.Get("X-Wikit"), "1")
	}
	if string(body) != "内容" {
		t.Errorf("body = %q，期望 %q", body, "内容")
	}
}

func TestProxyReturns502WhenUpstreamDown(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	dead := newProxy(t, up)
	up.Close()

	front := httptest.NewServer(dead)
	defer front.Close()

	resp, err := front.Client().Get(front.URL + "/")
	if err != nil {
		t.Fatalf("Get() err = %v，期望 nil", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d，期望 %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestProxyHonorsForwardedForFromTrustedPeer(t *testing.T) {
	var xff string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff = r.Header.Get("X-Forwarded-For")
	}))
	defer up.Close()

	trust, err := proxyheader.NewTrust([]string{"127.0.0.0/8", "::1/128"})
	if err != nil {
		t.Fatalf("NewTrust() err = %v，期望 nil", err)
	}
	p, err := New(up.URL, trust, quietLog())
	if err != nil {
		t.Fatalf("New() err = %v，期望 nil", err)
	}

	front := httptest.NewServer(p)
	defer front.Close()

	req, _ := http.NewRequest(http.MethodGet, front.URL+"/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	resp, err := front.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() err = %v，期望 nil", err)
	}
	resp.Body.Close()

	if xff != "1.2.3.4" {
		t.Errorf("上游收到的 X-Forwarded-For = %q，期望 %q", xff, "1.2.3.4")
	}
}

func TestProxySetsForwardedHostToInboundHost(t *testing.T) {
	var got string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Forwarded-Host")
	}))
	defer up.Close()

	front := httptest.NewServer(newProxy(t, up))
	defer front.Close()

	req, _ := http.NewRequest(http.MethodGet, front.URL+"/", nil)
	req.Host = "media.example"
	resp, err := front.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() err = %v，期望 nil", err)
	}
	resp.Body.Close()

	if got != "media.example" {
		t.Errorf("上游收到的 X-Forwarded-Host = %q，期望 %q", got, "media.example")
	}
}
