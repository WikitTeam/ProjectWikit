package page

import (
	"crypto/rand"
	"net/http"
)

// CSRFCookie is the name the frontend already reads the token back from, so
// the two have to stay spelled the same.
const CSRFCookie = "pwikit_csrftoken"

const csrfAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const csrfTokenLength = 32

func CSRFToken(r *http.Request) (token string, isNew bool) {
	if cookie, err := r.Cookie(CSRFCookie); err == nil && validCSRFToken(cookie.Value) {
		return cookie.Value, false
	}
	return newCSRFToken(), true
}

func validCSRFToken(token string) bool {
	if len(token) != csrfTokenLength && len(token) != 2*csrfTokenLength {
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

func newCSRFToken() string {
	buf := make([]byte, csrfTokenLength)
	rand.Read(buf)
	for i, b := range buf {
		buf[i] = csrfAlphabet[int(b)%len(csrfAlphabet)]
	}
	return string(buf)
}
