package account

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDestinationRefusesOtherSites(t *testing.T) {
	for _, to := range []string{"//evil.example", `/\evil.example`, "https://evil.example", "evil"} {
		r := httptest.NewRequest(http.MethodGet, "/-/login?to="+url.QueryEscape(to), nil)
		if got := destination(r); got != homePath {
			t.Errorf("destination(%q) = %q, want %q", to, got, homePath)
		}
	}
}

func TestDestinationKeepsLocalPath(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/-/login?to="+url.QueryEscape("/scpper/stock"), nil)
	if got := destination(r); got != "/scpper/stock" {
		t.Errorf("destination(/scpper/stock) = %q, want /scpper/stock", got)
	}
}
