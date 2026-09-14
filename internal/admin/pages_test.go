package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type seenRequest struct {
	user   *db.User
	method string
	path   string
	body   string
	exempt bool
}

func recordingArticles(into *seenRequest) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		into.user = auth.FromContext(r.Context())
		into.method = r.Method
		into.path = r.URL.Path
		into.exempt = csrf.Verify(r, nil) == nil
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		into.body = string(buf[:n])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

func TestCallArticleKeepsTheSignedInUser(t *testing.T) {
	var seen seenRequest
	h := &Handler{deps: Deps{Articles: recordingArticles(&seen)}}

	mine := &db.User{ID: 42, Username: "wikitadmin"}
	outer := httptest.NewRequest(http.MethodPost, Prefix+pageSlug+"/", nil)
	outer = outer.WithContext(auth.NewContext(outer.Context(), mine))

	status, answer, err := h.callArticle(outer, http.MethodPut, "scp-173", "", map[string]any{"pageId": "scp-173"})
	if err != nil {
		t.Fatalf("callArticle() err = %v, want nil", err)
	}
	if status != http.StatusOK {
		t.Errorf("callArticle() status = %d, want %d", status, http.StatusOK)
	}
	if answer == "" {
		t.Error("callArticle() answer = \"\", want the API body")
	}
	if seen.user == nil {
		t.Fatal("the article API saw no user, want the signed-in one")
	}
	if seen.user.ID != mine.ID {
		t.Errorf("the article API saw user %d, want %d", seen.user.ID, mine.ID)
	}
}

func TestCallArticleBuildsTheApiRequest(t *testing.T) {
	var seen seenRequest
	h := &Handler{deps: Deps{Articles: recordingArticles(&seen)}}
	outer := httptest.NewRequest(http.MethodPost, Prefix+pageSlug+"/", nil)

	if _, _, err := h.callArticle(outer, http.MethodPut, "scp-173", "log", map[string]any{"revNumber": 3}); err != nil {
		t.Fatalf("callArticle() err = %v, want nil", err)
	}
	if want := "/pw-api/articles/scp-173/log"; seen.path != want {
		t.Errorf("path = %q, want %q", seen.path, want)
	}
	if seen.method != http.MethodPut {
		t.Errorf("method = %q, want %q", seen.method, http.MethodPut)
	}
	if want := `{"revNumber":3}`; seen.body != want {
		t.Errorf("body = %q, want %q", seen.body, want)
	}
	if !seen.exempt {
		t.Error("csrf.Verify() = error, want the sub request to be exempt")
	}
}

func TestCallArticleWithoutTheApi(t *testing.T) {
	h := &Handler{}
	outer := httptest.NewRequest(http.MethodPost, Prefix+pageSlug+"/", nil)
	if _, _, err := h.callArticle(outer, http.MethodDelete, "scp-173", "", nil); err == nil {
		t.Error("callArticle() err = nil, want an error")
	}
}
