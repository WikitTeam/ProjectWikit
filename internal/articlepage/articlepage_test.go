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
	if got := rec.Header().Get("Allow"); got != allowedMethods {
		t.Errorf("Allow = %q, want %q", got, allowedMethods)
	}
}

func TestOptionsIsAnswered(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/main", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS /main = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Allow"); got != allowedMethods {
		t.Errorf("Allow = %q, want %q", got, allowedMethods)
	}
	if got := rec.Header().Get("Content-Length"); got != "0" {
		t.Errorf("Content-Length = %q, want %q", got, "0")
	}
	if got := rec.Body.String(); got != "" {
		t.Errorf("OPTIONS /main body = %q, want %q", got, "")
	}
}

func TestRequestWithoutASiteFails(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/main", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /main without a site = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestNotFoundNamesAskTheCategoryFirst(t *testing.T) {
	got := notFoundNames("scp:9999")
	want := []string{"scp:_404", "_404"}
	if len(got) != len(want) {
		t.Fatalf("notFoundNames(%q) = %v, want %v", "scp:9999", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("notFoundNames(%q)[%d] = %q, want %q", "scp:9999", i, got[i], want[i])
		}
	}
}

func TestNotFoundNamesOfTheDefaultCategory(t *testing.T) {
	got := notFoundNames("no-such-page")
	if len(got) != 1 || got[0] != "_404" {
		t.Errorf("notFoundNames(%q) = %v, want [_404]", "no-such-page", got)
	}
}

func TestNotFoundNamesOfAnExplicitDefaultCategory(t *testing.T) {
	got := notFoundNames("_default:no-such-page")
	if len(got) != 1 || got[0] != "_404" {
		t.Errorf("notFoundNames(%q) = %v, want [_404]", "_default:no-such-page", got)
	}
}
