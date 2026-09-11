package modules

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
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

func TestRenderDateWithoutASiteIsUTC(t *testing.T) {
	at := time.Date(2023, 9, 10, 11, 12, 13, 0, time.FixedZone("east", 8*3600))
	want := `<span class="odate w-date" style="display: inline" data-timestamp="1694315533000" ` +
		`data-format="%m.%d.%Y %H:%M">09.10.2023 03:12 (UTC)</span>`
	if got := renderDate(forumEnv(t), at); got != want {
		t.Errorf("renderDate(%s) = %q, want %q", at, got, want)
	}
}

func TestServerDateUsesTheSiteZone(t *testing.T) {
	env := forumEnv(t)
	env.Site = &db.Site{TimeZone: "Asia/Shanghai"}
	at := time.Date(2021, 3, 4, 20, 6, 7, 0, time.UTC)
	if got, want := serverDate(env, at), "03.05.2021 04:06 (UTC+08:00)"; got != want {
		t.Errorf("serverDate(%s) = %q, want %q", at, got, want)
	}
}

func TestServerDatePadsEveryField(t *testing.T) {
	at := time.Date(2021, 3, 4, 5, 6, 7, 0, time.UTC)
	if got, want := serverDate(forumEnv(t), at), "03.04.2021 05:06 (UTC)"; got != want {
		t.Errorf("serverDate(%s) = %q, want %q", at, got, want)
	}
}
