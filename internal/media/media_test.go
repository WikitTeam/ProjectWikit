package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type fakeFiles map[string]*db.ArticleFile

func (f fakeFiles) ArticleFile(_ context.Context, articleRef, fileName string) (*db.ArticleFile, error) {
	if af, ok := f[articleRef+"/"+fileName]; ok {
		return af, nil
	}
	return nil, db.ErrNotFound
}

// probe is the same 300-byte body the Django oracle was recorded against.
func probe() []byte {
	b := make([]byte, 300)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

func newTestHandler(t *testing.T, files Attachments) (*Handler, string) {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"-/probe", "media/article-uuid"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) err = %v, want nil", dir, err)
		}
	}
	write := func(name string, data []byte) {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%q) err = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"-/probe/a.pdf", "-/probe/a.txt", "-/probe/a.txt.gz", "-/probe/a.bin"} {
		write(name, probe())
	}
	write("-/probe/empty", nil)
	write("media/article-uuid/file-uuid", probe())

	if files == nil {
		files = fakeFiles{}
	}
	return New(root, files), root
}

func get(t *testing.T, h *Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestServeHTTPRangedDefaults(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/a.pdf", nil)

	if w.Code != http.StatusPartialContent {
		t.Errorf("GET a.pdf = %d, want %d", w.Code, http.StatusPartialContent)
	}
	if got, want := w.Header().Get("Content-Range"), "bytes 0-299/300"; got != want {
		t.Errorf("Content-Range = %q, want %q", got, want)
	}
	if got, want := w.Header().Get("Accept-Ranges"), "bytes"; got != want {
		t.Errorf("Accept-Ranges = %q, want %q", got, want)
	}
	if got, want := w.Header().Get("Content-Disposition"), "inline"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}
	if got, want := w.Body.Len(), 300; got != want {
		t.Errorf("body length = %d, want %d", got, want)
	}
}

func TestServeHTTPRangedContentTypeIsDjangoDefault(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/a.pdf", nil)

	// The guessed type never reaches the response.
	if got, want := w.Header().Get("Content-Type"), defaultHTMLMime; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestServeHTTPNonRanged(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/a.txt", nil)

	if w.Code != http.StatusOK {
		t.Errorf("GET a.txt = %d, want %d", w.Code, http.StatusOK)
	}
	if got, want := w.Header().Get("Content-Type"), "text/plain"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got, want := w.Header().Get("Content-Disposition"), `inline; filename="a.txt"`; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}
	// The encoding is guessed and then dropped.
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want %q", got, "")
	}
}

func TestServeHTTPGzipExtensionKeepsInnerType(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/a.txt.gz", nil)

	if got, want := w.Header().Get("Content-Type"), "text/plain"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want %q", got, "")
	}
}

func TestServeHTTPRange(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	tests := []struct {
		header      string
		wantCode    int
		wantRange   string
		wantBodyLen int
	}{
		// One byte short of what was asked for.
		{"bytes=0-99", http.StatusPartialContent, "bytes 0-98/300", 99},
		{"bytes=100-199", http.StatusPartialContent, "bytes 100-198/300", 99},
		{"bytes=250-", http.StatusPartialContent, "bytes 250-299/300", 50},
		// A zero end and a suffix range both read as "not given".
		{"bytes=0-0", http.StatusPartialContent, "bytes 0-299/300", 300},
		{"bytes=-50", http.StatusPartialContent, "bytes 0-49/300", 50},
		{"bytes=0-99999", http.StatusPartialContent, "bytes 0-299/300", 300},
		// The last byte of a file is unreachable.
		{"bytes=299-299", http.StatusRequestedRangeNotSatisfiable, "bytes */300", 0},
		{"bytes=300-400", http.StatusRequestedRangeNotSatisfiable, "bytes */300", 0},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			w := get(t, h, Prefix+"-/probe/a.pdf", map[string]string{"Range": tt.header})
			if w.Code != tt.wantCode {
				t.Errorf("GET a.pdf Range=%q = %d, want %d", tt.header, w.Code, tt.wantCode)
			}
			if got := w.Header().Get("Content-Range"); got != tt.wantRange {
				t.Errorf("Content-Range = %q, want %q", got, tt.wantRange)
			}
			if got := w.Body.Len(); got != tt.wantBodyLen {
				t.Errorf("body length = %d, want %d", got, tt.wantBodyLen)
			}
		})
	}
}

func TestServeHTTPRangeBodyIsTheRequestedSlice(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/a.pdf", map[string]string{"Range": "bytes=100-199"})

	if got, want := w.Body.String(), string(probe()[100:199]); got != want {
		t.Errorf("body = %d bytes, want the 99 bytes at offset 100", len(got))
	}
}

func TestServeHTTPRangeUnsatisfiableSendsNoBody(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/a.pdf", map[string]string{"Range": "bytes=300-400"})

	// Django leaves Content-Length at the file size here.
	if got, want := w.Header().Get("Content-Length"), "0"; got != want {
		t.Errorf("Content-Length = %q, want %q", got, want)
	}
}

