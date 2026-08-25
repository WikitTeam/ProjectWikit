// Package respheader carries the response headers the entry layer adds to
// every response, whatever handler produced it.
package respheader

import (
	"net/http"
	"strings"
)

const (
	headerReferrerPolicy = "Referrer-Policy"
	headerCOOP           = "Cross-Origin-Opener-Policy"
	headerVary           = "Vary"

	sameOrigin = "same-origin"
)

// OriginPolicy states which origins may see the referrer and share a browsing
// context group.
func OriginPolicy(next http.Handler) http.Handler {
	return wrap(next, func(h http.Header) {
		setDefault(h, headerReferrerPolicy, sameOrigin)
		setDefault(h, headerCOOP, sameOrigin)
	})
}

// VaryCookie belongs on every response whose handler could have read the
// session, which is every response outside the asset bundle.
func VaryCookie(next http.Handler) http.Handler {
	return wrap(next, func(h http.Header) { addVary(h, "Cookie") })
}

func setDefault(h http.Header, name, value string) {
	if h.Get(name) == "" {
		h.Set(name, value)
	}
}

func addVary(h http.Header, name string) {
	var fields []string
	for _, raw := range h.Values(headerVary) {
		for _, field := range strings.Split(raw, ",") {
			if field = strings.TrimSpace(field); field != "" {
				fields = append(fields, field)
			}
		}
	}
	for _, field := range fields {
		if strings.EqualFold(field, name) {
			return
		}
		if field == "*" {
			h.Set(headerVary, "*")
			return
		}
	}
	h.Set(headerVary, strings.Join(append(fields, name), ", "))
}

func wrap(next http.Handler, apply func(http.Header)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &writer{ResponseWriter: w, apply: apply}
		next.ServeHTTP(ww, r)
		// A handler that writes nothing still leaves net/http a 200 to send.
		ww.applyOnce()
	})
}

// writer sets the headers on the way out rather than before the handler runs,
// so a handler that names one of them keeps its own value.
type writer struct {
	http.ResponseWriter
	apply   func(http.Header)
	applied bool
}

var _ http.ResponseWriter = (*writer)(nil)

func (w *writer) WriteHeader(code int) {
	w.applyOnce()
	w.ResponseWriter.WriteHeader(code)
}

func (w *writer) Write(b []byte) (int, error) {
	w.applyOnce()
	return w.ResponseWriter.Write(b)
}

// Unwrap keeps http.ResponseController reaching the real writer, which the
// reverse proxy needs to flush.
func (w *writer) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *writer) applyOnce() {
	if w.applied {
		return
	}
	w.applied = true
	w.apply(w.Header())
}
