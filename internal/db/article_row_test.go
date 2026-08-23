package db

import (
	"context"
	"errors"
	"testing"
)

func TestArticleFullName(t *testing.T) {
	cases := []struct {
		category, name, want string
	}{
		{DefaultCategory, "main", "main"},
		{"nav", "top", "nav:top"},
		{"component", "box", "component:box"},
	}
	for _, c := range cases {
		a := &Article{Category: c.category, Name: c.name}
		if got := a.FullName(); got != c.want {
			t.Errorf("Article{%q, %q}.FullName() = %q, want %q", c.category, c.name, got, c.want)
		}
	}
}

func TestArticleDisplayName(t *testing.T) {
	cases := []struct {
		title, want string
	}{
		{"Main Page", "Main Page"},
		{"", "nav:top"},
		{"   ", "nav:top"},
		{"  Padded  ", "Padded"},
	}
	for _, c := range cases {
		a := &Article{Category: "nav", Name: "top", Title: c.title}
		if got := a.DisplayName(); got != c.want {
			t.Errorf("Article{Title: %q}.DisplayName() = %q, want %q", c.title, got, c.want)
		}
	}
}

func TestArticleByName(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleByName(context.Background(), "main")
	if err != nil {
		t.Fatalf("ArticleByName(%q) err = %v, want nil", "main", err)
	}
	if got.Name != "main" {
		t.Errorf("ArticleByName(%q).Name = %q, want %q", "main", got.Name, "main")
	}
	if got.Category != DefaultCategory {
		t.Errorf("ArticleByName(%q).Category = %q, want %q", "main", got.Category, DefaultCategory)
	}
	if got.MediaName == "" {
		t.Errorf("ArticleByName(%q).MediaName = %q, want a uuid", "main", got.MediaName)
	}
}

func TestArticleByNameIsCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleByName(context.Background(), "NAV:Top")
	if err != nil {
		t.Fatalf("ArticleByName(%q) err = %v, want nil", "NAV:Top", err)
	}
	if got.FullName() != "nav:top" {
		t.Errorf("ArticleByName(%q).FullName() = %q, want %q", "NAV:Top", got.FullName(), "nav:top")
	}
}

func TestArticleByNameMissing(t *testing.T) {
	d := newTestDB(t)

	_, err := d.ArticleByName(context.Background(), "no-such-page")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ArticleByName(%q) err = %v, want ErrNotFound", "no-such-page", err)
	}
}