func TestServeHTTPMalformedRange(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	// Django raises on all but the first of these.
	for _, header := range []string{"items=0-10", "bogus", "bytes=abc", "bytes=0-10,20-30", "bytes=0-10-20"} {
		t.Run(header, func(t *testing.T) {
			w := get(t, h, Prefix+"-/probe/a.pdf", map[string]string{"Range": header})
			if w.Code != http.StatusRequestedRangeNotSatisfiable {
				t.Errorf("GET a.pdf Range=%q = %d, want %d", header, w.Code, http.StatusRequestedRangeNotSatisfiable)
			}
		})
	}
}

func TestServeHTTPEmptyFileIsUnsatisfiable(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	w := get(t, h, Prefix+"-/probe/empty", nil)

	if w.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("GET empty = %d, want %d", w.Code, http.StatusRequestedRangeNotSatisfiable)
	}
	if got, want := w.Header().Get("Content-Range"), "bytes */0"; got != want {
		t.Errorf("Content-Range = %q, want %q", got, want)
	}
}

func TestServeHTTPConditionalOnlyOnRangedPath(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	future := "Mon, 02 Jan 2100 00:00:00 GMT"

	w := get(t, h, Prefix+"-/probe/a.pdf", map[string]string{"If-Modified-Since": future})
	if w.Code != http.StatusNotModified {
		t.Errorf("GET a.pdf If-Modified-Since=future = %d, want %d", w.Code, http.StatusNotModified)
	}
	// FileResponse never looks at the header.
	w = get(t, h, Prefix+"-/probe/a.txt", map[string]string{"If-Modified-Since": future})
	if w.Code != http.StatusOK {
		t.Errorf("GET a.txt If-Modified-Since=future = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestServeHTTPAttachmentRemap(t *testing.T) {
	files := fakeFiles{"main/report.pdf": {
		ArticleMediaName: "article-uuid",
		MediaName:        "file-uuid",
		MimeType:         "application/pdf",
		Size:             300,
	}}
	h, _ := newTestHandler(t, files)
	w := get(t, h, Prefix+"main/report.pdf", nil)

	if w.Code != http.StatusPartialContent {
		t.Errorf("GET main/report.pdf = %d, want %d", w.Code, http.StatusPartialContent)
	}
	// The remapped branch is the only one whose type reaches the response.
	if got, want := w.Header().Get("Content-Type"), "application/pdf"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got, want := w.Header().Get("Content-Range"), "bytes 0-299/300"; got != want {
		t.Errorf("Content-Range = %q, want %q", got, want)
	}
}

func TestServeHTTPExistingPathSkipsRemap(t *testing.T) {
	files := fakeFiles{"article-uuid/file-uuid": {
		ArticleMediaName: "wrong", MediaName: "wrong", MimeType: "application/pdf", Size: 1,
	}}
	h, _ := newTestHandler(t, files)
	w := get(t, h, Prefix+"article-uuid/file-uuid", nil)

	if w.Code != http.StatusPartialContent {
		t.Errorf("GET article-uuid/file-uuid = %d, want %d", w.Code, http.StatusPartialContent)
	}
	if got, want := w.Header().Get("Content-Type"), defaultHTMLMime; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestServeHTTPNotFound(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	for _, path := range []string{
		Prefix + "-/probe/nope.pdf",
		Prefix + "nope/nope",
		Prefix + "-/probe",
		Prefix,
	} {
		t.Run(path, func(t *testing.T) {
			w := get(t, h, path, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("GET %q = %d, want %d", path, w.Code, http.StatusNotFound)
			}
			if got, want := w.Body.String(), notFoundBody; got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
		})
	}
}

func TestServeHTTPRejectsTraversal(t *testing.T) {
	h, root := newTestHandler(t, nil)
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) err = %v, want nil", outside, err)
	}
	t.Cleanup(func() { os.Remove(outside) })

	// Django serves every one of these.
	for _, path := range []string{
		Prefix + "-/probe/../../../outside.txt",
		Prefix + "-/../../outside.txt",
		Prefix + "article-uuid/../../../outside.txt",
	} {
		t.Run(path, func(t *testing.T) {
			w := get(t, h, path, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("GET %q = %d, want %d", path, w.Code, http.StatusNotFound)
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Errorf("GET %q served a file outside the root", path)
			}
		})
	}
}

func TestServeHTTPRejectsNonReadMethods(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	r := httptest.NewRequest(http.MethodPost, Prefix+"-/probe/a.txt", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST a.txt = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServeHTTPHeadSendsNoBody(t *testing.T) {
	h, _ := newTestHandler(t, nil)
	r := httptest.NewRequest(http.MethodHead, Prefix+"-/probe/a.pdf", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if got, want := w.Header().Get("Content-Length"), "300"; got != want {
		t.Errorf("Content-Length = %q, want %q", got, want)
	}
	if got := w.Body.Len(); got != 0 {
		t.Errorf("body length = %d, want 0", got)
	}
}
