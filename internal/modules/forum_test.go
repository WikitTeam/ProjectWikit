package modules

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func forumEnv(t *testing.T) module.Env {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return module.Env{Loc: bundle.Localizer(i18n.DefaultLanguage)}
}

func TestForumURLsNormalizeTheName(t *testing.T) {
	if got, want := forumSectionURL(3, "Probe Open"), "/forum/s-3/probe-open"; got != want {
		t.Errorf("forumSectionURL(3, %q) = %q, want %q", "Probe Open", got, want)
	}
	if got, want := forumCategoryURL(7, "Probe Chat"), "/forum/c-7/probe-chat"; got != want {
		t.Errorf("forumCategoryURL(7, %q) = %q, want %q", "Probe Chat", got, want)
	}
	if got, want := forumPostURL(2, "Probe Thread", 9), "/forum/t-2/probe-thread#post-9"; got != want {
		t.Errorf("forumPostURL(2, %q, 9) = %q, want %q", "Probe Thread", got, want)
	}
}

func TestRenderDateIsUTC(t *testing.T) {
	at := time.Date(2023, 9, 10, 11, 12, 13, 0, time.FixedZone("east", 8*3600))
	want := `<span class="odate w-date" style="display: inline" data-timestamp="1694315533000" ` +
		`data-format="%m.%d.%Y %H:%M">09.10.2023 03:12</span>`
	if got := renderDate(forumEnv(t), at); got != want {
		t.Errorf("renderDate(%s) = %q, want %q", at, got, want)
	}
}

func TestServerDatePadsEveryField(t *testing.T) {
	at := time.Date(2021, 3, 4, 5, 6, 7, 0, time.UTC)
	if got, want := serverDate(forumEnv(t), at), "03.04.2021 05:06"; got != want {
		t.Errorf("serverDate(%s) = %q, want %q", at, got, want)
	}
}
