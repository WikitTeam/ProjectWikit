package compress

import (
	"compress/gzip"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func handlerOf(body string, headers map[string]string, status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, value := range headers {
			w.Header().Set(name, value)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

func serve(t *testing.T, next http.Handler, accept string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/main", nil)
	if accept != "" {
		req.Header.Set("Accept-Encoding", accept)
	}
	rec := httptest.NewRecorder()
	New(next).ServeHTTP(rec, req)
	return rec.Result()
}

func unpack(t *testing.T, res *http.Response) string {
	t.Helper()
	zr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader() err = %v, want nil", err)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("ReadAll() err = %v, want nil", err)
	}
	return string(out)
}

func TestCompressesALongBody(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, nil, http.StatusOK), "gzip, deflate")

	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding = %q, want %q", got, "gzip")
	}
	if got := unpack(t, res); got != body {
		t.Errorf("body = %q, want %q", got, body)
	}
}

func TestContentLengthCountsTheCompressedBody(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, nil, http.StatusOK), "gzip")

	length, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		t.Fatalf("Atoi(Content-Length) err = %v, want nil", err)
	}
	if length >= len(body) {
		t.Errorf("Content-Length = %d, want less than %d", length, len(body))
	}
}

func TestShortBodyIsLeftAlone(t *testing.T) {
	res := serve(t, handlerOf("short", nil, http.StatusOK), "gzip")

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want %q", got, "")
	}
	if got := res.Header.Get("Vary"); got != "" {
		t.Errorf("Vary = %q, want %q", got, "")
	}
}

func TestVaryIsSetWithoutCompressing(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, nil, http.StatusOK), "deflate")

	if got := res.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
	}
	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want %q", got, "")
	}
}

func TestVaryKeepsWhatTheHandlerSet(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, map[string]string{"Vary": "Cookie"}, http.StatusOK), "gzip")

	if got := res.Header.Get("Vary"); got != "Cookie, Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Cookie, Accept-Encoding")
	}
}

func TestVaryIsNotRepeated(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, map[string]string{"Vary": "Cookie, accept-encoding"}, http.StatusOK), "gzip")

	if got := res.Header.Get("Vary"); got != "Cookie, accept-encoding" {
		t.Errorf("Vary = %q, want %q", got, "Cookie, accept-encoding")
	}
}

func TestAlreadyEncodedBodyIsLeftAlone(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, map[string]string{"Content-Encoding": "br"}, http.StatusOK), "gzip")

	if got := res.Header.Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding = %q, want %q", got, "br")
	}
	if got := res.Header.Get("Vary"); got != "" {
		t.Errorf("Vary = %q, want %q", got, "")
	}
}

func TestIncompressibleBodyIsLeftAlone(t *testing.T) {
	raw := make([]byte, 512)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand.Read() err = %v, want nil", err)
	}
	res := serve(t, handlerOf(string(raw), nil, http.StatusOK), "gzip")

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want %q", got, "")
	}
	if got := res.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
	}
}

func TestStrongETagBecomesWeak(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	res := serve(t, handlerOf(body, map[string]string{"ETag": `"abc"`}, http.StatusOK), "gzip")

	if got := res.Header.Get("ETag"); got != `W/"abc"` {
		t.Errorf("ETag = %q, want %q", got, `W/"abc"`)
	}
}

func TestStatusIsKept(t *testing.T) {
	body := strings.Repeat("missing page ", 100)
	res := serve(t, handlerOf(body, nil, http.StatusNotFound), "gzip")

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestPaddingVariesBetweenResponses(t *testing.T) {
	body := strings.Repeat("hello wiki ", 100)
	lengths := make(map[string]bool)
	for i := 0; i < 20; i++ {
		res := serve(t, handlerOf(body, nil, http.StatusOK), "gzip")
		lengths[res.Header.Get("Content-Length")] = true
	}
	if len(lengths) < 2 {
		t.Errorf("len(distinct Content-Length over 20 responses) = %d, want more than 1", len(lengths))
	}
}
