package modules

import (
	"slices"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type suggestData struct {
	module.Data
	asked  string
	typed  string
	hidden []string
}

func (s *suggestData) SuggestTags(typed string, limit int) ([]db.Suggestion, error) {
	s.asked, s.typed = "tags", typed
	return []db.Suggestion{{Value: "scp", Label: "scp"}}, nil
}

func (s *suggestData) SuggestCategories(hidden []string, typed string, limit int) ([]db.Suggestion, error) {
	s.asked, s.typed, s.hidden = "category", typed, hidden
	return nil, nil
}

func (s *suggestData) SuggestAuthors(typed string, limit int) ([]db.Suggestion, error) {
	s.asked, s.typed = "author", typed
	return nil, nil
}

func (s *suggestData) HiddenCategories(*db.User) ([]string, error) {
	return []string{"admin"}, nil
}

func TestSuggestAPIAsksTheNamedField(t *testing.T) {
	for _, field := range []string{"tags", "category", "author"} {
		data := &suggestData{}
		if _, err := suggestAPI(module.Env{Data: data}, map[string]string{"field": field, "q": " ab "}); err != nil {
			t.Fatalf("suggestAPI(%s) err = %v, want nil", field, err)
		}
		if data.asked != field {
			t.Errorf("suggestAPI(%s) asked %q, want %q", field, data.asked, field)
		}
		if data.typed != "ab" {
			t.Errorf("suggestAPI(%s) typed = %q, want %q", field, data.typed, "ab")
		}
	}
}

func TestSuggestAPIPassesHiddenCategories(t *testing.T) {
	data := &suggestData{}
	if _, err := suggestAPI(module.Env{Data: data}, map[string]string{"field": "category"}); err != nil {
		t.Fatalf("suggestAPI() err = %v, want nil", err)
	}
	if !slices.Equal(data.hidden, []string{"admin"}) {
		t.Errorf("SuggestCategories hidden = %v, want [admin]", data.hidden)
	}
}

func TestSuggestAPIDropsTheExcludingDash(t *testing.T) {
	data := &suggestData{}
	got, err := suggestAPI(module.Env{Data: data}, map[string]string{"field": "tags", "q": "-sc"})
	if err != nil {
		t.Fatalf("suggestAPI() err = %v, want nil", err)
	}
	if data.typed != "sc" {
		t.Errorf("SuggestTags typed = %q, want %q", data.typed, "sc")
	}
	out, err := wikijson.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"items": [{"value": "scp", "label": "scp"}]}`; out != want {
		t.Errorf("suggestAPI() = %s, want %s", out, want)
	}
}

func TestSuggestAPIUnknownFieldIsEmpty(t *testing.T) {
	got, err := suggestAPI(module.Env{Data: &suggestData{}}, map[string]string{"field": "q"})
	if err != nil {
		t.Fatalf("suggestAPI() err = %v, want nil", err)
	}
	out, _ := wikijson.Marshal(got)
	if want := `{"items": []}`; out != want {
		t.Errorf("suggestAPI(q) = %s, want %s", out, want)
	}
}
