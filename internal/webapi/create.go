package webapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const CreatePath = ArticlesPrefix + "new"

const (
	sourceLimit = 200000
	bodyLimit   = 4 * sourceLimit
)

type createInput struct {
	PageID string  `json:"pageId"`
	Title  *string `json:"title"`
	Source *string `json:"source"`
	Parent *string `json:"parent"`
}

// A refusal the visitor caused carries the message it should show, so it comes
// back as a body rather than as an error.
type refusal struct {
	key    string
	status int
}

func refuse(key string, status int) refusal { return refusal{key: key, status: status} }

func (h *Articles) create(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	body, status, err := h.createArticle(r, loc)
	if err != nil {
		h.fail(w, loc, err)
		return
	}
	writeJSON(w, status, body)
}

func (h *Articles) createArticle(r *http.Request, loc *i18n.Localizer) (string, int, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		return field("error", loc.T("api-csrf-failed")), http.StatusForbidden, nil
	}

	input, bad := readCreateInput(r)
	if bad != nil {
		return field("error", loc.T(bad.key)), bad.status, nil
	}
	name := wikidot.Normalize(input.PageID)
	user := auth.FromContext(ctx)

	if bad := h.mayCreate(r, name, user); bad != nil {
		return field("error", loc.T(bad.key)), bad.status, nil
	}
	if _, err := h.deps.DB.ArticleByName(ctx, name); err == nil {
		return field("error", loc.T("api-page-exists")), http.StatusConflict, nil
	} else if !errors.Is(err, db.ErrNotFound) {
		return "", 0, err
	}

	if err := h.writeNewArticle(r, current, name, input, user); err != nil {
		return "", 0, err
	}
	return field("status", "ok"), http.StatusCreated, nil
}

func readCreateInput(r *http.Request) (createInput, *refusal) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, bodyLimit+1))
	if err != nil || len(raw) > bodyLimit {
		return createInput{}, ptr(refuse("api-bad-request", http.StatusBadRequest))
	}
	if len(raw) == 0 {
		return createInput{}, ptr(refuse("api-bad-request", http.StatusBadRequest))
	}
	var input createInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return createInput{}, ptr(refuse("api-bad-json", http.StatusBadRequest))
	}

	switch {
	case input.PageID == "" || !wikidot.NameAllowed(input.PageID):
		return createInput{}, ptr(refuse("api-bad-page-id", http.StatusBadRequest))
	case input.Source == nil || strings.TrimSpace(*input.Source) == "":
		return createInput{}, ptr(refuse("api-missing-source", http.StatusBadRequest))
	case input.Title == nil:
		return createInput{}, ptr(refuse("api-missing-title", http.StatusBadRequest))
	case len(*input.Source) > sourceLimit:
		return createInput{}, ptr(refuse("api-source-too-long", http.StatusBadRequest))
	}
	return input, nil
}

func ptr[T any](v T) *T { return &v }

func (h *Articles) mayCreate(r *http.Request, name string, user *db.User) *refusal {
	ctx := r.Context()
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return ptr(refuse("api-internal-error", http.StatusInternalServerError))
	}
	category, _ := wikidot.Split(name)
	object, err := perm.Category(category)
	if err != nil {
		return ptr(refuse("api-internal-error", http.StatusInternalServerError))
	}
	if !perms.Resolve(subject, object).Has(perms.CreateArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden))
	}
	return nil
}

func (h *Articles) writeNewArticle(r *http.Request, current *db.Site, name string,
	input createInput, user *db.User) error {

	ctx := r.Context()
	category, bare := wikidot.Split(name)
	at := time.Now().UTC()
	var userID *int64
	if user != nil {
		userID = &user.ID
	}

	id, err := h.deps.DB.CreateArticle(ctx, category, bare, *input.Title, userID, at)
	if err != nil {
		return err
	}
	rev, err := h.deps.DB.CreateArticleVersion(ctx, db.VersionWrite{
		ArticleID: id,
		Source:    *input.Source,
		UserID:    userID,
		Kind:      db.LogNew,
		Title:     *input.Title,
		At:        at,
	})
	if err != nil {
		return err
	}
	fresh, err := h.deps.DB.ArticleByID(ctx, id)
	if err != nil {
		return err
	}
	if err := h.logCreate(r, fresh, user); err != nil {
		return err
	}
	if err := h.notifyRevision(r, fresh, rev); err != nil {
		return err
	}
	if err := h.refreshLinks(r, current, id, *input.Source); err != nil {
		return err
	}
	if err := h.setParent(r, current, id, input, userID, at); err != nil {
		return err
	}
	if user != nil {
		return h.deps.DB.SubscribeToArticle(ctx, user.ID, id)
	}
	return nil
}

// The pass runs as nobody, so a page's links are the same set whoever saved it.
func (h *Articles) refreshLinks(r *http.Request, current *db.Site, id int64, source string) error {
	ctx := r.Context()
	article, err := h.deps.DB.ArticleByID(ctx, id)
	if err != nil {
		return err
	}
	env := h.env(r, current, nil)

	info, err := env.PageInfo(article)
	if err != nil {
		return err
	}
	pc := page.NewContext(article, article, nil, nil)
	links, err := env.Backlinks(source, info, env.Callbacks(env.Vars(article), pc))
	if err != nil {
		return err
	}
	return h.deps.DB.ReplaceArticleLinks(ctx, strings.ToLower(article.FullName()), links)
}

func (h *Articles) env(r *http.Request, current *db.Site, viewer *db.User) *pagerender.Env {
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	return pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(r.Context(), loc, current, viewer)
}

func (h *Articles) setParent(r *http.Request, current *db.Site, id int64,
	input createInput, userID *int64, at time.Time) error {

	if input.Parent == nil {
		return nil
	}
	ctx := r.Context()
	wanted := wikidot.Normalize(*input.Parent)
	parent, err := h.deps.DB.ArticleByName(ctx, wanted)
	if errors.Is(err, db.ErrNotFound) {
		parent = nil
	} else if err != nil {
		return err
	}

	var parentID *int64
	if parent != nil {
		parentID = &parent.ID
	}
	meta, err := parentMeta(wanted, parentID)
	if err != nil {
		return err
	}
	rev, err := h.deps.DB.SetArticleParent(ctx, id, parentID, userID, meta, at)
	if err != nil {
		return err
	}
	child, err := h.deps.DB.ArticleByID(ctx, id)
	if err != nil {
		return err
	}
	return h.notifyRevision(r, child, rev)
}

// A page created with a parent had none a moment ago, so the previous half of
// the entry is always empty.
func parentMeta(wanted string, parentID *int64) (string, error) {
	encoded, err := json.Marshal(map[string]any{
		"parent":         wanted,
		"prev_parent":    nil,
		"parent_id":      parentID,
		"prev_parent_id": nil,
	})
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
