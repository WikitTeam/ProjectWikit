package articlepage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/article"
)

func TestExcerpt(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"one line", "one line"},
		{"  padded  ", "padded"},
		{"a\n\n\nb", "a\nb"},
		{"  a  \n   \n  b  ", "a\nb"},
		{"a\nb\nc", "a\nb\nc"},
	}
	for _, c := range cases {
		if got := excerpt(c.in); got != c.want {
			t.Errorf("excerpt(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExcerptTruncatesByRunes(t *testing.T) {
	got := excerpt(strings.Repeat("中", 500))
	want := strings.Repeat("中", excerptLimit) + "..."
	if got != want {
		t.Errorf("len([]rune(excerpt(500 runes))) = %d, want %d", len([]rune(got)), len([]rune(want)))
	}
}

func TestExcerptKeepsAnExactFit(t *testing.T) {
	got := excerpt(strings.Repeat("a", excerptLimit))
	if strings.HasSuffix(got, "...") {
		t.Errorf("excerpt(%d chars) ends with an ellipsis, want none", excerptLimit)
	}
}

func TestPathParamsWritesBareKeysAsNull(t *testing.T) {
	_, params := article.ParsePath("main/offset/20/bare", "main")
	got := pathParams(params)
	if len(got) != 2 {
		t.Fatalf("len(pathParams()) = %d, want 2", len(got))
	}
	if got[0].Key != "offset" || got[0].Value != "20" {
		t.Errorf("pathParams()[0] = %v, want offset=20", got[0])
	}
	if got[1].Key != "bare" || got[1].Value != nil {
		t.Errorf("pathParams()[1] = %v, want bare=nil", got[1])
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "second", "third"); got != "second" {
		t.Errorf("firstNonEmpty() = %q, want %q", got, "second")
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("firstNonEmpty() = %q, want %q", got, "")
	}
}

func TestPostIsNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/main", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /main = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want %q", got, "GET, HEAD")
	}
}

func TestRequestWithoutASiteFails(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/main", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /main without a site = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
