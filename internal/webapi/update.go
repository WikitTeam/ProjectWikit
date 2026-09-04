package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

// A key that is present and null does not mean the same as a key that is
// absent, so the body is kept as fields until each one is asked for.
type fields map[string]json.RawMessage

func (f fields) has(key string) bool { _, ok := f[key]; return ok }

func (f fields) text(key string) (string, bool) {
	var out string
	if raw, ok := f[key]; ok && json.Unmarshal(raw, &out) == nil {
		return out, true
	}
	return "", false
}

func (f fields) flag(key string) (bool, bool) {
	var out bool
	if raw, ok := f[key]; ok && json.Unmarshal(raw, &out) == nil {
		return out, true
	}
	return false, false
}

func (f fields) list(key string) ([]string, bool) {
	var out []string
	if raw, ok := f[key]; ok && json.Unmarshal(raw, &out) == nil {
		return out, true
	}
	return nil, false
}

type editor struct {
	handler *Articles
	req     *http.Request
	loc     *i18n.Localizer
	site    *db.Site
	user    *db.User
	userID  *int64
	subject perms.Subject
	perm    *repo.Perms
	at      time.Time

	article *db.Article
	body    fields
}

func (h *Articles) update(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		return field("error", loc.T("api-csrf-failed")), http.StatusForbidden, nil
	}

	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	raw, err := readBody(r)
	if err != nil {
		return field("error", loc.T("api-bad-request")), http.StatusBadRequest, nil
	}
	var body fields
	if err := json.Unmarshal(raw, &body); err != nil {
		return field("error", loc.T("api-bad-json")), http.StatusBadRequest, nil
	}
	if bad := validateEdit(body); bad != nil {
		return field("error", loc.T(bad.key)), bad.status, nil
	}

	user := auth.FromContext(ctx)
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return "", 0, err
	}
	e := &editor{
		handler: h, req: r, loc: loc, site: current, user: user,
		perm: perm, subject: subject, at: time.Now().UTC(),
		article: article, body: body,
	}
	if user != nil {
		e.userID = &user.ID
	}

	if bad, err := e.apply(); err != nil {
		return "", 0, err
	} else if bad != nil {
		return field("error", loc.T(bad.key)), bad.status, nil
	}
	return e.answer()
}

func readBody(r *http.Request) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, bodyLimit+1))
	if err != nil || len(raw) == 0 || len(raw) > bodyLimit {
		return nil, errors.New("webapi: the body is missing or too long")
	}
	return raw, nil
}

func validateEdit(body fields) *refusal {
	id, ok := body.text("pageId")
	if !ok || id == "" || !wikidot.NameAllowed(id) {
		return ptr(refuse("api-bad-page-id", http.StatusBadRequest))
	}
	if body.has("source") {
		source, ok := body.text("source")
		if !ok || strings.TrimSpace(source) == "" {
			return ptr(refuse("api-missing-source", http.StatusBadRequest))
		}
		if len(source) > sourceLimit {
			return ptr(refuse("api-source-too-long", http.StatusBadRequest))
		}
	}
	if body.has("title") {
		if _, ok := body.text("title"); !ok {
			return ptr(refuse("api-missing-title", http.StatusBadRequest))
		}
	}
	return nil
}

// The order the pieces run in is the order the page's own history will show
// them, so a rename lands before the edit that follows it.
func (e *editor) apply() (*refusal, error) {
	for _, step := range []func() (*refusal, error){
		e.rename, e.retitle, e.rewrite, e.retag, e.reparent, e.relock, e.recredit,
	} {
		if bad, err := step(); err != nil || bad != nil {
			return bad, err
		}
	}
	return nil, nil
}

func (e *editor) may(permission string) bool {
	object, err := e.perm.Article(e.article, e.user)
	if err != nil {
		return false
	}
	return perms.Resolve(e.subject, object).Has(permission)
}

func (e *editor) rename() (*refusal, error) {
	asked, _ := e.body.text("pageId")
	from := e.article.FullName()
	if strings.EqualFold(asked, from) {
		return nil, nil
	}
	wanted := wikidot.Normalize(asked)
	if strings.EqualFold(wanted, from) {
		return nil, nil
	}

	category, _ := wikidot.Split(wanted)
	target, err := e.perm.Category(category)
	if err != nil {
		return nil, err
	}
	if !e.may(perms.MoveArticles) || !perms.Resolve(e.subject, target).Has(perms.MoveArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}

	forced, _ := e.body.flag("forcePageId")
	taken, err := e.handler.deps.DB.ArticleByName(e.req.Context(), wanted)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	if taken != nil && taken.ID != e.article.ID && !forced {
		return ptr(refuse("api-page-exists", http.StatusConflict)), nil
	}
	free, err := e.free(wanted)
	if err != nil {
		return nil, err
	}

	newCategory, newName := wikidot.Split(free)
	rev, err := e.handler.deps.DB.RenameArticle(e.req.Context(), e.article.ID,
		newCategory, newName, from, e.userID, e.at)
	if err != nil {
		return nil, err
	}
	e.article.Category, e.article.Name = newCategory, newName
	return nil, e.announce(rev)
}

