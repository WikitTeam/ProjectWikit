package webapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

func (h *Articles) requiredTags(ctx context.Context, siteID int64, category string, subject perms.Subject) ([][]string, error) {
	if subject.Superuser {
		return nil, nil
	}
	roleIDs := make([]int64, len(subject.Roles))
	for i, role := range subject.Roles {
		roleIDs[i] = role.ID
	}
	return h.deps.DB.RequiredTags(ctx, siteID, category, roleIDs)
}

// The refusal spells out every group still missing, so the reader fixes them in
// one go rather than one per attempt.
func missingTagsRefusal(groups [][]string, tags []string) *refusal {
	missing := db.MissingTagGroups(groups, tags)
	if len(missing) == 0 {
		return nil
	}
	names := make([]string, len(missing))
	for i, group := range missing {
		names[i] = "[" + strings.Join(group, " / ") + "]"
	}
	return &refusal{key: "api-tags-required", status: http.StatusBadRequest, args: []any{"groups", strings.Join(names, " ")}}
}

// Everything the tags could be refused for is settled before the page exists,
// so a refusal never leaves a page behind without them.
func (h *Articles) mayTagNew(r *http.Request, current *db.Site, name string, user *db.User, tags []string) (*refusal, error) {
	ctx := r.Context()
	perm := repo.NewPerms(ctx, h.deps.DB)
	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return nil, err
	}
	category, _ := wikidot.Split(name)
	groups, err := h.requiredTags(ctx, current.ID, category, subject)
	if err != nil {
		return nil, err
	}
	if bad := missingTagsRefusal(groups, tags); bad != nil {
		return bad, nil
	}
	if len(tags) == 0 {
		return nil, nil
	}
	object, err := perm.Category(category)
	if err != nil {
		return nil, err
	}
	if !perms.Resolve(subject, object).Has(perms.TagArticles) {
		return ptr(refuse("api-forbidden", http.StatusForbidden)), nil
	}
	allow, err := h.tagsCreatable(ctx, current.ID, category)
	if err != nil || allow {
		return nil, err
	}
	unknown, err := h.deps.DB.UnknownTags(ctx, current.ID, tags)
	if err != nil || len(unknown) == 0 {
		return nil, err
	}
	return &refusal{key: "api-tags-unknown", status: http.StatusBadRequest, args: []any{"tags", strings.Join(unknown, " ")}}, nil
}

func (h *Articles) tagsCreatable(ctx context.Context, siteID int64, category string) (bool, error) {
	siteMode, err := h.deps.DB.SiteCanCreateTags(ctx, siteID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return false, err
	}
	own, err := h.deps.DB.CategoryCanCreateTags(ctx, siteID, category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return false, err
	}
	return db.CreatingTagsAllowed(siteMode, own), nil
}
