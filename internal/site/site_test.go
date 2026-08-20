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
		t.Fatalf("Parse(%q) err = %v，期望 nil", raw, err)
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
				t.Errorf("IsMediaPath(%q) = %v，期望 %v", tt.path, got, tt.want)
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
				t.Errorf("StripPort(%q) = %q，期望 %q", tt.in, got, tt.want)
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
		{"无端口时先试带端口的", "example.org", "8000", []string{"example.org:8000", "example.org"}},
		{"已带端口时只试去端口的", "example.org:8000", "8000", []string{"example.org"}},
		{"没有端口信息", "example.org", "", []string{"example.org"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupHosts(tt.rawHost, tt.serverPort)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("LookupHosts(%q, %q) = %v，期望 %v", tt.rawHost, tt.serverPort, got, tt.want)
			}
		})
	}
}

func TestDecideServesWhenDomainsAreSame(t *testing.T) {
	s := Site{Domain: "wiki.example", MediaDomain: "wiki.example"}
	got := Decide(s, "wiki.example", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v，期望 Serve", got.Action)
	}
}

func TestDecideServesWhenMediaDomainEmpty(t *testing.T) {
	s := Site{Domain: "wiki.example"}
	got := Decide(s, "wiki.example", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v，期望 Serve", got.Action)
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
		{"媒体路径打在主域上", "wiki.example", "/local--files/a.png", "//media.example/local--files/a.png", Redirect},
		{"非媒体路径打在媒体域上", "media.example", "/scp-173", "//wiki.example/scp-173", Redirect},
		{"媒体路径打在媒体域上", "media.example", "/local--files/a.png", "", Serve},
		{"非媒体路径打在主域上", "wiki.example", "/scp-173", "", Serve},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(split, tt.host, mustURL(t, tt.target))
			if got.Action != tt.wantKind {
				t.Errorf("Decide().Action = %v，期望 %v", got.Action, tt.wantKind)
			}
			if got.Location != tt.wantLoc {
				t.Errorf("Decide().Location = %q，期望 %q", got.Location, tt.wantLoc)
			}
		})
	}
}

func TestDecideKeepsQueryAndEscaping(t *testing.T) {
	got := Decide(split, "wiki.example", mustURL(t, "/local--files/a%20b.png?size=200"))
	want := "//media.example/local--files/a%20b.png?size=200"
	if got.Location != want {
		t.Errorf("Decide().Location = %q，期望 %q", got.Location, want)
	}
}

func TestDecideIgnoresPortInHost(t *testing.T) {
	got := Decide(split, "media.example:8000", mustURL(t, "/local--files/a.png"))
	if got.Action != Serve {
		t.Errorf("Decide().Action = %v，期望 Serve", got.Action)
	}
}

func TestMediaPrefixesAreInRouteTable(t *testing.T) {
	inTable := make(map[string]bool, len(routing.Table))
	for _, r := range routing.Table {
		inTable[r.Prefix] = true
	}
	for _, prefix := range MediaPrefixes {
		if !inTable[prefix] {
			t.Errorf("routing.Table 缺少媒体前缀 %q", prefix)
		}
	}
}
