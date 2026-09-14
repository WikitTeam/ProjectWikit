package modules

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

type memberData struct {
	module.Data
	all      []db.Member
	role     *roles.Role
	roleErr  error
	lastRole *int64
	offset   int
	limit    int
}

func (d *memberData) Members(roleID *int64, offset, limit int) ([]db.Member, error) {
	d.lastRole, d.offset, d.limit = roleID, offset, limit
	if offset >= len(d.all) {
		return nil, nil
	}
	return d.all[offset:min(offset+limit, len(d.all))], nil
}

func (d *memberData) MemberCount(roleID *int64) (int, error) {
	d.lastRole = roleID
	return len(d.all), nil
}

func (d *memberData) RoleByRef(string) (*roles.Role, error) { return d.role, d.roleErr }

func (d *memberData) RenderUser(u db.User) (string, error) {
	return `<span class="printuser">` + u.Username + `</span>`, nil
}

func membersEnv(t *testing.T, data *memberData) module.Env {
	t.Helper()
	env := forumEnv(t)
	env.Data = data
	env.Render = func(source string, _ *page.Context) (string, error) { return source, nil }
	return env
}

func someMembers(n int) []db.Member {
	out := make([]db.Member, n)
	for i := range out {
		out[i] = db.Member{
			User:     db.User{ID: int64(i + 1), Username: "user" + strconv.Itoa(i+1)},
			JoinedAt: time.Unix(int64(1700000000+i), 0),
		}
	}
	return out
}

func TestMembersNumbersRowsAcrossPages(t *testing.T) {
	data := &memberData{all: someMembers(250)}
	env := membersEnv(t, data)
	env.Page = page.NewContext(nil, nil, page.PathParams{{Key: "p", Value: "2"}}, nil)

	got, err := renderMembers(env, nil, "%%index%%")
	if err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if !strings.Contains(got, "\n101\n") {
		t.Errorf("renderMembers(%%%%index%%%%) page 2 = %q, want it to start at 101", got)
	}
	if data.offset != 100 {
		t.Errorf("Members() offset = %d, want %d", data.offset, 100)
	}
}

func TestMembersDefaultsToOneHundredPerPage(t *testing.T) {
	data := &memberData{all: someMembers(150)}
	env := membersEnv(t, data)

	if _, err := renderMembers(env, nil, "%%number%%"); err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if data.limit != 100 {
		t.Errorf("Members() limit = %d, want %d", data.limit, 100)
	}
}

func TestMembersPerPageParam(t *testing.T) {
	data := &memberData{all: someMembers(150)}
	env := membersEnv(t, data)

	if _, err := renderMembers(env, map[string]string{"perpage": "25"}, "%%number%%"); err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if data.limit != 25 {
		t.Errorf("Members() limit = %d, want %d", data.limit, 25)
	}
}

func TestMembersPerPageIsCapped(t *testing.T) {
	data := &memberData{all: someMembers(10)}
	env := membersEnv(t, data)

	if _, err := renderMembers(env, map[string]string{"perpage": "9000"}, "%%number%%"); err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if data.limit != maxMembersPerPage {
		t.Errorf("Members() limit = %d, want %d", data.limit, maxMembersPerPage)
	}
}

func TestMembersFiltersByRole(t *testing.T) {
	data := &memberData{all: someMembers(3), role: &roles.Role{ID: 42, Slug: "staff"}}
	env := membersEnv(t, data)

	if _, err := renderMembers(env, map[string]string{"role": "staff"}, "%%number%%"); err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if data.lastRole == nil || *data.lastRole != 42 {
		t.Errorf("Members() roleID = %v, want 42", data.lastRole)
	}
}

func TestMembersReportsAnUnknownRole(t *testing.T) {
	data := &memberData{all: someMembers(3), roleErr: db.ErrNotFound}
	env := membersEnv(t, data)

	_, err := renderMembers(env, map[string]string{"role": "nope"}, "%%number%%")
	var moduleErr *module.Error
	if !asModuleError(err, &moduleErr) {
		t.Fatalf("renderMembers() err = %v, want a module error", err)
	}
	if !strings.Contains(moduleErr.Message, "nope") {
		t.Errorf("renderMembers() message = %q, want it to name %q", moduleErr.Message, "nope")
	}
}

func TestMembersRendersTheUserChip(t *testing.T) {
	data := &memberData{all: someMembers(1)}
	env := membersEnv(t, data)

	got, err := renderMembers(env, nil, "%%members%%")
	if err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if want := `<span class="printuser">user1</span>`; !strings.Contains(got, want) {
		t.Errorf("renderMembers(%%%%members%%%%) = %q, want it to contain %q", got, want)
	}
}

func TestMembersTimeIsTheJoinDate(t *testing.T) {
	data := &memberData{all: someMembers(1)}
	env := membersEnv(t, data)

	got, err := renderMembers(env, nil, "%%time%%")
	if err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if want := "[[date 1700000000]]"; !strings.Contains(got, want) {
		t.Errorf("renderMembers(%%%%time%%%%) = %q, want it to contain %q", got, want)
	}
}

func TestMembersOrdersByUserNumber(t *testing.T) {
	data := &memberData{all: someMembers(3)}
	env := membersEnv(t, data)

	got, err := renderMembers(env, nil, "%%number%%")
	if err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if want := "1\n2\n3"; !strings.Contains(got, want) {
		t.Errorf("renderMembers(%%%%number%%%%) = %q, want it to contain %q", got, want)
	}
}

func TestMembersHasNoPagerOnASinglePage(t *testing.T) {
	data := &memberData{all: someMembers(3)}
	env := membersEnv(t, data)

	got, err := renderMembers(env, nil, "%%number%%")
	if err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if strings.Contains(got, `class="pager"`) {
		t.Errorf("renderMembers() = %q, want no pager", got)
	}
}

func TestMembersPagerLinksToTheHostPage(t *testing.T) {
	data := &memberData{all: someMembers(250)}
	env := membersEnv(t, data)
	env.Page = page.NewContext(&db.Article{Category: "_default", Name: "roster"}, nil, nil, nil)

	got, err := renderMembers(env, nil, "%%number%%")
	if err != nil {
		t.Fatalf("renderMembers() err = %v, want nil", err)
	}
	if want := `href="/roster/p/2"`; !strings.Contains(got, want) {
		t.Errorf("renderMembers() = %q, want it to contain %q", got, want)
	}
}
