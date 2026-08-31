package modules

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func formEnv(t *testing.T, user *db.User) module.Env {
	t.Helper()
	env := forumEnv(t)
	env.User = user
	env.Render = func(source string, _ *page.Context) (string, error) { return source, nil }
	env.Page = page.NewContext(&db.Article{Category: "_default", Name: "join"}, nil, nil, user)
	env.Page.CSRF = "abcdefghijklmnopqrstuvwxyz012345"
	return env
}

func TestApplicationFormNeedsALogin(t *testing.T) {
	env := formEnv(t, nil)

	got, err := renderApplicationForm(env, nil, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if strings.Contains(got, "<form") {
		t.Errorf("renderApplicationForm(anonymous) = %q, want no form", got)
	}
}

func TestApplicationFormDefaultsToATicket(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, nil, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `name="kind" value="ticket"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm() = %q, want it to contain %q", got, want)
	}
}

func TestApplicationFormTakesTheMembershipKind(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, map[string]string{"type": "membershipapply"}, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `name="kind" value="membershipapply"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm(type=membershipapply) = %q, want it to contain %q", got, want)
	}
}

func TestApplicationFormCarriesTheToken(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, nil, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `name="csrfmiddlewaretoken" value="abcdefghijklmnopqrstuvwxyz012345"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm() = %q, want it to contain %q", got, want)
	}
}

func TestApplicationFormNamesTheHostPage(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, nil, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `name="page" value="join"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm() = %q, want it to contain %q", got, want)
	}
}

func TestApplicationFormShowsTheOutcome(t *testing.T) {
	user := &db.User{ID: 7, Username: "probe"}
	env := formEnv(t, user)
	env.Page = page.NewContext(&db.Article{Category: "_default", Name: "join"},
		nil, page.PathParams{{Key: "applied", Value: "ok"}}, user)

	got, err := renderApplicationForm(env, nil, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if !strings.Contains(got, "success-block") {
		t.Errorf("renderApplicationForm(applied/ok) = %q, want a success block", got)
	}
}

func TestMembershipByPasswordIsSilentWhenOff(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})
	env.Site = &db.Site{}

	got, err := renderMembershipByPassword(env, nil, "")
	if err != nil {
		t.Fatalf("renderMembershipByPassword() err = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("renderMembershipByPassword(disabled) = %q, want %q", got, "")
	}
}

func TestMembershipByPasswordAsksForThePassword(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})
	env.Site = &db.Site{MembershipPasswordEnabled: true}

	got, err := renderMembershipByPassword(env, nil, "")
	if err != nil {
		t.Fatalf("renderMembershipByPassword() err = %v, want nil", err)
	}
	if want := `type="password" name="password"`; !strings.Contains(got, want) {
		t.Errorf("renderMembershipByPassword() = %q, want it to contain %q", got, want)
	}
}

func TestMembershipByPasswordNeedsALogin(t *testing.T) {
	env := formEnv(t, nil)
	env.Site = &db.Site{MembershipPasswordEnabled: true}

	got, err := renderMembershipByPassword(env, nil, "")
	if err != nil {
		t.Fatalf("renderMembershipByPassword() err = %v, want nil", err)
	}
	if strings.Contains(got, "<form") {
		t.Errorf("renderMembershipByPassword(anonymous) = %q, want no form", got)
	}
}

func TestApplicationFormShowsTheSubjectInputByDefault(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, nil, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `type="text" name="subject"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm() = %q, want it to contain %q", got, want)
	}
}

func TestApplicationFormTitleNoDropsTheInput(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, map[string]string{"title": "no"}, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if strings.Contains(got, `name="subject"`) {
		t.Errorf("renderApplicationForm(title=no) = %q, want no subject field at all", got)
	}
}

func TestApplicationFormTitleYesKeepsTheInput(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, map[string]string{"title": "yes"}, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `type="text" name="subject"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm(title=yes) = %q, want it to contain %q", got, want)
	}
}

func TestApplicationFormTitleWritesTheSubject(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, map[string]string{"title": "入组申请"}, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `type="hidden" name="subject" value="入组申请"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm(title=...) = %q, want it to contain %q", got, want)
	}
	if strings.Contains(got, `type="text" name="subject"`) {
		t.Errorf("renderApplicationForm(title=...) = %q, want no subject input", got)
	}
}

func TestApplicationFormSubjectParamIsAnAlias(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderApplicationForm(env, map[string]string{"subject": "报修"}, "")
	if err != nil {
		t.Fatalf("renderApplicationForm() err = %v, want nil", err)
	}
	if want := `type="hidden" name="subject" value="报修"`; !strings.Contains(got, want) {
		t.Errorf("renderApplicationForm(subject=...) = %q, want it to contain %q", got, want)
	}
}

func TestMembershipByPasswordLabelDefaults(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})
	env.Site = &db.Site{MembershipPasswordEnabled: true}

	got, err := renderMembershipByPassword(env, nil, "")
	if err != nil {
		t.Fatalf("renderMembershipByPassword() err = %v, want nil", err)
	}
	if want := "输入密码"; !strings.Contains(got, want) {
		t.Errorf("renderMembershipByPassword() = %q, want it to contain %q", got, want)
	}
}

func TestMembershipByPasswordTakesALabel(t *testing.T) {
	env := formEnv(t, &db.User{ID: 7, Username: "probe"})
	env.Site = &db.Site{MembershipPasswordEnabled: true}

	got, err := renderMembershipByPassword(env, map[string]string{"label": "口令"}, "")
	if err != nil {
		t.Fatalf("renderMembershipByPassword() err = %v, want nil", err)
	}
	if want := "<td>口令</td>"; !strings.Contains(got, want) {
		t.Errorf("renderMembershipByPassword(label=口令) = %q, want it to contain %q", got, want)
	}
	if strings.Contains(got, "输入密码") {
		t.Errorf("renderMembershipByPassword(label=口令) = %q, want the default label gone", got)
	}
}