// Without a free name the move would fail on the unique constraint.
func (e *editor) free(wanted string) (string, error) {
	for i := 1; ; i++ {
		candidate := wanted
		if i > 1 {
			candidate = wanted + "-" + strconv.Itoa(i)
		}
		found, err := e.handler.deps.DB.ArticleByName(e.req.Context(), candidate)
		if errors.Is(err, db.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		if found.ID == e.article.ID {
			return candidate, nil
		}
	}
}

func (e *editor) retitle() (*refusal, error) {
	title, ok := e.body.text("title")
	if !ok || title == e.article.Title {
		return nil, nil
	}
	if !e.may(perms.EditArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}
	rev, err := e.handler.deps.DB.UpdateArticleTitle(e.req.Context(), e.article.ID, title, e.userID, e.at)
	if err != nil {
		return nil, err
	}
	e.article.Title = title
	return nil, e.announce(rev)
}

func (e *editor) rewrite() (*refusal, error) {
	source, ok := e.body.text("source")
	if !ok {
		return nil, nil
	}
	ctx := e.req.Context()
	previous, err := e.handler.deps.DB.LatestSource(ctx, e.article.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	if source == previous {
		return nil, nil
	}
	if !e.may(perms.EditArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}

	comment, _ := e.body.text("comment")
	rev, err := e.handler.deps.DB.CreateArticleVersion(ctx, db.VersionWrite{
		ArticleID: e.article.ID,
		Source:    source,
		UserID:    e.userID,
		Kind:      db.LogSource,
		Comment:   comment,
		At:        e.at,
	})
	if err != nil {
		return nil, err
	}
	if err := e.handler.refreshLinks(e.req, e.site, e.article.ID, source); err != nil {
		return nil, err
	}
	return nil, e.announce(rev)
}

func (e *editor) retag() (*refusal, error) {
	tags, ok := e.body.list("tags")
	if !ok {
		return nil, nil
	}
	if !e.may(perms.TagArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}
	allow, err := e.mayCreateTags()
	if err != nil {
		return nil, err
	}
	rev, _, err := e.handler.deps.DB.SetArticleTags(e.req.Context(), e.article.ID, tags, allow, e.userID, e.at)
	if err != nil {
		return nil, err
	}
	return nil, e.announce(rev)
}

func (e *editor) mayCreateTags() (bool, error) {
	ctx := e.req.Context()
	siteMode, err := e.handler.deps.DB.SiteCanCreateTags(ctx, e.site.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return false, err
	}
	own, err := e.handler.deps.DB.CategoryCanCreateTags(ctx, e.article.Category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return false, err
	}
	return db.CreatingTagsAllowed(siteMode, own), nil
}

func (e *editor) reparent() (*refusal, error) {
	if !e.body.has("parent") {
		return nil, nil
	}
	if !e.may(perms.EditArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}
	ctx := e.req.Context()

	wanted, _ := e.body.text("parent")
	var parentID *int64
	if wanted != "" {
		parent, err := e.handler.deps.DB.ArticleByName(ctx, wikidot.Normalize(wanted))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		if parent != nil {
			parentID = &parent.ID
		}
	}
	if samePointer(e.article.ParentID, parentID) {
		return nil, nil
	}

	previous, err := e.parentName(e.article.ParentID)
	if err != nil {
		return nil, err
	}
	meta, err := json.Marshal(map[string]any{
		"parent": nullable(wanted), "prev_parent": nullable(previous),
		"parent_id": parentID, "prev_parent_id": e.article.ParentID,
	})
	if err != nil {
		return nil, err
	}
	rev, err := e.handler.deps.DB.SetArticleParent(ctx, e.article.ID, parentID, e.userID, string(meta), e.at)
	if err != nil {
		return nil, err
	}
	if err := e.announce(rev); err != nil {
		return nil, err
	}
	e.article.ParentID = parentID
	return nil, nil
}

func (e *editor) parentName(id *int64) (string, error) {
	return e.handler.parentName(e.req.Context(), id)
}

func (h *Articles) parentName(ctx context.Context, id *int64) (string, error) {
	if id == nil {
		return "", nil
	}
	parent, err := h.deps.DB.ArticleByID(ctx, *id)
	if errors.Is(err, db.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return parent.FullName(), nil
}

func (e *editor) relock() (*refusal, error) {
	locked, ok := e.body.flag("locked")
	if !ok || locked == e.article.Locked {
		return nil, nil
	}
	if !e.may(perms.LockArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}
	if err := e.handler.deps.DB.SetArticleLock(e.req.Context(), e.article.ID, locked); err != nil {
		return nil, err
	}
	e.article.Locked = locked
	return nil, nil
}

func (e *editor) recredit() (*refusal, error) {
	raw, ok := e.body.list("authorsIds")
	if !ok {
		return nil, nil
	}
	if !e.may(perms.EditArticles) || !e.may(perms.ManageArticleAuthors) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}

	ids := make([]int64, 0, len(raw))
	for _, one := range raw {
		if id, err := strconv.ParseInt(one, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	rev, _, err := e.handler.deps.DB.SetArticleAuthors(e.req.Context(), e.article.ID, ids, e.userID, e.at)
	if err != nil {
		return nil, err
	}
	return nil, e.announce(rev)
}

func (e *editor) announce(rev db.Revision) error {
	return e.handler.notifyRevision(e.req, e.article, rev)
}

func (e *editor) answer() (string, int, error) {
	ctx := e.req.Context()
	article, err := e.handler.deps.DB.ArticleByID(ctx, e.article.ID)
	if err != nil {
		return "", 0, err
	}
	source, err := e.handler.latestSource(ctx, article.ID)
	if err != nil {
		return "", 0, err
	}
	if source != nil {
		if err := e.reindex(article, *source); err != nil {
			return "", 0, err
		}
	}

	body, err := e.handler.articleJSON(e.req, article, source)
	if err != nil {
		return "", 0, err
	}
	return body, http.StatusOK, nil
}

// A source the engine cannot read still has to be searchable, so the raw text
// stands in for the rendering.
func (e *editor) reindex(article *db.Article, source string) error {
	if source == "" {
		return nil
	}
	indexed := article.Title + "\n\n" + source
	plaintext := indexed
	if text, err := e.handler.renderText(e.req, e.site, article, source); err == nil {
		plaintext = article.Title + "\n\n" + text
	}
	return e.handler.deps.DB.UpdateSearchIndex(e.req.Context(), article.ID, indexed, plaintext)
}

func samePointer(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (h *Articles) renderText(r *http.Request, current *db.Site, article *db.Article, source string) (string, error) {
	env := h.env(r, current, nil)
	info, err := env.PageInfo(article)
	if err != nil {
		return "", err
	}
	result, err := env.Text(source, info, env.Callbacks(nil, nil), renderer.ModeSystem)
	if err != nil {
		return "", err
	}
	return result.Body, nil
}

func (h *Articles) articleJSON(r *http.Request, article *db.Article, source *string) (string, error) {
	ctx := r.Context()
	tags, err := h.deps.DB.ArticleTags(ctx, article.ID)
	if err != nil {
		return "", err
	}
	for i := range tags {
		tags[i] = strings.ToLower(tags[i])
	}
	slices.Sort(tags)
	authors, err := h.deps.DB.ArticleAuthors(ctx, article.ID)
	if err != nil {
		return "", err
	}

	rendered := make(wikijson.Array, 0, len(authors))
	for i := range authors {
		one, err := repo.UserJSON(ctx, h.deps.DB, &authors[i])
		if err != nil {
			return "", err
		}
		rendered = append(rendered, one)
	}
	if len(rendered) == 0 {
		nobody, err := repo.UserJSON(ctx, h.deps.DB, nil)
		if err != nil {
			return "", err
		}
		rendered = append(rendered, nobody)
	}

	parent := ""
	if article.ParentID != nil {
		if found, err := h.deps.DB.ArticleByID(ctx, *article.ParentID); err == nil {
			parent = found.FullName()
		} else if !errors.Is(err, db.ErrNotFound) {
			return "", err
		}
	}

	return wikijson.Marshal(wikijson.Object{
		{Key: "uid", Value: article.ID},
		{Key: "pageId", Value: article.FullName()},
		{Key: "title", Value: article.Title},
		{Key: "source", Value: textOrNil(source)},
		{Key: "tags", Value: tags},
		{Key: "author", Value: rendered[0]},
		{Key: "authors", Value: rendered},
		{Key: "parent", Value: nullable(parent)},
		{Key: "locked", Value: article.Locked},
	})
}
