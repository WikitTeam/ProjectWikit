package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestSourceAsksTheMirrorForTheSamePath(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unreachable", http.StatusBadGateway)
	}))
	defer down.Close()

	var mu sync.Mutex
	var asked []string
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.URL.Path)
		mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/latest.json"):
			w.Write([]byte(`{"version":"v1.2.0","published_at":"2026-09-10T00:00:00Z","packages":{}}`))
		case strings.HasSuffix(r.URL.Path, "/SHA256SUMS"):
			w.Write([]byte("abc  pwikit-v1.2.0-linux-amd64.tar.gz\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mirror.Close()

	s := Source{Releases: down.URL + "/releases", Mirror: mirror.URL + "/projwikit/update"}
	ctx := context.Background()
	if _, err := s.Latest(ctx); err != nil {
		t.Fatalf("Latest() err = %v, want nil", err)
	}
	if _, err := s.Checksums(ctx, "v1.2.0"); err != nil {
		t.Fatalf("Checksums() err = %v, want nil", err)
	}

	want := []string{
		"/projwikit/update/latest/download/latest.json",
		"/projwikit/update/download/v1.2.0/SHA256SUMS",
	}
	if !slices.Equal(asked, want) {
		t.Errorf("mirror paths = %q, want %q", asked, want)
	}
}
