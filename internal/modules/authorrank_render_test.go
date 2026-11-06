package modules

import (
	"strconv"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type rankData struct {
	module.Data
	rows  []db.AuthorRankRow
	users map[int64]*db.User
}

func (d *rankData) HiddenCategories(*db.User) ([]string, error) { return nil, nil }

func (d *rankData) AuthorRank(_ db.AuthorRankFilter, offset, limit int) ([]db.AuthorRankRow, error) {
	if offset >= len(d.rows) {
		return nil, nil
	}
	return d.rows[offset:min(offset+limit, len(d.rows))], nil
}

func (d *rankData) AuthorRankOf(_ db.AuthorRankFilter, userID int64) (*db.AuthorRankRow, error) {
	for i := range d.rows {
		if d.rows[i].UserID == userID {
			return &d.rows[i], nil
		}
	}
	return nil, nil
}

func (d *rankData) UserByID(id int64) (*db.User, error) {
	if u, ok := d.users[id]; ok {
		return u, nil
	}
	return nil, db.ErrNotFound
}

func (d *rankData) find(match func(*db.User) bool) (*db.User, error) {
	for _, u := range d.users {
		if match(u) {
			return u, nil
		}
	}
	return nil, db.ErrNotFound
}

func (d *rankData) UserByUsername(name string) (*db.User, error) {
	return d.find(func(u *db.User) bool { return u.Username == name })
}

func (d *rankData) UserByDisplayName(name string) (*db.User, error) {
	return d.find(func(u *db.User) bool { return u.DisplayName == name })
}

func (d *rankData) RenderUserByID(id *int64) (string, error) {
	return "<user " + strconv.FormatInt(*id, 10) + ">", nil
}

func someRanks() *rankData {
	d := &rankData{users: map[int64]*db.User{}}
	for i := 1; i <= 3; i++ {
		id := int64(i)
		d.users[id] = &db.User{ID: id, Username: "user" + strconv.Itoa(i), DisplayName: "Shown" + strconv.Itoa(i)}
		d.rows = append(d.rows, db.AuthorRankRow{UserID: id, Score: float64(40 - 10*i), Pages: i, Place: i, Pos: i, Total: 3})
	}
	d.users[9] = &db.User{ID: 9, Username: "nobody"}
	return d
}

func rankEnv(t *testing.T, data *rankData) module.Env {
	t.Helper()
	env := forumEnv(t)
	env.Data = data
	env.Render = func(source string, _ *page.Context) (string, error) { return source, nil }
	return env
}

func TestAuthorRankDefaultTable(t *testing.T) {
	got, err := renderAuthorRank(rankEnv(t, someRanks()), map[string]string{}, "")
	if err != nil {
		t.Fatalf("renderAuthorRank() err = %v, want nil", err)
	}
	if !strings.Contains(got, "||~ ") {
		t.Errorf("renderAuthorRank() = %q, want a header row", got)
	}
	if !strings.Contains(got, "||1||<user 1>||+30||1||") {
		t.Errorf("renderAuthorRank() = %q, want the first row", got)
	}
}

func TestAuthorRankTemplateAndLines(t *testing.T) {
	params := map[string]string{"prependline": "||~ N||~ Who||", "appendline": "of %%total%%"}
	got, err := renderAuthorRank(rankEnv(t, someRanks()), params, "||%%index%%||%%author%%/%%author_name%%||")
	if err != nil {
		t.Fatalf("renderAuthorRank() err = %v, want nil", err)
	}
	want := "||~ N||~ Who||\n||1||Shown1/user1||\n||2||Shown2/user2||\n||3||Shown3/user3||\nof 3"
	if !strings.Contains(got, want) {
		t.Errorf("renderAuthorRank() = %q, want it to contain %q", got, want)
	}
}

func TestAuthorRankUserVariables(t *testing.T) {
	params := map[string]string{"user": "Shown2", "rows": "no",
		"prependline": "%%user_linked%% %%user_rank%% %%user_score%% %%user_pages%% %%user_index%% %%user_ranked%%"}
	got, err := renderAuthorRank(rankEnv(t, someRanks()), params, "||%%rank%%||")
	if err != nil {
		t.Fatalf("renderAuthorRank() err = %v, want nil", err)
	}
	if !strings.Contains(got, "<user 2> 2 +20 2 2 true") {
		t.Errorf("renderAuthorRank(user=Shown2) = %q, want the user's place", got)
	}
	if strings.Contains(got, "||1||") {
		t.Errorf("renderAuthorRank(rows=no) = %q, want no rows", got)
	}
}

func TestAuthorRankUnrankedUser(t *testing.T) {
	params := map[string]string{"user": "nobody", "rows": "no", "prependline": "[%%user_rank%%] %%user_ranked%%"}
	got, err := renderAuthorRank(rankEnv(t, someRanks()), params, "")
	if err != nil {
		t.Fatalf("renderAuthorRank() err = %v, want nil", err)
	}
	if !strings.Contains(got, "[] false") {
		t.Errorf("renderAuthorRank(user=nobody) = %q, want an empty place", got)
	}
}

func TestAuthorRankIsUser(t *testing.T) {
	got, err := renderAuthorRank(rankEnv(t, someRanks()), map[string]string{"user": "user3"}, "%%rank%%:%%is_user%%")
	if err != nil {
		t.Fatalf("renderAuthorRank() err = %v, want nil", err)
	}
	if !strings.Contains(got, "1:false\n2:false\n3:true") {
		t.Errorf("renderAuthorRank(user=user3) = %q, want only the third row marked", got)
	}
}

func TestAuthorRankUnknownUser(t *testing.T) {
	got, err := renderAuthorRank(rankEnv(t, someRanks()), map[string]string{"user": "ghost"}, "")
	if err != nil {
		t.Fatalf("renderAuthorRank() err = %v, want nil", err)
	}
	if !strings.Contains(got, "ghost") {
		t.Errorf("renderAuthorRank(user=ghost) = %q, want the name in the error", got)
	}
}
