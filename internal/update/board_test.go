package update

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
)

func boardWith(t *testing.T, st db.UpdateState, s Settings) *Board {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatal(err)
	}
	return &Board{Bundle: bundle, Settings: s, Current: "v1.0.0", state: st, loaded: time.Now()}
}

func serveThrough(b *Board, contentType, body string) *httptest.ResponseRecorder {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(body))
	})
	rec := httptest.NewRecorder()
	b.Wrap(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec
}

func TestWrapAddsTheBannerBeforeTheBodyEnds(t *testing.T) {
	at := time.Now().Add(10 * time.Minute)
	b := boardWith(t, db.UpdateState{ScheduledVersion: "v1.1.0", ScheduledAt: &at}, defaultSettings())

	rec := serveThrough(b, "text/html; charset=utf-8", "<html><body><div id=\"container\">page</div></body></html>")
	got := rec.Body.String()
	if !strings.Contains(got, `id="pwikit-update-banner"`) {
		t.Fatalf("page = %q, want the banner in it", got)
	}
	if strings.Index(got, "pwikit-update-banner") > strings.Index(got, "</body>") {
		t.Errorf("page = %q, want the banner before </body>", got)
	}
	if !strings.Contains(got, `<div id="container">page</div>`) {
		t.Errorf("page = %q, want the page's own markup unchanged", got)
	}
	if want := strconv.Itoa(len(got)); rec.Header().Get("Content-Length") != want {
		t.Errorf("Content-Length = %q, want %s", rec.Header().Get("Content-Length"), want)
	}
}

func TestWrapLeavesOtherResponsesAlone(t *testing.T) {
	soon := time.Now().Add(10 * time.Minute)
	later := time.Now().Add(3 * time.Hour)
	private := defaultSettings()
	private.PublicBanner = false

	cases := []struct {
		name        string
		st          db.UpdateState
		s           Settings
		contentType string
	}{
		{"json", db.UpdateState{ScheduledVersion: "v1.1.0", ScheduledAt: &soon}, defaultSettings(), "application/json"},
		{"not yet announced", db.UpdateState{ScheduledVersion: "v1.1.0", ScheduledAt: &later}, defaultSettings(), "text/html"},
		{"nothing scheduled", db.UpdateState{}, defaultSettings(), "text/html"},
		{"banner for admins only", db.UpdateState{ScheduledVersion: "v1.1.0", ScheduledAt: &soon}, private, "text/html"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveThrough(boardWith(t, tt.st, tt.s), tt.contentType, "<body>x</body>")
			if strings.Contains(rec.Body.String(), "pwikit-update-banner") {
				t.Errorf("body = %q, want no banner", rec.Body.String())
			}
		})
	}
}
