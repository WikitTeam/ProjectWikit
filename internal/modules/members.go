package modules

import (
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("members", renderMembers) }

const (
	defaultMembersPerPage = 100
	maxMembersPerPage     = 500
)

func renderMembers(env module.Env, params map[string]string, body string) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "members")}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	var roleID *int64
	if ref := strings.TrimSpace(params["role"]); ref != "" {
		role, err := env.Data.RoleByRef(ref)
		if err != nil {
			return "", &module.Error{Message: env.Text("module-members-no-role", "name", ref)}
		}
		roleID = &role.ID
	}

	perPage := membersPerPage(params)
	total, err := env.Data.MemberCount(roleID)
	if err != nil {
		return "", err
	}
	totalPages := (total + perPage - 1) / perPage
	current := membersPage(pc.PathParams, totalPages)

	members, err := env.Data.Members(roleID, (current-1)*perPage, perPage)
	if err != nil {
		return "", err
	}

	chips := map[string]string{}
	rows := make([]string, 0, len(members))
	for i := range members {
		filled, err := memberRow(env, body, members[i], (current-1)*perPage+i+1, total, chips)
		if err != nil {
			return "", err
		}
		rows = append(rows, filled)
	}

	html, err := env.Render(strings.Join(rows, "\n"), pc)
	if err != nil {
		return "", err
	}
	html = putChipsBack(html, chips)

	basePath := "#"
	if pc.Article != nil {
		basePath = listpages.BasePath(pc.Article.FullName(), pc.PathParams)
	}
	pager := listpages.Pagination(env.Loc, basePath, current, totalPages)
	if pager == "" {
		return `<div class="members-list">` + "\n" + html + "\n</div>", nil
	}
	return `<div class="members-list">` + "\n" + html + "\n" + pager + "\n</div>", nil
}

func memberRow(env module.Env, body string, m db.Member, index, total int, chips map[string]string) (string, error) {
	var chipErr error
	filled := page.ApplyTemplate(strings.TrimSpace(body), func(name string) (string, bool) {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "members", "member":
			// A user chip is markup, and the row still has to go through the
			// renderer, so a plain token stands in until the markup is back.
			token := chipToken(index)
			if _, done := chips[token]; !done {
				chip, err := env.Data.RenderUser(m.User)
				if err != nil {
					chipErr = err
					return "", true
				}
				chips[token] = chip
			}
			return token, true
		case "time":
			return memberJoined(m.JoinedAt), true
		case "number":
			return strconv.FormatInt(m.ID, 10), true
		case "name":
			return m.Username, true
		case "title":
			return firstNonBlank(m.DisplayName, m.Username), true
		case "index":
			return strconv.Itoa(index), true
		case "total":
			return strconv.Itoa(total), true
		}
		return "", false
	})
	return filled, chipErr
}

func memberJoined(at time.Time) string {
	return "[[date " + strconv.FormatInt(at.Unix(), 10) + "]]"
}

func membersPerPage(params map[string]string) int {
	raw, ok := params["perpage"]
	if !ok {
		raw = params["limit"]
	}
	n, err := wikinum.Int(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return defaultMembersPerPage
	}
	return min(n, maxMembersPerPage)
}

func membersPage(path page.PathParams, totalPages int) int {
	n, err := wikinum.Int(path.Get("p"))
	if err != nil || n < 1 {
		return 1
	}
	if totalPages > 0 && n > totalPages {
		return totalPages
	}
	return n
}
