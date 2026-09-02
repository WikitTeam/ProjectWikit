package csrf

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func requestWithToken(value string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	if value != "" {
		r.AddCookie(&http.Cookie{Name: CookieName, Value: value})
	}
	return r
}

func TestTokenKeepsTheCookie(t *testing.T) {
	want := "abcdefghijklmnopqrstuvwxyz012345"

	got, isNew := Token(requestWithToken(want))
	if got != want {
		t.Errorf("Token() = %q, want %q", got, want)
	}
	if isNew {
		t.Error("Token() isNew = true, want false")
	}
}

func TestTokenMintsOneWhenTheCookieIsMissing(t *testing.T) {
	got, isNew := Token(requestWithToken(""))
	if !isNew {
		t.Error("Token() isNew = false, want true")
	}
	if !Valid(got) {
		t.Errorf("Token() = %q, want a valid token", got)
	}
}

func TestTokenReplacesAMalformedCookie(t *testing.T) {
	for _, bad := range []string{"short", "has-a-dash-in-it-and-is-32-chars", ""} {
		got, isNew := Token(requestWithToken(bad))
		if !isNew {
			t.Errorf("Token(%q) isNew = false, want true", bad)
		}
		if !Valid(got) {
			t.Errorf("Token(%q) = %q, want a valid token", bad, got)
		}
	}
}

func TestValidAcceptsBothLengths(t *testing.T) {
	short := "abcdefghijklmnopqrstuvwxyz012345"
	if !Valid(short) {
		t.Errorf("Valid(%q) = false, want true", short)
	}
	if long := short + short; !Valid(long) {
		t.Errorf("Valid(%q) = false, want true", long)
	}
}

func TestUnmaskAgreesWithTheMaskingItMirrors(t *testing.T) {
	cases := map[string]string{
		"Ykik5gyNkKCT3GYdyRzhVy2JhcwLHZhwh9oiXILOzIKpP3mJk7ewzuNAxXYN32CS": "tZg82Cnbp8iGWxyGWqPpO6V1qVCcwdvw",
		"k2JBICY6IT5TFrIwg2mBYoVbIZKGLRjzpLeJLJsRP297eenQMqQLRbvqcZ2cFmRB": "fTFidhEVhjeoJXPuGyEk3XKpEasG4FIc",
		"EJttOBbCOmSK4PSG360oniSsLI8Qc9kb4z0GIwksQVzDA2lwIg2Sw0kW5lyt149U": "A0Hn45j0cJR3GnD0PkcEjSCEuNANZ5ZT",
		"L2rSN5fa8jqm98Jz3OsOFx2CirITybghj3dIpxoymRnEtfdVEZ1a9yLWrrdKvz2w": "IbW0MCjyoI7suhEwLlJwEbTujaF17yWp",
	}
	for masked, want := range cases {
		if got := unmask(masked); got != want {
			t.Errorf("unmask(%q) = %q, want %q", masked, got, want)
		}
	}
}

func TestUnmaskLeavesABareSecretAlone(t *testing.T) {
	const secret = "tZg82Cnbp8iGWxyGWqPpO6V1qVCcwdvw"
	if got := unmask(secret); got != secret {
		t.Errorf("unmask(%q) = %q, want it unchanged", secret, got)
	}
}

const (
	testSecret = "tZg82Cnbp8iGWxyGWqPpO6V1qVCcwdvw"
	testMasked = "Ykik5gyNkKCT3GYdyRzhVy2JhcwLHZhwh9oiXILOzIKpP3mJk7ewzuNAxXYN32CS"
)

func postWith(cookie, header, origin string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "http://wiki.test/pw-api/preview", nil)
	r.Host = "wiki.test"
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
	}
	if header != "" {
		r.Header.Set(HeaderName, header)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

func TestVerifyAcceptsAMatchingToken(t *testing.T) {
	cases := map[string]*http.Request{
		"both bare":     postWith(testSecret, testSecret, "http://wiki.test"),
		"masked header": postWith(testSecret, testMasked, "http://wiki.test"),
		"masked cookie": postWith(testMasked, testSecret, "http://wiki.test"),
		"both masked":   postWith(testMasked, testMasked, "https://wiki.test"),
		"no origin":     postWith(testSecret, testSecret, ""),
	}
	for name, r := range cases {
		if err := Verify(r, nil); err != nil {
			t.Errorf("Verify(%s) err = %v, want nil", name, err)
		}
	}
}

func TestVerifyRejects(t *testing.T) {
	other := "Ykik5gyNkKCT3GYdyRzhVy2JhcwLHZhwh9oiXILOzIKpP3mJk7ewzuNAxXYN32CT"
	cases := map[string]struct {
		request *http.Request
		want    error
	}{
		"no cookie":        {postWith("", testSecret, ""), ErrNoCookie},
		"malformed cookie": {postWith("short", testSecret, ""), ErrNoCookie},
		"no token":         {postWith(testSecret, "", ""), ErrNoToken},
		"malformed token":  {postWith(testSecret, "short", ""), ErrBadToken},
		"another secret":   {postWith(testSecret, other, ""), ErrBadToken},
		"foreign origin":   {postWith(testSecret, testSecret, "https://evil.test"), ErrBadOrigin},
	}
	for name, c := range cases {
		if err := Verify(c.request, nil); !errors.Is(err, c.want) {
			t.Errorf("Verify(%s) err = %v, want %v", name, err, c.want)
		}
	}
}

func TestVerifyAcceptsATrustedHost(t *testing.T) {
	r := postWith(testSecret, testSecret, "https://media.wiki.test")

	if err := Verify(r, []string{"media.wiki.test"}); err != nil {
		t.Errorf("Verify(trusted host) err = %v, want nil", err)
	}
	if err := Verify(r, nil); !errors.Is(err, ErrBadOrigin) {
		t.Errorf("Verify(untrusted host) err = %v, want %v", err, ErrBadOrigin)
	}
}

func TestVerifyReadsAFormField(t *testing.T) {
	body := strings.NewReader(FormField + "=" + testMasked + "&other=1")
	r, _ := http.NewRequest(http.MethodPost, "http://wiki.test/x", body)
	r.Host = "wiki.test"
	r.Header.Set("Content-Type", formMime)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: testSecret})

	if err := Verify(r, nil); err != nil {
		t.Errorf("Verify(form field) err = %v, want nil", err)
	}
}

func TestVerifyLeavesABodyThatIsNotAFormUnread(t *testing.T) {
	body := `{"module": "x"}`
	r, _ := http.NewRequest(http.MethodPost, "http://wiki.test/x", strings.NewReader(body))
	r.Host = "wiki.test"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(HeaderName, testSecret)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: testSecret})

	if err := Verify(r, nil); err != nil {
		t.Fatalf("Verify(json body) err = %v, want nil", err)
	}
	read, _ := io.ReadAll(r.Body)
	if string(read) != body {
		t.Errorf("body after Verify = %q, want %q", read, body)
	}
}
