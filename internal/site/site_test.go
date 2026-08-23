package site

import (
	"net/url"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/routing"
)

var split = Site{Domain: "wiki.example", MediaDomain: "media.example"}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%q) err = %v, want nil", raw, err)
	}
	return u
}

func TestIsMediaPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/local--files/a.png", true},
		{"/local--code/12/0", true},
		{"/local--html/12/abc", true},
		{"/local--theme/12/style.css", true},
		{"/scp-173", false},
		{"/", false},
		{"/local--files", false},
		{"/x/local--files/a.png", false},
		{"/local--filesx/a.png", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsMediaPath(tt.path); got != tt.want {
				t.Errorf("IsMediaPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestStripPort(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"example.org", "example.org"},
		{"example.org:8000", "example.org"},
		{"[::1]:8000", "::1"},
		{"[::1]", "::1"},
		{"::1", "::1"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := StripPort(tt.in); got != tt.want {
				t.Errorf("StripPort(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLookupHosts(t *testing.T) {
	tests := []struct {
		name       string
		rawHost    string
		serverPort string
		want       []string
	}{
		{"host without a port tries host:port first", "example.org", "8000", []string{"example.org:8000", "example.org"}},
		{"host with a port tries only the portless lookup", "example.org:8000", "8000", []string{"example.org"}},
		{"no port information", "example.org", "", []string{"example.org"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupHosts(tt.rawHost, tt.serverPort)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("LookupHosts(%q, %q) = %v, want %v", tt.rawHost, tt.serverPort, got, tt.want)
			}
		})
	}
}

func TestDecideServesWhenDomainsAreSame(t *testing.T) {
	s := Site{Domain: "wiki.example", MediaDomain: "wiki.example"}
	got := Decide(s, "wiki.example", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v, want Serve", got.Action)
	}
}

func TestDecideServesWhenMediaDomainEmpty(t *testing.T) {
	s := Site{Domain: "wiki.example"}
	got := Decide(s, "wiki.example", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v, want Serve", got.Action)
	}
}

func TestDecideRedirects(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		target   string
		wantLoc  string
		wantKind Action
	}{
		{"media path on the main domain", "wiki.example", "/local--files/a.png", "//media.example/local--files/a.png", Redirect},
		{"non-media path on the media domain", "media.example", "/scp-173", "//wiki.example/scp-173", Redirect},
		{"media path on the media domain", "media.example", "/local--files/a.png", "", Serve},
		{"non-media path on the main domain", "wiki.example", "/scp-173", "", Serve},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(split, tt.host, mustURL(t, tt.target))
			if got.Action != tt.wantKind {
				t.Errorf("Decide().Action = %v, want %v", got.Action, tt.wantKind)
			}
			if got.Location != tt.wantLoc {
				t.Errorf("Decide().Location = %q, want %q", got.Location, tt.wantLoc)
			}
		})
	}
}

func TestDecideKeepsQueryAndEscaping(t *testing.T) {
	got := Decide(split, "wiki.example", mustURL(t, "/local--files/a%20b.png?size=200"))
	want := "//media.example/local--files/a%20b.png?size=200"
	if got.Location != want {
		t.Errorf("Decide().Location = %q, want %q", got.Location, want)
	}
}

func TestDecideIgnoresPortInHost(t *testing.T) {
	got := Decide(split, "media.example:8000", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v, want Serve", got.Action)
	}
}

func TestMediaPrefixesAreInRouteTable(t *testing.T) {
	inTable := make(map[string]bool, len(routing.Table))
	for _, r := range routing.Table {
		inTable[r.Prefix] = true
	}
	for _, prefix := range MediaPrefixes {
		if !inTable[prefix] {
			t.Errorf("routing.Table has no media prefix %q", prefix)
		}
	}
}

func TestDecideHeaders(t *testing.T) {
	merged := Site{Domain: "wiki.example", MediaDomain: "wiki.example"}

	tests := []struct {
		name string
		site Site
		host string
		path string
		want map[string]string
	}{
		{"media host", split, "media.example", "/local--files/a.png", crossOriginHeaders},
		{"main host", split, "wiki.example", "/scp-173", sameOriginHeaders},
		{"merged domains, media path", merged, "wiki.example", "/local--files/a.png", crossOriginHeaders},
		// Merged domains make every host the media host, so nosniff and DENY
		// are never sent anywhere.
		{"merged domains, page path", merged, "wiki.example", "/scp-173", crossOriginHeaders},
		{"unknown host", split, "other.example", "/scp-173", sameOriginHeaders},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.site, tt.host, mustURL(t, tt.path))
			if got.Action != Serve {
				t.Fatalf("Decide(...).Action = %v, want Serve", got.Action)
			}
			if len(got.Headers) != len(tt.want) {
				t.Fatalf("Decide(...).Headers = %v, want %v", got.Headers, tt.want)
			}
			for k, v := range tt.want {
				if got.Headers[k] != v {
					t.Errorf("Decide(...).Headers[%q] = %q, want %q", k, got.Headers[k], v)
				}
			}
		})
	}
}

func TestDecideRedirectCarriesNoHeaders(t *testing.T) {
	for _, tt := range []struct{ host, path string }{
		{"media.example", "/scp-173"},
		{"wiki.example", "/local--files/a.png"},
	} {
		t.Run(tt.host+tt.path, func(t *testing.T) {
			got := Decide(split, tt.host, mustURL(t, tt.path))
			if got.Action != Redirect {
				t.Fatalf("Decide(...).Action = %v, want Redirect", got.Action)
			}
			if got.Headers != nil {
				t.Errorf("Decide(...).Headers = %v, want nil", got.Headers)
			}
		})
	}
}
