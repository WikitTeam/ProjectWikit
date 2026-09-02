package roles

import "testing"

func categorized(id int64) *int64 { return &id }

func TestShowcaseOfReadsTheProfileColumn(t *testing.T) {
	rs := []Role{
		{Slug: "staff", Name: "Staff", ProfileVisualMode: ProfileStatus, InlineVisualMode: InlineHidden},
		{Slug: "hidden", Name: "Hidden", ProfileVisualMode: ProfileHidden, InlineVisualMode: InlineBadge},
		{Slug: "friend", Name: "Friend", ProfileVisualMode: ProfileBadge, BadgeBg: "#111"},
	}

	got := ShowcaseOf(rs)
	if len(got.Titles) != 1 || got.Titles[0] != "Staff" {
		t.Errorf("ShowcaseOf().Titles = %v, want [Staff]", got.Titles)
	}
	if len(got.Badges) != 1 || got.Badges[0].Text != "Friend" {
		t.Errorf("ShowcaseOf().Badges = %v, want one named Friend", got.Badges)
	}
}

func TestShowcaseOfKeepsOneRolePerCategory(t *testing.T) {
	rs := []Role{
		{Slug: "first", Name: "First", CategoryID: categorized(1), ProfileVisualMode: ProfileStatus},
		{Slug: "second", Name: "Second", CategoryID: categorized(1), ProfileVisualMode: ProfileBadge},
		{Slug: "other", Name: "Other", CategoryID: categorized(2), ProfileVisualMode: ProfileStatus},
		{Slug: "loose", Name: "Loose", ProfileVisualMode: ProfileStatus},
	}

	got := ShowcaseOf(rs)
	want := []string{"First", "Other", "Loose"}
	if len(got.Titles) != len(want) {
		t.Fatalf("ShowcaseOf().Titles = %v, want %v", got.Titles, want)
	}
	for i := range want {
		if got.Titles[i] != want[i] {
			t.Errorf("ShowcaseOf().Titles[%d] = %q, want %q", i, got.Titles[i], want[i])
		}
	}
	if len(got.Badges) != 0 {
		t.Errorf("ShowcaseOf().Badges = %v, want none", got.Badges)
	}
}

func TestShowcaseOfNames(t *testing.T) {
	rs := []Role{
		{Slug: "a", ProfileVisualMode: ProfileStatus},
		{Slug: "b", Name: "B", BadgeText: "badge", ProfileVisualMode: ProfileBadge},
		{Slug: "c", Name: "C", ProfileVisualMode: ProfileBadge},
	}

	got := ShowcaseOf(rs)
	if len(got.Titles) != 1 || got.Titles[0] != "a" {
		t.Errorf("ShowcaseOf().Titles = %v, want [a]", got.Titles)
	}
	if len(got.Badges) != 2 || got.Badges[0].Text != "badge" || got.Badges[1].Text != "C" {
		t.Errorf("ShowcaseOf().Badges = %v, want [badge C]", got.Badges)
	}
}
