package modules

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type stubData struct{ module.Data }

func listUsersEnv(t *testing.T, user *db.User) module.Env {
	t.Helper()
	env := forumEnv(t)
	env.User = user
	env.Data = stubData{}
	env.Render = func(source string, _ *page.Context) (string, error) { return source, nil }
	return env
}

func TestListUsersAvatarOfAViewerWhoUploadedOne(t *testing.T) {
	env := listUsersEnv(t, &db.User{ID: 7, Username: "probe", Avatar: "avatars/7.png"})

	got, err := renderListUsers(env, nil, "%%avatar%%")
	if err != nil {
		t.Fatalf("renderListUsers() err = %v, want nil", err)
	}
	if want := "/local--files/avatars/7.png"; got != want {
		t.Errorf("renderListUsers(%%%%avatar%%%%) = %q, want %q", got, want)
	}
}

func TestListUsersAvatarOfAViewerWithoutOne(t *testing.T) {
	env := listUsersEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderListUsers(env, nil, "%%avatar%%")
	if err != nil {
		t.Fatalf("renderListUsers() err = %v, want nil", err)
	}
	if want := "/-/static/images/default_avatar.png"; got != want {
		t.Errorf("renderListUsers(%%%%avatar%%%%) = %q, want %q", got, want)
	}
}

func TestListUsersSkipsAnAnonymousReader(t *testing.T) {
	env := listUsersEnv(t, nil)

	got, err := renderListUsers(env, nil, "%%name%%")
	if err != nil {
		t.Fatalf("renderListUsers() err = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("renderListUsers(%%%%name%%%%) = %q, want %q", got, "")
	}
}

func TestListUsersLeavesAnUnknownVariableAlone(t *testing.T) {
	env := listUsersEnv(t, &db.User{ID: 7, Username: "probe"})

	got, err := renderListUsers(env, nil, "%%nope%%")
	if err != nil {
		t.Fatalf("renderListUsers() err = %v, want nil", err)
	}
	if !strings.Contains(got, "%%nope%%") {
		t.Errorf("renderListUsers(%%%%nope%%%%) = %q, want it kept", got)
	}
}

func TestListUsersChipTokensDoNotOverlap(t *testing.T) {
	chips := map[string]string{chipToken(1): "one", chipToken(10): "ten"}

	got := putChipsBack(chipToken(1)+" "+chipToken(10), chips)
	if want := "one ten"; got != want {
		t.Errorf("putChipsBack() = %q, want %q", got, want)
	}
}

func TestListUsersLeavesTheBodyAloneWithoutChips(t *testing.T) {
	got := putChipsBack("<p>nothing to put back</p>", nil)
	if want := "<p>nothing to put back</p>"; got != want {
		t.Errorf("putChipsBack() = %q, want %q", got, want)
	}
}

func TestListUsersNamesThatSurviveAPageList(t *testing.T) {
	env := listUsersEnv(t, &db.User{ID: 7, Username: "probe", DisplayName: "Probe Person"})

	cases := map[string]string{
		"%%title%%":            "Probe Person",
		"%%user_displayname%%": "Probe Person",
		"%%name%%":             "probe",
		"%%user_name%%":        "probe",
	}
	for body, want := range cases {
		got, err := renderListUsers(env, nil, body)
		if err != nil {
			t.Fatalf("renderListUsers(%s) err = %v, want nil", body, err)
		}
		if got != want {
			t.Errorf("renderListUsers(%s) = %q, want %q", body, got, want)
		}
	}
}

func TestListUsersNamesOfAnAnonymousReader(t *testing.T) {
	env := listUsersEnv(t, nil)

	for _, body := range []string{"%%user_displayname%%", "%%user_name%%"} {
		got, err := renderListUsers(env, map[string]string{"always": "yes", "anonname": "Guest"}, body)
		if err != nil {
			t.Fatalf("renderListUsers(%s) err = %v, want nil", body, err)
		}
		if want := "Guest"; got != want {
			t.Errorf("renderListUsers(%s) = %q, want %q", body, got, want)
		}
	}
}
