package roles

import (
	"errors"
	"testing"
)

func category(id int64) *int64 { return &id }

func badge(slug string, index int, cat *int64) Role {
	return Role{Slug: slug, Index: index, CategoryID: cat, InlineVisualMode: InlineBadge,
		BadgeBg: "#808080", BadgeTextColor: "#ffffff"}
}

func failingLoader(t *testing.T) IconLoader {
	t.Helper()
	return func(name string) (string, error) {
		t.Errorf("IconLoader called with %q, want no call", name)
		return "", nil
	}
}

func TestNameTailsSkipsHidden(t *testing.T) {
	rs := []Role{{Slug: "hidden", InlineVisualMode: InlineHidden}}
	tails, err := NameTails(rs, failingLoader(t))
	if err != nil {
		t.Fatalf("NameTails() err = %v, want nil", err)
	}
	if len(tails.Badges) != 0 {
		t.Errorf("len(NameTails().Badges) = %d, want 0", len(tails.Badges))
	}
	if len(tails.Icons) != 0 {
		t.Errorf("len(NameTails().Icons) = %d, want 0", len(tails.Icons))
	}
}

func TestNameTailsKeepsFirstOfEachCategory(t *testing.T) {
	rs := []Role{badge("first", 0, category(7)), badge("second", 1, category(7))}
	tails, err := NameTails(rs, failingLoader(t))
	if err != nil {
		t.Fatalf("NameTails() err = %v, want nil", err)
	}
	if len(tails.Badges) != 1 {
		t.Fatalf("len(NameTails().Badges) = %d, want 1", len(tails.Badges))
	}
	if tails.Badges[0].Text != "first" {
		t.Errorf("NameTails().Badges[0].Text = %q, want %q", tails.Badges[0].Text, "first")
	}
}

func TestNameTailsKeepsEveryUncategorizedRole(t *testing.T) {
	rs := []Role{badge("one", 0, nil), badge("two", 1, nil)}
	tails, err := NameTails(rs, failingLoader(t))
	if err != nil {
		t.Fatalf("NameTails() err = %v, want nil", err)
	}
	if len(tails.Badges) != 2 {
		t.Errorf("len(NameTails().Badges) = %d, want 2", len(tails.Badges))
	}
}

func TestNameTailsSeparatesModesInOneCategory(t *testing.T) {
	icon := Role{Slug: "icon", Index: 1, CategoryID: category(7), InlineVisualMode: InlineIcon, Icon: "i.svg"}
	rs := []Role{badge("badge", 0, category(7)), icon}
	tails, err := NameTails(rs, func(string) (string, error) { return "<svg></svg>", nil })
	if err != nil {
		t.Fatalf("NameTails() err = %v, want nil", err)
	}
	if len(tails.Badges) != 1 {
		t.Errorf("len(NameTails().Badges) = %d, want 1", len(tails.Badges))
	}
	if len(tails.Icons) != 1 {
		t.Errorf("len(NameTails().Icons) = %d, want 1", len(tails.Icons))
	}
}

func TestNameTailsFallsBackToSlug(t *testing.T) {
	tails, err := NameTails([]Role{badge("the-slug", 0, nil)}, failingLoader(t))
	if err != nil {
		t.Fatalf("NameTails() err = %v, want nil", err)
	}
	if tails.Badges[0].Text != "the-slug" {
		t.Errorf("NameTails().Badges[0].Text = %q, want %q", tails.Badges[0].Text, "the-slug")
	}
}

func TestNameTailsSkipsIconRoleWithoutFile(t *testing.T) {
	rs := []Role{{Slug: "icon", InlineVisualMode: InlineIcon}}
	tails, err := NameTails(rs, failingLoader(t))
	if err != nil {
		t.Fatalf("NameTails() err = %v, want nil", err)
	}
	if len(tails.Icons) != 0 {
		t.Errorf("len(NameTails().Icons) = %d, want 0", len(tails.Icons))
	}
}

func TestNameTailsReportsLoaderFailure(t *testing.T) {
	want := errors.New("gone")
	rs := []Role{{Slug: "icon", InlineVisualMode: InlineIcon, Icon: "i.svg"}}
	_, err := NameTails(rs, func(string) (string, error) { return "", want })
	if !errors.Is(err, want) {
		t.Errorf("NameTails() err = %v, want %v", err, want)
	}
}

func TestColorizeIcon(t *testing.T) {
	got, err := ColorizeIcon(`<?xml?><svg a="b"><path/></svg>`, "#ff8800")
	if err != nil {
		t.Fatalf("ColorizeIcon() err = %v, want nil", err)
	}
	want := "%3Csvg%20a%3D%22b%22%3E%3Cstyle%3Esvg%7Bcolor%3A%23ff8800%7D%3C/style%3E%3Cpath/%3E%3C/svg%3E"
	if got != want {
		t.Errorf("ColorizeIcon() = %q, want %q", got, want)
	}
}

func TestColorizeIconRejectsFileWithoutSVGTag(t *testing.T) {
	_, err := ColorizeIcon("<html></html>", "#000000")
	if !errors.Is(err, ErrNoSVGTag) {
		t.Errorf("ColorizeIcon() err = %v, want %v", err, ErrNoSVGTag)
	}
}
