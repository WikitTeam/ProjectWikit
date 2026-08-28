package page

import (
	"net/http"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func article(title string) *db.Article {
	return &db.Article{ID: 1, Category: db.DefaultCategory, Name: "main", Title: title}
}

func TestNewContextDefaults(t *testing.T) {
	c := NewContext(article("Main Page"), nil, nil, nil)

	if c.Status != http.StatusOK {
		t.Errorf("NewContext(...).Status = %d, want %d", c.Status, http.StatusOK)
	}
	if c.Title != "Main Page" {
		t.Errorf("NewContext(...).Title = %q, want %q", c.Title, "Main Page")
	}
	if c.RedirectTo != "" {
		t.Errorf("NewContext(...).RedirectTo = %q, want %q", c.RedirectTo, "")
	}
}

func TestNewContextWithoutArticleHasNoTitle(t *testing.T) {
	if got := NewContext(nil, nil, nil, nil).Title; got != "" {
		t.Errorf("NewContext(nil, ...).Title = %q, want %q", got, "")
	}
}

func TestCloneWithCarriesStatusRedirectAndTitle(t *testing.T) {
	c := NewContext(article("Main Page"), nil, nil, nil)
	c.Status = http.StatusNotFound
	c.RedirectTo = "/elsewhere"
	c.Title = "Overridden"

	clone := c.CloneWith(article("Nav"), nil, nil, nil)

	if clone.Status != http.StatusNotFound {
		t.Errorf("CloneWith(...).Status = %d, want %d", clone.Status, http.StatusNotFound)
	}
	if clone.RedirectTo != "/elsewhere" {
		t.Errorf("CloneWith(...).RedirectTo = %q, want %q", clone.RedirectTo, "/elsewhere")
	}
	if clone.Title != "Overridden" {
		t.Errorf("CloneWith(...).Title = %q, want %q", clone.Title, "Overridden")
	}
}

func TestCloneWithLeavesStylesBehind(t *testing.T) {
	c := NewContext(article("Main Page"), nil, nil, nil)
	c.ComputedStyle = "a{}"
	c.AddCSS = "b{}"
	c.OGImage = "/i.png"
	c.OGDescription = "d"

	clone := c.CloneWith(nil, nil, nil, nil)

	if clone.ComputedStyle != "" {
		t.Errorf("CloneWith(...).ComputedStyle = %q, want %q", clone.ComputedStyle, "")
	}
	if clone.AddCSS != "" {
		t.Errorf("CloneWith(...).AddCSS = %q, want %q", clone.AddCSS, "")
	}
	if clone.OGImage != "" {
		t.Errorf("CloneWith(...).OGImage = %q, want %q", clone.OGImage, "")
	}
	if clone.OGDescription != "" {
		t.Errorf("CloneWith(...).OGDescription = %q, want %q", clone.OGDescription, "")
	}
}

func TestMergeAccumulatesComputedStyle(t *testing.T) {
	c := NewContext(nil, nil, nil, nil)
	c.ComputedStyle = "a{}"
	other := NewContext(nil, nil, nil, nil)
	other.ComputedStyle = "b{}"

	c.Merge(other)

	if c.ComputedStyle != "a{}b{}" {
		t.Errorf("Merge(...).ComputedStyle = %q, want %q", c.ComputedStyle, "a{}b{}")
	}
}

func TestMergeOverwritesStatusRedirectAndTitle(t *testing.T) {
	c := NewContext(article("Main Page"), nil, nil, nil)
	other := NewContext(nil, nil, nil, nil)
	other.Status = http.StatusForbidden
	other.RedirectTo = "/elsewhere"
	other.Title = "Nested"

	c.Merge(other)

	if c.Status != http.StatusForbidden {
		t.Errorf("Merge(...).Status = %d, want %d", c.Status, http.StatusForbidden)
	}
	if c.RedirectTo != "/elsewhere" {
		t.Errorf("Merge(...).RedirectTo = %q, want %q", c.RedirectTo, "/elsewhere")
	}
	if c.Title != "Nested" {
		t.Errorf("Merge(...).Title = %q, want %q", c.Title, "Nested")
	}
}

func TestMergeLeavesAddCSSAndOpenGraphAlone(t *testing.T) {
	c := NewContext(nil, nil, nil, nil)
	c.AddCSS = "a{}"
	c.OGImage = "/keep.png"
	other := NewContext(nil, nil, nil, nil)
	other.AddCSS = "b{}"
	other.OGImage = "/drop.png"

	c.Merge(other)

	if c.AddCSS != "a{}" {
		t.Errorf("Merge(...).AddCSS = %q, want %q", c.AddCSS, "a{}")
	}
	if c.OGImage != "/keep.png" {
		t.Errorf("Merge(...).OGImage = %q, want %q", c.OGImage, "/keep.png")
	}
}
