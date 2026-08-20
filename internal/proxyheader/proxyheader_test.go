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
		t.Fatalf("NewTrust(%v) err = %v, want nil", sources, err)
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
		{"not an address", "nonsense"},
		{"prefix length out of range", "10.0.0.0/33"},
		{"with port", "10.0.0.1:80"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewTrust([]string{tt.source}); err == nil {
				t.Errorf("NewTrust(%q) err = nil, want non-nil", tt.source)
			}
		})
	}
}

func TestNewTrustSkipsEmptySource(t *testing.T) {
	tr := trust(t, "", "  ", "10.0.0.0/8")
	if len(tr.nets) != 1 {
		t.Errorf("len(nets) = %d, want 1", len(tr.nets))
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
				t.Fatalf("PeerAddr(%q) ok = false, want true", tt.addr)
			}
			if got := tr.Trusted(a); got != tt.want {
				t.Errorf("Trusted(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestClientIPIgnoresHeaderFromUntrustedPeer(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("203.0.113.9:1234", headers(headerFor, "1.2.3.4"))

	got, ok := tr.ClientIP(r)
	if !ok {
		t.Fatalf("ClientIP() ok = false, want true")
	}
	if got.String() != "203.0.113.9" {
		t.Errorf("ClientIP() = %q, want %q", got, "203.0.113.9")
	}
}

func TestClientIPUsesHeaderFromTrustedPeer(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIPSkipsTrustedHops(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4, 10.0.0.9, 10.0.0.8"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIPReadsRepeatedHeaderLines(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4", headerFor, "10.0.0.9"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIPFallsBackToLeftmostWhenAllHopsTrusted(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "10.0.0.7, 10.0.0.9"))

	got, _ := tr.ClientIP(r)
	if got.String() != "10.0.0.7" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.7")
	}
}

func TestClientIPSkipsUnparsableHops(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	r := request("10.0.0.1:1234", headers(headerFor, "unknown, 1.2.3.4, 10.0.0.9"))

	got, _ := tr.ClientIP(r)
	if got.String() != "1.2.3.4" {
		t.Errorf("ClientIP() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIPWithEmptyTrustAlwaysUsesPeer(t *testing.T) {
	tr := trust(t)
	r := request("10.0.0.1:1234", headers(headerFor, "1.2.3.4"))

	got, _ := tr.ClientIP(r)
	if got.String() != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestClientIPReportsUnparsablePeer(t *testing.T) {
	tr := trust(t, "10.0.0.0/8")
	if _, ok := tr.ClientIP(request("@", nil)); ok {
		t.Error("ClientIP() ok = true, want false")
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
		{"no header without TLS", "10.0.0.1:1", "", false, "http"},
		{"no header over TLS", "10.0.0.1:1", "", true, "https"},
		{"trusted peer claims https", "10.0.0.1:1", "https", false, "https"},
		{"untrusted peer claims https", "203.0.113.9:1", "https", false, "http"},
		{"trusted peer claims http over TLS", "10.0.0.1:1", "http", true, "http"},
		{"takes the first comma separated value", "10.0.0.1:1", "https, http", false, "https"},
		{"case insensitive", "10.0.0.1:1", "HTTPS", false, "https"},
		{"falls back on an invalid value", "10.0.0.1:1", "gopher", false, "http"},
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
				t.Errorf("Scheme() = %q, want %q", got, tt.want)
			}
		})
	}
}
