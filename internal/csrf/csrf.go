// Package csrf mints the token a form carries and checks the one that comes
// back.
package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"mime"
	"net/http"
	"net/url"
)

// CookieName is the name the frontend already reads the token back from, so
// the two have to stay spelled the same.
const CookieName = "pwikit_csrftoken"

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const secretLength = 32

func Token(r *http.Request) (token string, isNew bool) {
	if cookie, err := r.Cookie(CookieName); err == nil && Valid(cookie.Value) {
		return cookie.Value, false
	}
	return newToken(), true
}

func Valid(token string) bool {
	if len(token) != secretLength && len(token) != 2*secretLength {
		return false
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func newToken() string {
	buf := make([]byte, secretLength)
	rand.Read(buf)
	for i, b := range buf {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf)
}

const (
	HeaderName = "X-CSRFToken"
	FormField  = "csrfmiddlewaretoken"

	formMime = "application/x-www-form-urlencoded"
)

var (
	ErrNoCookie   = errors.New("csrf: the request carries no token cookie")
	ErrNoToken    = errors.New("csrf: the request carries no token")
	ErrBadToken   = errors.New("csrf: the token does not match the cookie")
	ErrBadOrigin  = errors.New("csrf: the origin is not one the site answers for")
	ErrNoReferer  = errors.New("csrf: a request over TLS carries no referer")
	ErrBadReferer = errors.New("csrf: the referer is not one the site answers for")
)

// The host the request arrived on is trusted alongside hosts.
func Verify(r *http.Request, hosts []string) error {
	if err := verifyOrigin(r, hosts); err != nil {
		return err
	}

	cookie, err := r.Cookie(CookieName)
	if err != nil || !Valid(cookie.Value) {
		return ErrNoCookie
	}
	sent := sentToken(r)
	if sent == "" {
		return ErrNoToken
	}
	if !Valid(sent) {
		return ErrBadToken
	}
	if subtle.ConstantTimeCompare([]byte(unmask(sent)), []byte(unmask(cookie.Value))) != 1 {
		return ErrBadToken
	}
	return nil
}

// A body that is not a form is left unread, since the handler still has to read
// it and a form parse would take it away.
func sentToken(r *http.Request) string {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && kind == formMime {
		if err := r.ParseForm(); err == nil {
			if token := r.PostFormValue(FormField); token != "" {
				return token
			}
		}
	}
	return r.Header.Get(HeaderName)
}

func verifyOrigin(r *http.Request, hosts []string) error {
	if origin := r.Header.Get("Origin"); origin != "" {
		for _, host := range append([]string{r.Host}, hosts...) {
			if host != "" && (origin == "https://"+host || origin == "http://"+host) {
				return nil
			}
		}
		return ErrBadOrigin
	}
	// A browser sends Origin on every unsafe request, so what follows is only
	// for the clients that do not.
	if r.TLS == nil {
		return nil
	}
	referer := r.Header.Get("Referer")
	if referer == "" {
		return ErrNoReferer
	}
	parsed, err := url.Parse(referer)
	if err != nil || parsed.Scheme != "https" {
		return ErrBadReferer
	}
	for _, host := range append([]string{r.Host}, hosts...) {
		if host != "" && parsed.Host == host {
			return nil
		}
	}
	return ErrBadReferer
}

func unmask(token string) string {
	if len(token) != 2*secretLength {
		return token
	}
	mask, cipher := token[:secretLength], token[secretLength:]
	out := make([]byte, secretLength)
	for i := 0; i < secretLength; i++ {
		out[i] = alphabet[((index(cipher[i])-index(mask[i]))%len(alphabet)+len(alphabet))%len(alphabet)]
	}
	return string(out)
}

func index(c byte) int {
	switch {
	case c >= 'a' && c <= 'z':
		return int(c - 'a')
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 26
	}
	return int(c-'0') + 52
}
