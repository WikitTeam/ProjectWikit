// Package module is what a wikidot module is written against. The modules
// themselves live in internal/modules, one file each.
package module

import (
	"errors"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

// Error is the failure that reaches the reader as a block on the page rather
// than as a broken response.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

// ErrNotPorted is not an Error on purpose. A module nobody has written yet is
// a hole in pwikit, not something to tell the reader about.
var ErrNotPorted = errors.New("module: not ported yet")

type Data interface {
	TagArticles(categorySlug, name string, hiddenCategories []string) ([]db.Article, error)
	TagCategory(slug string) (db.TagCategory, error)
	HiddenCategories(user *db.User) ([]string, error)
}

type Env struct {
	Page *page.Context
	Loc  *i18n.Localizer
	Site *db.Site
	User *db.User
	Data Data
}

func (e Env) Text(id string, args ...any) string {
	if e.Loc == nil {
		return id
	}
	return e.Loc.T(id, args...)
}

func (e Env) MediaDomain() string {
	if e.Site == nil {
		return ""
	}
	return e.Site.MediaDomain
}

type Renderer func(env Env, params map[string]string, body string) (string, error)

var renderers = map[string]Renderer{}

// Register is called from each module's own file. A name the registry does not
// know is a typo, so it panics rather than registering something unreachable.
func Register(name string, r Renderer) {
	if _, ok := registry[name]; !ok {
		panic("module: " + name + " is not in the registry")
	}
	renderers[name] = r
}

// Render answers one [[module]]. The page's own parameters go underneath the
// module's own, which is what lets a URL override what the page asked for.
func Render(env Env, name string, params map[string]string, body string) (string, error) {
	if env.Page != nil && env.Page.PathParams["nomodule"] == "true" {
		return "", &Error{Message: env.Text("module-disabled")}
	}
	info, ok := Lookup(name)
	if !ok || info.Removed {
		return "", &Error{Message: env.Text("module-unknown", "name", name)}
	}
	render, ok := renderers[info.Name]
	if !ok {
		return "", ErrNotPorted
	}
	return render(env, merge(env.Page, params), body)
}

func merge(ctx *page.Context, params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	if ctx != nil {
		for key, value := range ctx.PathParams {
			out[key] = value
		}
	}
	for key, value := range params {
		out[key] = value
	}
	return out
}

// BoolParam is get_boolean_param, where anything the list does not name reads
// as the default rather than as false.
func BoolParam(params map[string]string, key string, def bool) bool {
	value, ok := params[key]
	if !ok {
		return def
	}
	switch strings.ToLower(value) {
	case "true", "t", "1", "yes":
		return true
	case "false", "f", "0", "no":
		return false
	}
	return def
}
