package articlepage

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/article"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

type result struct {
	Status   int
	Location string
	Body     string
}

type request struct {
	ctx  context.Context
	loc  *i18n.Localizer
	site *db.Site
	user *db.User

	name    string
	params  article.Params
	encoded string

	article   *db.Article
	forbidden bool
	perms     perms.Set
}

func (h *Handler) build(r *http.Request) (*result, error) {
	ctx := r.Context()
	found := site.FromContext(ctx)
	if found == nil {
		return nil, errors.New("articlepage: the request carries no site")
	}

	req := &request{
		ctx:  ctx,
		loc:  h.deps.Bundle.Localizer(i18n.DefaultLanguage),
		site: found,
		user: auth.FromContext(ctx),
	}
	req.name, req.params = article.ParsePath(strings.TrimPrefix(r.URL.EscapedPath(), "/"), found.HomePage)
	req.encoded = req.params.Encode()

	if target, ok := article.RedirectTarget(req.name, req.params); ok {
		return &result{Status: http.StatusFound, Location: target}, nil
	}

	if err := h.load(req); err != nil {
		return nil, err
	}

	if req.article != nil && req.params.Get("comments") == "show" {
		target, err := h.commentsRedirect(req)
		if err != nil {
			return nil, err
		}
		if target != "" {
			return &result{Status: http.StatusFound, Location: target}, nil
		}
	}

	return h.render(req)
}

// A page the visitor may not see is dropped here, which is what leaves every
// layer below working on nothing at all.
func (h *Handler) load(req *request) error {
	found, err := h.deps.DB.ArticleByName(req.ctx, req.name)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return err
	}
	req.article = found

	perm := repo.NewPerms(req.ctx, h.deps.DB)
	subject, err := perm.Subject(req.user, h.now())
	if err != nil {
		return err
	}

	object, checked, err := h.permsObject(req, perm)
	if err != nil {
		return err
	}
	req.perms = perms.Resolve(subject, object)
	if checked && !req.perms.Has(perms.ViewArticles) {
		req.forbidden = true
		req.article = nil
	}
	return nil
}

// permsObject reports checked=false when neither the page nor a row for its
// category exists, where Django asks nothing and lets the response be a 404.
func (h *Handler) permsObject(req *request, perm *repo.Perms) (*perms.Object, bool, error) {
	if req.article != nil {
		object, err := perm.Article(req.article, req.user)
		return object, true, err
	}
	category, _ := wikidot.Split(req.name)
	exists, err := h.deps.DB.CategoryExists(req.ctx, category)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	object, err := perm.Category(category)
	return object, true, err
}

// Django creates the thread on the way past. This only reads, so a page nobody
// has commented on stays where it is.
func (h *Handler) commentsRedirect(req *request) (string, error) {
	info, err := h.deps.DB.CommentInfo(req.ctx, req.article.ID)
	if err != nil {
		return "", err
	}
	if info.ThreadID == 0 {
		return "", nil
	}
	return "/forum/t-" + strconv.FormatInt(info.ThreadID, 10) + "/" +
		wikidot.Normalize(req.article.DisplayName()), nil
}

func (h *Handler) canonicalURL(req *request) string {
	name := req.name
	if req.article != nil {
		name = req.article.FullName()
	}
	return "https://" + req.site.Domain + "/" + name + req.encoded
}
