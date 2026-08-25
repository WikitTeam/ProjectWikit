package respheader

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// serve reads the recorder snapshot rather than its header map, so a header
// set after the response started counts as missing.
func serve(t *testing.T, h http.Handler) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec.Result()
}

func TestOriginPolicySetsReferrerPolicy(t *testing.T) {
	resp := serve(t, OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Referrer-Policy"), "same-origin"; got != want {
		t.Errorf("Referrer-Policy = %q, want %q", got, want)
	}
}

func TestOriginPolicySetsCrossOriginOpenerPolicy(t *testing.T) {
	resp := serve(t, OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Cross-Origin-Opener-Policy"), "same-origin"; got != want {
		t.Errorf("Cross-Origin-Opener-Policy = %q, want %q", got, want)
	}
}

func TestOriginPolicyKeepsHandlerValue(t *testing.T) {
	resp := serve(t, OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Referrer-Policy"), "no-referrer"; got != want {
		t.Errorf("Referrer-Policy = %q, want %q", got, want)
	}
}

func TestOriginPolicySetsHeadersWithoutWrite(t *testing.T) {
	resp := serve(t, OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	if got, want := resp.Header.Get("Referrer-Policy"), "same-origin"; got != want {
		t.Errorf("Referrer-Policy = %q, want %q", got, want)
	}
}

func TestOriginPolicyPassesStatusThrough(t *testing.T) {
	resp := serve(t, OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusFound)
	})))
	if got, want := resp.StatusCode, http.StatusFound; got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
}

func TestOriginPolicyPassesBodyThrough(t *testing.T) {
	resp := serve(t, OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("body"))
	})))
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() = %v, want nil", err)
	}
	if got, want := string(body), "body"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestVaryCookieSetsHeader(t *testing.T) {
	resp := serve(t, VaryCookie(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Vary"), "Cookie"; got != want {
		t.Errorf("Vary = %q, want %q", got, want)
	}
}

func TestVaryCookieAppendsToHandlerValue(t *testing.T) {
	resp := serve(t, VaryCookie(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Vary"), "Accept-Encoding, Cookie"; got != want {
		t.Errorf("Vary = %q, want %q", got, want)
	}
}

func TestVaryCookieDoesNotRepeatField(t *testing.T) {
	resp := serve(t, VaryCookie(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "cookie")
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Vary"), "cookie"; got != want {
		t.Errorf("Vary = %q, want %q", got, want)
	}
}

func TestVaryCookieCollapsesRepeatedHeaderLines(t *testing.T) {
	resp := serve(t, VaryCookie(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Add("Vary", "Origin")
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := len(resp.Header.Values("Vary")), 1; got != want {
		t.Errorf("len(Vary) = %d, want %d", got, want)
	}
	if got, want := resp.Header.Get("Vary"), "Accept-Encoding, Origin, Cookie"; got != want {
		t.Errorf("Vary = %q, want %q", got, want)
	}
}

func TestVaryCookieKeepsWildcard(t *testing.T) {
	resp := serve(t, VaryCookie(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "*")
		w.WriteHeader(http.StatusOK)
	})))
	if got, want := resp.Header.Get("Vary"), "*"; got != want {
		t.Errorf("Vary = %q, want %q", got, want)
	}
}

func TestWriterUnwrapsToResponseWriter(t *testing.T) {
	var flushed bool
	h := OriginPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush() = %v, want nil", err)
		}
		flushed = true
	}))
	serve(t, h)
	if !flushed {
		t.Error("Flush() did not run, want it to run")
	}
}
