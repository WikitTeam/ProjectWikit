package admin

import (
	"strings"
	"testing"
)

func TestSettingChoicesSpellOutWhatFollowingTheSiteMeans(t *testing.T) {
	loc := testLocalizer(t)

	got := settingChoices(loc, "admin.rating-", categoryRatingModes, "stars")
	if len(got) != 4 {
		t.Fatalf("len(settingChoices(category)) = %d, want 4", len(got))
	}
	if got[0].Value != followSite {
		t.Errorf("settingChoices()[0].Value = %q, want %q", got[0].Value, followSite)
	}
	if want := loc.T("admin.rating-stars"); !strings.Contains(got[0].Label, want) {
		t.Errorf("settingChoices()[0].Label = %q, want it to name %q", got[0].Label, want)
	}
	if got[3].Label != loc.T("admin.rating-stars") {
		t.Errorf("settingChoices()[3].Label = %q, want %q", got[3].Label, loc.T("admin.rating-stars"))
	}
}

func TestSiteSetting(t *testing.T) {
	cases := map[string]string{"": "updown", followSite: "updown", "stars": "stars"}
	for value, want := range cases {
		if got := siteSetting(value, "updown"); got != want {
			t.Errorf("siteSetting(%q, updown) = %q, want %q", value, got, want)
		}
	}
}
