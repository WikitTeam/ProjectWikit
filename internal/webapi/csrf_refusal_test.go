package webapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
)

func TestCSRFRefusalAsksTheSignedOutToSignIn(t *testing.T) {
	bundle, err := i18n.LoadKeys()
	if err != nil {
		t.Fatalf("LoadKeys() err = %v, want nil", err)
	}
	loc := bundle.Localizer(i18n.DefaultLanguage)
	r := httptest.NewRequest(http.MethodPost, "/pw-api/modules", nil)

	body, status := csrfRefusal(r, loc)
	if status != http.StatusUnauthorized {
		t.Errorf("csrfRefusal(signed out) status = %d, want %d", status, http.StatusUnauthorized)
	}
	if want := field("error", "[api-login-required]"); body != want {
		t.Errorf("csrfRefusal(signed out) body = %q, want %q", body, want)
	}

	r = r.WithContext(auth.NewContext(r.Context(), &db.User{ID: 1}))
	body, status = csrfRefusal(r, loc)
	if status != http.StatusForbidden {
		t.Errorf("csrfRefusal(signed in) status = %d, want %d", status, http.StatusForbidden)
	}
	if want := field("error", "[api-csrf-failed]"); body != want {
		t.Errorf("csrfRefusal(signed in) body = %q, want %q", body, want)
	}
}
