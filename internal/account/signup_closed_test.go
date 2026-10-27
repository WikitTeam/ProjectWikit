package account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
)

func closedSignup(t *testing.T) (*SignupHandler, *i18n.Bundle) {
	t.Helper()
	b, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return NewSignup(Deps{Bundle: b, Assets: static.NewAssets(nil)}), b
}

func closedRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader("username=a&email=a@b.c&password=x"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r.WithContext(site.WithSite(r.Context(), &db.Site{Domain: "example.com", SignupOpen: false}))
}

func TestSignupClosedRefusesThePage(t *testing.T) {
	h, b := closedSignup(t)
	want := b.Localizer(i18n.DefaultLanguage).T("signup.closed")
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, closedRequest(method, SignupPath))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s status = %d, want %d", method, SignupPath, w.Code, http.StatusForbidden)
		}
		body := w.Body.String()
		if !strings.Contains(body, `class="meta w-signup-closed"`) {
			t.Errorf("%s %s body lacks the w-signup-closed notice", method, SignupPath)
		}
		if !strings.Contains(body, want) {
			t.Errorf("%s %s body lacks %q", method, SignupPath, want)
		}
		if strings.Contains(body, `id="signup-form"`) {
			t.Errorf("%s %s body has the signup form, want none", method, SignupPath)
		}
	}
}

func TestSignupClosedRefusesTheWikidotSteps(t *testing.T) {
	h, _ := closedSignup(t)
	for _, path := range []string{CheckPath + "?username=a", SendCodePath} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, closedRequest(http.MethodPost, path))
		if w.Code != http.StatusForbidden {
			t.Errorf("POST %s status = %d, want %d", path, w.Code, http.StatusForbidden)
		}
	}
}
