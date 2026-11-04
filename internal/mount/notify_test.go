package mount

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type sent struct {
	calls      int
	siteID     int64
	kind, meta string
	recipients []int64
}

func (s *sent) SendNotification(_ context.Context, siteID int64, kind, meta string, recipients []int64, _ time.Time) error {
	s.calls++
	s.siteID, s.kind, s.meta, s.recipients = siteID, kind, meta, recipients
	return nil
}

const notifyKey = "k3y"

func notifyRequest(method, mount, body, sig string) *http.Request {
	r := httptest.NewRequest(method, NotifyPath, strings.NewReader(body))
	r.Header.Set(MountHeader, mount)
	r.Header.Set(SignatureHeader, sig)
	return r.WithContext(site.WithSite(r.Context(), &db.Site{ID: 7}))
}

func TestNotifyHandlerSends(t *testing.T) {
	s := &sent{}
	h := NotifyHandler(map[string]string{"/scpper/": notifyKey}, s)
	body := `{"users":[3,4],"title":" Tip ","message":"a <b>\nb","link":"/scpper/me/coins/"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, notifyRequest(http.MethodPost, "/scpper/", body, Signature(notifyKey, body)))

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if s.calls != 1 {
		t.Fatalf("SendNotification calls = %d, want 1", s.calls)
	}
	if s.siteID != 7 {
		t.Errorf("siteID = %d, want 7", s.siteID)
	}
	if s.kind != db.NotifyExternal {
		t.Errorf("kind = %q, want %q", s.kind, db.NotifyExternal)
	}
	if len(s.recipients) != 2 || s.recipients[0] != 3 || s.recipients[1] != 4 {
		t.Errorf("recipients = %v, want [3 4]", s.recipients)
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(s.meta), &meta); err != nil {
		t.Fatalf("meta %q: %v", s.meta, err)
	}
	want := map[string]string{"title": "Tip", "message": "a &lt;b&gt;<br>b", "link": "/scpper/me/coins/", "from": "/scpper/"}
	for k, v := range want {
		if meta[k] != v {
			t.Errorf("meta[%q] = %q, want %q", k, meta[k], v)
		}
	}
}

func TestNotifyHandlerRejects(t *testing.T) {
	ok := `{"users":[1],"title":"t","link":"/x"}`
	many := `{"users":[` + strings.TrimSuffix(strings.Repeat("1,", mostRecipients+1), ",") + `],"title":"t"}`
	long := `{"users":[1],"title":"` + strings.Repeat("长", mostTitle+1) + `"}`
	cases := []struct {
		name, method, mount, body, sig string
		want                           int
	}{
		{"get", http.MethodGet, "/scpper/", ok, Signature(notifyKey, ok), http.StatusMethodNotAllowed},
		{"unknown mount", http.MethodPost, "/other/", ok, Signature(notifyKey, ok), http.StatusForbidden},
		{"bad signature", http.MethodPost, "/scpper/", ok, Signature("wrong", ok), http.StatusForbidden},
		{"not json", http.MethodPost, "/scpper/", "{", Signature(notifyKey, "{"), http.StatusBadRequest},
		{"no users", http.MethodPost, "/scpper/", `{"users":[],"title":"t"}`, Signature(notifyKey, `{"users":[],"title":"t"}`), http.StatusBadRequest},
		{"too many users", http.MethodPost, "/scpper/", many, Signature(notifyKey, many), http.StatusBadRequest},
		{"empty title", http.MethodPost, "/scpper/", `{"users":[1],"title":" "}`, Signature(notifyKey, `{"users":[1],"title":" "}`), http.StatusBadRequest},
		{"long title", http.MethodPost, "/scpper/", long, Signature(notifyKey, long), http.StatusBadRequest},
		{"offsite link", http.MethodPost, "/scpper/", `{"users":[1],"title":"t","link":"//evil.example"}`, Signature(notifyKey, `{"users":[1],"title":"t","link":"//evil.example"}`), http.StatusBadRequest},
		{"backslash link", http.MethodPost, "/scpper/", `{"users":[1],"title":"t","link":"/\\evil"}`, Signature(notifyKey, `{"users":[1],"title":"t","link":"/\\evil"}`), http.StatusBadRequest},
		{"absolute link", http.MethodPost, "/scpper/", `{"users":[1],"title":"t","link":"https://evil.example"}`, Signature(notifyKey, `{"users":[1],"title":"t","link":"https://evil.example"}`), http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &sent{}
			h := NotifyHandler(map[string]string{"/scpper/": notifyKey}, s)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, notifyRequest(c.method, c.mount, c.body, c.sig))
			if w.Code != c.want {
				t.Errorf("status = %d, want %d", w.Code, c.want)
			}
			if s.calls != 0 {
				t.Errorf("SendNotification calls = %d, want 0", s.calls)
			}
		})
	}
}
