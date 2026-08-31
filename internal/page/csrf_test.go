package page

import (
	"net/http"
	"testing"
)

func requestWithToken(value string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	if value != "" {
		r.AddCookie(&http.Cookie{Name: CSRFCookie, Value: value})
	}
	return r
}

func TestCSRFTokenKeepsTheCookie(t *testing.T) {
	want := "abcdefghijklmnopqrstuvwxyz012345"

	got, isNew := CSRFToken(requestWithToken(want))
	if got != want {
		t.Errorf("CSRFToken() = %q, want %q", got, want)
	}
	if isNew {
		t.Error("CSRFToken() isNew = true, want false")
	}
}

func TestCSRFTokenMintsOneWhenTheCookieIsMissing(t *testing.T) {
	got, isNew := CSRFToken(requestWithToken(""))
	if !isNew {
		t.Error("CSRFToken() isNew = false, want true")
	}
	if !validCSRFToken(got) {
		t.Errorf("CSRFToken() = %q, want a valid token", got)
	}
}

func TestCSRFTokenReplacesAMalformedCookie(t *testing.T) {
	for _, bad := range []string{"short", "has-a-dash-in-it-and-is-32-chars", ""} {
		got, isNew := CSRFToken(requestWithToken(bad))
		if !isNew {
			t.Errorf("CSRFToken(%q) isNew = false, want true", bad)
		}
		if !validCSRFToken(got) {
			t.Errorf("CSRFToken(%q) = %q, want a valid token", bad, got)
		}
	}
}

func TestValidCSRFTokenAcceptsBothLengths(t *testing.T) {
	short := "abcdefghijklmnopqrstuvwxyz012345"
	if !validCSRFToken(short) {
		t.Errorf("validCSRFToken(%q) = false, want true", short)
	}
	if long := short + short; !validCSRFToken(long) {
		t.Errorf("validCSRFToken(%q) = false, want true", long)
	}
}
