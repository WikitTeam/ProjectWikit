package page

import (
	"net/http"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

// Context is the state one render writes back to the page around it. Modules
// reach it through the render callback, which is why a module can set the HTTP
// status or redirect the whole page.
type Context struct {
	Article       *db.Article
	SourceArticle *db.Article
	PathParams    PathParams
	User          *db.User

	// The handler owns this because it is the only layer that can also plant
	// the cookie a form needs.
	CSRF string

	// An empty list means every module, which is what an article gets.
	OnlyModules []string

	Title      string
	Status     int
	RedirectTo string

	Depth int

	AddCSS        string
	ComputedStyle string
	OGDescription string
	OGImage       string
}

func NewContext(article, sourceArticle *db.Article, pathParams PathParams, user *db.User) *Context {
	c := &Context{
		Article:       article,
		SourceArticle: sourceArticle,
		PathParams:    pathParams,
		User:          user,
		Status:        http.StatusOK,
	}
	if article != nil {
		c.Title = article.Title
	}
	return c
}

// CloneWith starts a nested render. Only the three fields listed carry over:
// the styles and the Open Graph fields belong to the page that produced them,
// and Merge is what brings the styles back.
func (c *Context) CloneWith(article, sourceArticle *db.Article, pathParams PathParams, user *db.User) *Context {
	clone := NewContext(article, sourceArticle, pathParams, user)
	clone.Depth = c.Depth + 1
	clone.Status = c.Status
	clone.RedirectTo = c.RedirectTo
	clone.Title = c.Title
	clone.CSRF = c.CSRF
	clone.OnlyModules = c.OnlyModules
	return clone
}

// Merge folds a nested render back. ComputedStyle accumulates; the rest is
// overwritten, so the innermost render decides the status and the title.
func (c *Context) Merge(other *Context) {
	c.Status = other.Status
	c.RedirectTo = other.RedirectTo
	c.ComputedStyle += other.ComputedStyle
	c.Title = other.Title
}
