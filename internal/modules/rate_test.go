package modules

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type fakeRateData struct {
	module.Data
	siteMode     string
	categoryMode string
	stats        db.VoteStats
	voted        bool
	askedVoted   bool
	askedUserID  *int64
}

func (f *fakeRateData) SiteRatingMode() (string, error) { return f.siteMode, nil }

func (f *fakeRateData) CategoryRatingMode(string) (string, error) { return f.categoryMode, nil }

func (f *fakeRateData) VoteStats(int64) (db.VoteStats, error) { return f.stats, nil }

func (f *fakeRateData) HasVoted(_ int64, userID *int64) (bool, error) {
	f.askedVoted = true
	f.askedUserID = userID
	return f.voted, nil
}

func rateEnv(data *fakeRateData, article *db.Article, user *db.User) module.Env {
	var pc *page.Context
	if article != nil {
		pc = page.NewContext(article, article, nil, user)
	}
	return module.Env{Page: pc, User: user, Data: data}
}

func TestRenderRateWithoutAPage(t *testing.T) {
	_, err := renderRate(rateEnv(&fakeRateData{}, nil, nil), nil, "")
	var blocked *module.Error
	if !asModuleError(err, &blocked) {
		t.Fatalf("renderRate() err = %v, want *module.Error", err)
	}
	if blocked.Message != "module-rate-no-page" {
		t.Errorf("renderRate() message = %q, want %q", blocked.Message, "module-rate-no-page")
	}
}

func TestRenderRateDisabledRendersNothing(t *testing.T) {
	data := &fakeRateData{siteMode: page.RatingModeDisabled}
	got, err := renderRate(rateEnv(data, &db.Article{ID: 1, Category: "probe", Name: "x"}, nil), nil, "")
	if err != nil {
		t.Fatalf("renderRate() err = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("renderRate() = %q, want %q", got, "")
	}
}

func TestRenderRateEscapesThePageID(t *testing.T) {
	data := &fakeRateData{siteMode: page.RatingModeUpDown}
	article := &db.Article{ID: 1, Category: "probe", Name: `a"b&c`}
	got, err := renderRate(rateEnv(data, article, nil), nil, "")
	if err != nil {
		t.Fatalf("renderRate() err = %v, want nil", err)
	}
	want := `data-page-id="probe:a&quot;b&amp;c"`
	if !strings.Contains(got, want) {
		t.Errorf("renderRate() = %q, want it to contain %q", got, want)
	}
}

func TestRenderRateLeavesAnAnonymousVoteUnasked(t *testing.T) {
	data := &fakeRateData{siteMode: page.RatingModeStars}
	_, err := renderRate(rateEnv(data, &db.Article{ID: 1, Category: "probe", Name: "x"}, nil), nil, "")
	if err != nil {
		t.Fatalf("renderRate() err = %v, want nil", err)
	}
	if data.askedVoted {
		t.Error("HasVoted called = true, want false")
	}
}

func TestRenderRateAsksAboutTheSignedInVote(t *testing.T) {
	data := &fakeRateData{siteMode: page.RatingModeStars, voted: true}
	user := &db.User{ID: 7}
	got, err := renderRate(rateEnv(data, &db.Article{ID: 1, Category: "probe", Name: "x"}, user), nil, "")
	if err != nil {
		t.Fatalf("renderRate() err = %v, want nil", err)
	}
	if data.askedUserID == nil || *data.askedUserID != user.ID {
		t.Errorf("HasVoted user id = %v, want %d", data.askedUserID, user.ID)
	}
	if !strings.Contains(got, starColorRated) {
		t.Errorf("renderRate() = %q, want it to contain %q", got, starColorRated)
	}
}

func TestRenderRateStarsWithoutVotes(t *testing.T) {
	data := &fakeRateData{siteMode: page.RatingModeStars}
	got, err := renderRate(rateEnv(data, &db.Article{ID: 1, Category: "probe", Name: "x"}, nil), nil, "")
	if err != nil {
		t.Fatalf("renderRate() err = %v, want nil", err)
	}
	if want := `<span class="w-stars-rate-number">—</span>`; !strings.Contains(got, want) {
		t.Errorf("renderRate() = %q, want it to contain %q", got, want)
	}
	if want := `width: 0%`; !strings.Contains(got, want) {
		t.Errorf("renderRate() = %q, want it to contain %q", got, want)
	}
}

func TestStarsWidgetTruncatesTheWidth(t *testing.T) {
	cases := []struct {
		average float64
		want    string
	}{
		{0, "width: 0%"},
		{0.8, "width: 16%"},
		{2.5, "width: 50%"},
		{3.3, "width: 66%"},
		{4.5, "width: 90%"},
		{5, "width: 100%"},
	}
	for _, c := range cases {
		rating := page.Rating{Value: c.average, Votes: 1, Mode: page.RatingModeStars}
		got := starsWidget(module.Env{}, "main", rating, false)
		if !strings.Contains(got, c.want) {
			t.Errorf("starsWidget(%v) = %q, want it to contain %q", c.average, got, c.want)
		}
	}
}
