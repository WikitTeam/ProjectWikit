package static

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"app.css":            {Data: []byte("body{}")},
		"images/logo.png":    {Data: []byte("\x89PNG")},
		"fonts/x.unknownext": {Data: []byte("blob")},
	}
}

func forwarded() (http.Handler, *bool) {
	hit := new(bool)
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*hit = true
		w.WriteHeader(http.StatusTeapot)
	}), hit
}

func get(t *testing.T, h http.Handler, target string) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec.Result()
}

func TestServesEmbeddedFile(t *testing.T) {
	next, hit := forwarded()
	resp := get(t, New(testFS(), next), Prefix+"app.css")

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if *hit {
		t.Error("next handler called = true, want false")
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "body{}" {
		t.Errorf("body = %q, want %q", body, "body{}")
	}
	if got := resp.Header.Get("Cache-Control"); got != cacheControl {
		t.Errorf("Cache-Control = %q, want %q", got, cacheControl)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := resp.Header.Get("Last-Modified"); got != "" {
		t.Errorf("Last-Modified = %q, want %q", got, "")
	}
}

func TestContentTypeComesFromTheExtension(t *testing.T) {
	cases := []struct{ target, want string }{
		{Prefix + "app.css", "text/css"},
		{Prefix + "images/logo.png", "image/png"},
		{Prefix + "fonts/x.unknownext", "application/octet-stream"},
	}
	for _, c := range cases {
		resp := get(t, New(testFS(), http.NotFoundHandler()), c.target)
		got := resp.Header.Get("Content-Type")
		if len(got) < len(c.want) || got[:len(c.want)] != c.want {
			t.Errorf("Content-Type of %q = %q, want it to start with %q", c.target, got, c.want)
		}
	}
}

func TestETagIsStableAndHonored(t *testing.T) {
	h := New(testFS(), http.NotFoundHandler())

	first := get(t, h, Prefix+"app.css").Header.Get("ETag")
	second := get(t, h, Prefix+"app.css").Header.Get("ETag")
	if first != second {
		t.Errorf("ETag = %q on the second request, want %q", second, first)
	}
	if first == "" {
		t.Fatal("ETag = \"\", want a value")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, Prefix+"app.css", nil)
	req.Header.Set("If-None-Match", first)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified {
		t.Errorf("StatusCode with If-None-Match = %d, want %d", rec.Code, http.StatusNotModified)
	}
}

func TestForwardsWhatTheBundleDoesNotCarry(t *testing.T) {
	cases := []struct {
		name   string
		fsys   *fstest.MapFS
		target string
	}{
		{"missing file", nil, Prefix + "admin/base.css"},
		{"directory", nil, Prefix + "images/"},
		{"escape attempt", nil, Prefix + "../../etc/passwd"},
		{"prefix itself", nil, Prefix},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next, hit := forwarded()
			resp := get(t, New(testFS(), next), c.target)
			if !*hit {
				t.Errorf("next handler called = false, want true")
			}
			if resp.StatusCode != http.StatusTeapot {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusTeapot)
			}
		})
	}
}

func TestForwardsEverythingWithoutABundle(t *testing.T) {
	next, hit := forwarded()
	get(t, New(nil, next), Prefix+"app.css")
	if !*hit {
		t.Error("next handler called = false, want true")
	}
}

func TestForwardsWrites(t *testing.T) {
	next, hit := forwarded()
	rec := httptest.NewRecorder()
	New(testFS(), next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, Prefix+"app.css", nil))
	if !*hit {
		t.Error("next handler called = false, want true")
	}
}

func TestServesRanges(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, Prefix+"app.css", nil)
	req.Header.Set("Range", "bytes=0-3")
	New(testFS(), http.NotFoundHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusPartialContent)
	}
	if got := rec.Body.String(); got != "body" {
		t.Errorf("body = %q, want %q", got, "body")
	}
}
