package proxyheader

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func trust(t *testing.T, sources ...string) *Trust {
	t.Helper()
	tr, err := NewTrust(sources)
	if err != nil {
		t.Fatalf("NewTrust(%v) err = %v，期望 nil", sources, err)
	}
	return tr
}

func request(remoteAddr string, header http.Header) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	if header != nil {
		r.Header = header
	}
	return r
}

func headers(pairs ...string) http.Header {
	h := http.Header{}
	for i := 0; i+1 < len(pairs); i += 2 {
		h.Add(pairs[i], pairs[i+1])
	}
	return h
}

func TestNewTrustRejectsBadSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"不是地址", "nonsense"},
		{"网段位数越界", "10.0.0.0/33"},
		{"带端口", "10.0.0.1:80"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewTrust([]string{tt.source}); err == nil {
				t.Errorf("NewTrust(%q) err = nil，期望非 nil", tt.source)
			}
		})
	}
}

func TestNewTrustSkipsEmptySource(t *testing.T) {
	tr := trust(t, "", "  ", "10.0.0.0/8")
	if len(tr.nets) != 1 {
		t.Errorf("len(nets) = %d，期望 1", len(tr.nets))
	}
}

func TestTrustedMatchesBareAddress(t *testing.T) {
	tr := trust(t, "10.0.0.1")
	tests := []struct {
		addr string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.0.0.2", false},
		{"::ffff:10.0.0.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			a, ok := PeerAddr(tt.addr)
			if !ok {
				t.Fatalf("PeerAddr(%q) ok = false，期望 true", tt.addr)
			}
			if got := tr.Trusted(a); got != tt.want {
				t.Errorf("Trusted(%q) = %v，期望 %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestClientIPIgnoresHeaderFromUntrustedPeer(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("203.0.113.9:1234", headers(headerFor, "1.2.3.4"))

	got, ok := tr.ClientIP(r)
	if !ok {
		t.Fatalf("ClientIP() ok = false，期望 true")
	}
	if got.String() != "203.0.113.9" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "203.0.113.9")
	}
}

func TestClientIPUsesHeaderFromTrustedPeer(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "1.2.3.4")
	}
}

func TestClientIPSkipsTrustedHops(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4, 10.0.0.9, 10.0.0.8"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "1.2.3.4")
	}
}

func TestClientIPReadsRepeatedHeaderLines(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4", headerFor, "10.0.0.9"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "1.2.3.4")
	}
}

func TestClientIPFallsBackToLeftmostWhenAllHopsTrusted(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "10.0.0.7, 10.0.0.9"))

	got, _ := tr.ClientIP(r)
	if got.String() != "10.0.0.7" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "10.0.0.7")
	}
}

func TestClientIPSkipsUnparsableHops(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "unknown, 1.2.3.4, 10.0.0.9"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "1.2.3.4")
	}
}

func TestClientIPWithEmptyTrustAlwaysUsesPeer(t *testing.T) {
	tr := trust(t)
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4"))

	got, _ := tr.ClientIP(r)
	if got.String() != "10.0.0.1" {
		t.Errorf("ClientIP() = %q，期望 %q", got, "10.0.0.1")
	}
}

func TestClientIPReportsUnparsablePeer(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	if _, ok := tr.ClientIP(request("@", nil)); ok {
		t.Error("ClientIP() ok = true，期望 false")
	}
}

func TestScheme(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		proto      string
		tls        bool
		want       string
	}{
		{"无头且非 TLS", "10.0.0.1:1", "", false, "http"},
		{"无头且 TLS", "10.0.0.1:1", "", true, "https"},
		{"可信来源声称 https", "10.0.0.1:1", "https", false, "https"},
		{"不可信来源声称 https", "203.0.113.9:1", "https", false, "http"},
		{"可信来源声称 http 而连接是 TLS", "10.0.0.1:1", "http", true, "http"},
		{"取逗号分隔的第一个", "10.0.0.1:1", "https, http", false, "https"},
		{"大小写不敏感", "10.0.0.1:1", "HTTPS", false, "https"},
		{"取值非法时回退", "10.0.0.1:1", "gopher", false, "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := trust(t, "10.0.0.0/8")
			var h http.Header
			if tt.proto != "" {
				h = headers(headerProto, tt.proto)
			}
			r := request(tt.remoteAddr, h)
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			if got := tr.Scheme(r); got != tt.want {
				t.Errorf("Scheme() = %q，期望 %q", got, tt.want)
			}
		})
	}
}
