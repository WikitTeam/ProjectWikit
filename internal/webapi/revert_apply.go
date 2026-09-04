package webapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type reverter struct {
	handler *Articles
	req     *http.Request
	site    *db.Site
	article *db.Article
	userID  *int64
	at      time.Time

	plan     *plan
	subtypes []string
	meta     map[string]any
}

func (h *Articles) revert(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
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
	user := auth.FromContext(ctx)
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return "", 0, err
	}
	object, err := perm.Article(article, user)
	if err != nil {
		return "", 0, err
	}
	if !perms.Resolve(subject, object).Has(perms.EditArticles) {
		return "", 0, errForbidden
	}

	target, ok := revertTarget(r)
	if !ok {
		return field("error", loc.T("api-bad-revision")), http.StatusBadRequest, nil
	}

	entries, err := h.deps.DB.ArticleLogAbove(ctx, article.ID, target)
	if err != nil {
		return "", 0, err
	}
	wanted, err := planRevert(entries)
	if err != nil {
		return "", 0, err
	}

	rev := &reverter{
		handler: h, req: r, site: current, article: article,
		at: time.Now().UTC(), plan: wanted, meta: map[string]any{},
	}
	if user != nil {
		rev.userID = &user.ID
	}
	if err := rev.apply(target); err != nil {
		return "", 0, err
	}

	body, err := wikijson.Marshal(wikijson.Object{{Key: "pageId", Value: rev.article.FullName()}})
	return body, http.StatusOK, err
}

func revertTarget(r *http.Request) (int, bool) {
	raw, err := readBody(r)
	if err != nil {
		return 0, false
	}
	var body fields
	if json.Unmarshal(raw, &body) != nil {
		return 0, false
	}
	value, ok := body["revNumber"]
	if !ok {
		return 0, false
	}
	var target int
	if json.Unmarshal(value, &target) != nil {
		return 0, false
	}
	return target, true
}

// The links a new source writes are keyed by the name the page still has, so
// the rename has to come after it.
func (rev *reverter) apply(target int) error {
	for _, step := range []func() error{
		rev.files, rev.tags, rev.source, rev.title, rev.name, rev.parent, rev.votes, rev.authors,
	} {
		if err := step(); err != nil {
			return err
		}
	}
	rev.meta["rev_number"] = target
	rev.meta["subtypes"] = rev.subtypes

	encoded, err := json.Marshal(rev.meta)
	if err != nil {
		return err
	}
	written, err := rev.handler.deps.DB.WriteRevertEntry(rev.req.Context(), db.RevertWrite{
		ArticleID: rev.article.ID, UserID: rev.userID, Meta: encoded, At: rev.at,
	})
	if err != nil {
		return err
	}
	if err := rev.handler.notifyRevision(rev.req, rev.article, written); err != nil {
		return err
	}
	return rev.reindex()
}

func (rev *reverter) mark(kind string) {
	if !slices.Contains(rev.subtypes, kind) {
		rev.subtypes = append(rev.subtypes, kind)
	}
}

func (rev *reverter) files() error {
	ctx := rev.req.Context()
	var renamed, deleted, restored []any

	for id, name := range rev.plan.filesRenamed {
		previous, ok, err := rev.handler.deps.DB.RenameFile(ctx, id, name)
		if err != nil {
			return err
		}
		if ok {
			renamed = append(renamed, map[string]any{"id": id, "name": name, "prev_name": previous})
		}
	}
	for id, drop := range rev.plan.filesDeleted {
		if !drop {
			continue
		}
		name, ok, err := rev.handler.deps.DB.SoftDeleteFile(ctx, id, rev.at, rev.userID)
		if err != nil {
			return err
		}
		if ok {
			deleted = append(deleted, map[string]any{"id": id, "name": name})
		}
	}
	for id, back := range rev.plan.filesRestored {
		if !back {
			continue
		}
		name, ok, err := rev.handler.deps.DB.RestoreFile(ctx, id)
		if err != nil {
			return err
		}
		if ok {
			restored = append(restored, map[string]any{"id": id, "name": name})
		}
	}

	if renamed == nil && deleted == nil && restored == nil {
		return nil
	}
	if restored != nil {
		rev.mark(db.LogFileAdded)
	}
	if deleted != nil {
		rev.mark(db.LogFileDeleted)
	}
	if renamed != nil {
		rev.mark(db.LogFileRenamed)
	}
	rev.meta["files"] = map[string]any{
		"added": orEmpty(restored), "deleted": orEmpty(deleted), "renamed": orEmpty(renamed),
	}
	return nil
}

func (rev *reverter) tags() error {
	ctx := rev.req.Context()
	held, err := rev.handler.deps.DB.ArticleTagIDs(ctx, rev.article.ID)
	if err != nil {
		return err
	}

	var added, removed []int64
	for _, id := range rev.plan.removedTags {
		if at := slices.Index(held, id); at >= 0 {
			held = slices.Delete(held, at, at+1)
			removed = append(removed, id)
		}
	}
	for _, id := range rev.plan.addedTags {
		held = append(held, id)
		added = append(added, id)
	}
	if err := rev.handler.deps.DB.SetArticleTagIDs(ctx, rev.article.ID, held); err != nil {
		return err
	}
	if added == nil && removed == nil {
		return nil
	}
	rev.mark(db.LogTags)
	rev.meta["tags"] = map[string]any{"added": orEmptyIDs(added), "removed": orEmptyIDs(removed)}
	return nil
}

