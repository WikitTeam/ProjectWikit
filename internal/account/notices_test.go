package account

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

func TestLinkUsesTheSchemeTheProxyReports(t *testing.T) {
	trust, err := proxyheader.NewTrust([]string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		trust *proxyheader.Trust
		proto string
		want  string
	}{
		{"trusted proxy terminating tls", trust, "https", "https://wiki.example" + EmailPrefix},
		{"trusted proxy over plain http", trust, "http", "http://wiki.example" + EmailPrefix},
		{"no proxy configured", nil, "https", "http://wiki.example" + EmailPrefix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/-/profile", nil)
			r.RemoteAddr = "127.0.0.1:5000"
			r.Header.Set("X-Forwarded-Proto", tt.proto)
			r = r.WithContext(site.WithSite(r.Context(), &db.Site{Domain: "wiki.example"}))
			d := Deps{Trust: tt.trust, Tokens: token.Generator{Secret: "probe"}}
			got := d.link(r, purposeVerify, 7, token.Custom(purposeVerify, "7"))
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("link() = %q, want prefix %q", got, tt.want)
			}
		})
	}
}
