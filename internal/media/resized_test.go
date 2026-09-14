package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func newResizedHandler(t *testing.T, mime string) (*ResizedHandler, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "media", "article-uuid")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() err = %v, want nil", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 1072, 876))
	for y := 0; y < 876; y++ {
		for x := 0; x < 1072; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 64, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("jpeg.Encode() err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file-uuid"), buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}

	files := fakeFiles{"probe/photo.jpg": &db.ArticleFile{
		ArticleMediaName: "article-uuid",
		MediaName:        "file-uuid",
		MimeType:         mime,
		Size:             int64(buf.Len()),
	}}
	return NewResized(root, files), root
}

func getResized(t *testing.T, h *ResizedHandler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestResizedServesTheRequestedSize(t *testing.T) {
	h, _ := newResizedHandler(t, "image/jpeg")

	w := getResized(t, h, "/local--resized-images/probe/photo.jpg/medium.jpg")
	if w.Code != http.StatusOK {
		t.Fatalf("GET medium = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q, want %q", got, "image/jpeg")
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("jpeg.DecodeConfig() err = %v, want nil", err)
	}
	if cfg.Width != 500 || cfg.Height != 409 {
		t.Errorf("GET medium = %dx%d, want 500x409", cfg.Width, cfg.Height)
	}
}

func TestResizedCachesWhatItGenerated(t *testing.T) {
	h, root := newResizedHandler(t, "image/jpeg")

	getResized(t, h, "/local--resized-images/probe/photo.jpg/small.jpg")

	cached := filepath.Join(root, "resized", "article-uuid", "file-uuid", "small.jpg")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("Stat(%q) err = %v, want nil", cached, err)
	}
}

func TestResizedAnswersFromTheCacheWhenTheOriginalIsGone(t *testing.T) {
	h, root := newResizedHandler(t, "image/jpeg")
	getResized(t, h, "/local--resized-images/probe/photo.jpg/square.jpg")

	if err := os.Remove(filepath.Join(root, "media", "article-uuid", "file-uuid")); err != nil {
		t.Fatalf("Remove() err = %v, want nil", err)
	}

	w := getResized(t, h, "/local--resized-images/probe/photo.jpg/square.jpg")
	if w.Code != http.StatusOK {
		t.Errorf("GET square = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestResizedRejectsTheSizeWikidotRejects(t *testing.T) {
	h, _ := newResizedHandler(t, "image/jpeg")

	w := getResized(t, h, "/local--resized-images/probe/photo.jpg/large.jpg")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET large = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestResizedRejectsWhatIsNotAnImage(t *testing.T) {
	h, _ := newResizedHandler(t, "audio/mpeg")

	w := getResized(t, h, "/local--resized-images/probe/photo.jpg/medium.jpg")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET of an audio attachment = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestResizedRejectsAMissingAttachment(t *testing.T) {
	h, _ := newResizedHandler(t, "image/jpeg")

	w := getResized(t, h, "/local--resized-images/probe/nothing.jpg/medium.jpg")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET of a missing attachment = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestResizedRejectsAPathThatEscapes(t *testing.T) {
	h, _ := newResizedHandler(t, "image/jpeg")

	w := getResized(t, h, "/local--resized-images/probe/photo.jpg/../../medium.jpg")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET of an escaping path = %d, want %d", w.Code, http.StatusNotFound)
	}
}