func (rev *reverter) source() error {
	if rev.plan.sourceFrom == nil {
		return nil
	}
	ctx := rev.req.Context()
	source, err := rev.handler.deps.DB.PreviousVersionSource(ctx, *rev.plan.sourceFrom)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	version, err := rev.handler.deps.DB.AddArticleVersion(ctx, rev.article.ID, source, rev.at)
	if err != nil {
		return err
	}
	rev.mark(db.LogSource)
	rev.meta["source"] = map[string]any{"version_id": version}
	return rev.handler.refreshLinks(rev.req, rev.site, rev.article.ID, source)
}

func (rev *reverter) title() error {
	if rev.plan.title == nil {
		return nil
	}
	previous := rev.article.Title
	if err := rev.handler.deps.DB.SetArticleTitle(rev.req.Context(), rev.article.ID, *rev.plan.title); err != nil {
		return err
	}
	rev.article.Title = *rev.plan.title
	rev.mark(db.LogTitle)
	rev.meta["title"] = map[string]any{"prev_title": previous, "title": *rev.plan.title}
	return nil
}

func (rev *reverter) name() error {
	if rev.plan.name == nil {
		return nil
	}
	from := rev.article.FullName()
	category, bare := wikidot.Split(*rev.plan.name)
	if err := rev.handler.deps.DB.MoveArticle(rev.req.Context(), rev.article.ID, category, bare, from); err != nil {
		return err
	}
	rev.article.Category, rev.article.Name = category, bare
	rev.mark(db.LogName)
	rev.meta["name"] = map[string]any{"prev_name": from, "name": rev.article.FullName()}
	return nil
}

func (rev *reverter) parent() error {
	if !rev.plan.parentSet {
		return nil
	}
	ctx := rev.req.Context()
	previous, err := rev.handler.parentName(ctx, rev.article.ParentID)
	if err != nil {
		return err
	}
	wanted, err := rev.handler.parentName(ctx, rev.plan.parentID)
	if err != nil {
		return err
	}
	if err := rev.handler.deps.DB.MoveArticleParent(ctx, rev.article.ID, rev.plan.parentID); err != nil {
		return err
	}
	rev.mark(db.LogParent)
	rev.meta["parent"] = map[string]any{
		"parent": nullable(wanted), "parent_id": rev.plan.parentID,
		"prev_parent": nullable(previous), "prev_parent_id": rev.article.ParentID,
	}
	rev.article.ParentID = rev.plan.parentID
	return nil
}

func (rev *reverter) votes() error {
	if rev.plan.votes == nil {
		return nil
	}
	ctx := rev.req.Context()
	current, err := rev.handler.votesMeta(rev.req, rev.article)
	if err != nil {
		return err
	}
	var kept map[string]any
	if err := json.Unmarshal([]byte(current), &kept); err != nil {
		return err
	}

	restored, err := restoredVotes(rev.plan.votes)
	if err != nil {
		return err
	}
	if err := rev.handler.deps.DB.RestoreArticleVotes(ctx, rev.article.ID, restored); err != nil {
		return err
	}
	rev.mark(db.LogVotesDeleted)
	rev.meta["votes"] = kept
	return nil
}

func restoredVotes(raw json.RawMessage) ([]db.RestoredVote, error) {
	var stored struct {
		Votes []struct {
			UserID *int64  `json:"user_id"`
			RoleID *int64  `json:"role_id"`
			Rate   float64 `json:"vote"`
			Date   *string `json:"date"`
		} `json:"votes"`
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}

	out := make([]db.RestoredVote, 0, len(stored.Votes))
	for _, one := range stored.Votes {
		if one.UserID == nil {
			continue
		}
		vote := db.RestoredVote{UserID: *one.UserID, RoleID: one.RoleID, Rate: one.Rate}
		if one.Date != nil {
			if at, err := time.Parse(time.RFC3339Nano, *one.Date); err == nil {
				vote.Date = &at
			}
		}
		out = append(out, vote)
	}
	return out, nil
}

func (rev *reverter) authors() error {
	ctx := rev.req.Context()
	held, err := rev.handler.deps.DB.ArticleAuthorIDs(ctx, rev.article.ID)
	if err != nil {
		return err
	}

	var added, removed []int64
	for _, id := range rev.plan.removedAuthors {
		if at := slices.Index(held, id); at >= 0 {
			held = slices.Delete(held, at, at+1)
			removed = append(removed, id)
		}
	}
	for _, id := range rev.plan.addedAuthors {
		held = append(held, id)
		added = append(added, id)
	}
	if _, _, err := rev.handler.deps.DB.SetArticleAuthors(ctx, rev.article.ID, held, rev.userID, rev.at); err != nil {
		return err
	}
	if added == nil && removed == nil {
		return nil
	}
	rev.mark(db.LogAuthorship)
	rev.meta["authorship"] = map[string]any{"added": orEmptyIDs(added), "removed": orEmptyIDs(removed)}
	return nil
}

func (rev *reverter) reindex() error {
	ctx := rev.req.Context()
	article, err := rev.handler.deps.DB.ArticleByID(ctx, rev.article.ID)
	if err != nil {
		return err
	}
	source, err := rev.handler.deps.DB.LatestSource(ctx, article.ID)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	indexed := article.Title + "\n\n" + source
	plaintext := indexed
	if text, err := rev.handler.renderText(rev.req, rev.site, article, source); err == nil {
		plaintext = article.Title + "\n\n" + text
	}
	return rev.handler.deps.DB.UpdateSearchIndex(ctx, article.ID, indexed, plaintext)
}

func orEmpty(list []any) []any {
	if list == nil {
		return []any{}
	}
	return list
}

func orEmptyIDs(list []int64) []int64 {
	if list == nil {
		return []int64{}
	}
	return list
}
