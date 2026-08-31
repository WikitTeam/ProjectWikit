package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
)

func init() { module.Register("listusers", renderListUsers) }

func renderListUsers(env module.Env, params map[string]string, body string) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "listusers")}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	body = strings.TrimSpace(body)
	var filled []string
	chips := map[string]string{}
	if module.BoolParam(params, "authors", false) {
		var err error
		if filled, err = authorTemplates(env, pc, body, chips); err != nil {
			return "", err
		}
	} else {
		if env.User == nil && !module.BoolParam(params, "always", false) {
			return "", nil
		}
		filled = append(filled, page.ApplyTemplate(body, viewerVars(env, params)))
	}

	html, err := env.Render(strings.Join(filled, " "), pc)
	if err != nil {
		return "", err
	}
	return putChipsBack(html, chips), nil
}

// A user chip is markup, and wikitext escapes markup, so the body goes through
// the renderer carrying a plain token where each chip belongs.
func putChipsBack(html string, chips map[string]string) string {
	if len(chips) == 0 {
		return html
	}
	pairs := make([]string, 0, 2*len(chips))
	for token, chip := range chips {
		pairs = append(pairs, token, chip)
	}
	return strings.NewReplacer(pairs...).Replace(html)
}

// The index sits inside the token rather than at its end, so one token is never
// the beginning of another once there are ten authors.
func chipToken(i int) string { return "pwikitauthor" + strconv.Itoa(i) + "chip" }

func authorTemplates(env module.Env, pc *page.Context, body string, chips map[string]string) ([]string, error) {
	if pc.Article == nil {
		return nil, nil
	}
	authors, err := env.Data.ArticleAuthors(pc.Article.ID)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(authors))
	for i := range authors {
		author := authors[i]
		// The chip is only rendered when the body asks for it, since building
		// one costs a roles lookup per author.
		var chipErr error
		out = append(out, page.ApplyTemplate(body, func(name string) (string, bool) {
			switch strings.ToLower(name) {
			case "author":
				return author.DisplayLabel(), true
			case "author_linked":
				token := chipToken(i)
				if _, done := chips[token]; !done {
					chip, err := env.Data.RenderUser(author)
					if err != nil {
						chipErr = err
						return "", true
					}
					chips[token] = chip
				}
				return token, true
			}
			return "", false
		}))
		if chipErr != nil {
			return nil, chipErr
		}
	}
	return out, nil
}

func viewerVars(env module.Env, params map[string]string) func(name string) (string, bool) {
	user := env.User
	number := "-1"
	name := params["anonname"]
	if _, ok := params["anonname"]; !ok {
		name = env.Text("module-listusers-anonymous")
	}
	title := name
	avatar := printuser.DefaultAvatar
	if user != nil {
		number = strconv.FormatInt(user.ID, 10)
		name = user.Username
		// The display name rather than the account name, which is what every
		// other place a reader sees a person shows.
		title = firstNonBlank(user.DisplayName, user.Username)
		if user.Avatar != "" {
			avatar = "/local--files/" + user.Avatar
		}
	}

	return func(key string) (string, bool) {
		switch strings.ToLower(key) {
		case "number":
			return number, true
		case "title":
			return title, true
		case "name":
			return name, true
		case "avatar":
			return avatar, true
		case "is_authenticated":
			return strconv.FormatBool(user != nil), true
		}
		return "", false
	}
}
