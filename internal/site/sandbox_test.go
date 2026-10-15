package site

import (
	"net/http/httptest"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestSandbox(t *testing.T) {
	cases := []struct {
		name          string
		domain, media string
		mimeType      string
		want          bool
	}{
		{"shared domain", "wiki.test", "wiki.test", "text/html", true},
		{"no media domain", "wiki.test", "", "image/svg+xml", true},
		{"shared domain in other case", "wiki.test", "WIKI.test", "text/html", true},
		{"own media domain", "wiki.test", "files.test", "text/html", false},
		{"pdf", "wiki.test", "wiki.test", "application/pdf", false},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		Sandbox(w, &db.Site{Domain: c.domain, MediaDomain: c.media}, c.mimeType)
		if got := w.Header().Get("Content-Security-Policy") != ""; got != c.want {
			t.Errorf("Sandbox(%s) sets the policy = %v, want %v", c.name, got, c.want)
		}
	}
	w := httptest.NewRecorder()
	Sandbox(w, nil, "text/html")
	if got := w.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("Sandbox(nil) = %q, want none", got)
	}
}
