package admin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/changelog"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const dashboardRows = 8

type queueItem struct {
	Href  string
	Label string
	Count int
}

type changeRow struct {
	Flags     []changelog.Flag
	Title     string
	Href      string
	User      string
	Comment   string
	CreatedAt time.Time
}

type actionRow struct {
	Action    string
	Screen    string
	User      string
	Label     string
	CreatedAt time.Time
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, granted perms.Set) error {
	ctx := r.Context()
	current := site.FromContext(ctx)

	settings, err := h.deps.DB.SiteSettings(ctx, current.ID)
	if err != nil {
		return err
	}
	queue, err := h.queue(ctx, loc, granted)
	if err != nil {
		return err
	}
	changes, err := h.recentChanges(ctx, loc)
	if err != nil {
		return err
	}

	data := map[string]any{
		"Site":     current,
		"Settings": settings,
		"Queue":    queue,
		"Changes":  changes,
		"SiteHref": Prefix + siteSlug + "/",
		"MaySite":  granted.Has(perms.ManageSite),
	}
	if granted.Has(perms.ViewActionsLog) {
		actions, err := h.recentActions(ctx)
		if err != nil {
			return err
		}
		data["Actions"] = actions
		data["ActionsHref"] = Prefix + adminLogSlug + "/"
	}
	return h.page(w, r, loc, loc.T("admin.dashboard"), "dashboard.html", data)
}

func (h *Handler) queue(ctx context.Context, loc *i18n.Localizer, granted perms.Set) ([]queueItem, error) {
	var out []queueItem
	if granted.Has(perms.ViewUserReports) {
		_, total, err := h.deps.DB.AdminReports(ctx, db.ReportPending, 1, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, queueItem{Prefix + reportSlug + "/?status=" + db.ReportPending, loc.T("admin.reports"), total})
	}
	if granted.Has(perms.ViewUserTickets) {
		_, total, err := h.deps.DB.AdminTickets(ctx, db.TicketKind, db.TicketPending, 1, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, queueItem{Prefix + ticketSlug + "/?status=" + db.TicketPending, loc.T("admin.tickets"), total})
	}
	if granted.Has(perms.ReviewMembershipApplications) {
		_, total, err := h.deps.DB.AdminTickets(ctx, db.MembershipApplyKind, db.TicketPending, 1, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, queueItem{Prefix + membershipSlug + "/?status=" + db.TicketPending, loc.T("admin.membership"), total})
	}
	if granted.Has(perms.ManageUsers) {
		total, err := h.deps.DB.OpenInviteCount(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, queueItem{Prefix + inviteSlug + "/", loc.T("admin.open-invites"), total})
	}
	return out, nil
}

func (h *Handler) recentChanges(ctx context.Context, loc *i18n.Localizer) ([]changeRow, error) {
	found, err := h.deps.DB.SiteChanges(ctx, db.SiteChangeFilter{}, 0, dashboardRows)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, c := range found {
		if c.UserID != nil {
			ids = append(ids, *c.UserID)
		}
	}
	names, err := h.userNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	users := func(want []int64) ([]db.User, error) { return h.deps.DB.UsersByIDs(ctx, want) }

	out := make([]changeRow, 0, len(found))
	for _, c := range found {
		row := changeRow{
			Title:     c.ArticleTitle,
			Href:      "/" + c.ArticleName,
			CreatedAt: c.CreatedAt,
		}
		if c.ArticleCategory != "_default" {
			row.Href = "/" + c.ArticleCategory + ":" + c.ArticleName
		}
		if row.Title == "" {
			row.Title = c.ArticleName
		}
		if c.UserID != nil {
			row.User = names[*c.UserID]
		}
		entry, err := changelog.Of(loc, users, c)
		switch {
		case errors.Is(err, changelog.ErrUnreadable):
		case err != nil:
			return nil, err
		default:
			row.Flags = entry.Flags
			row.Comment = entry.Comment
		}
		out = append(out, row)
	}
	return out, nil
}

func (h *Handler) recentActions(ctx context.Context) ([]actionRow, error) {
	found, err := h.deps.DB.AdminNotes(ctx, "", dashboardRows, 0)
	if err != nil {
		return nil, err
	}
	out := make([]actionRow, 0, len(found))
	for _, e := range found {
		who := e.User
		if who == "" {
			who = e.Stale
		}
		out = append(out, actionRow{
			Action:    e.Action,
			Screen:    e.Screen,
			User:      who,
			Label:     e.Label,
			CreatedAt: e.CreatedAt,
		})
	}
	return out, nil
}

func (h *Handler) userNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	users, err := h.deps.DB.UsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u.DisplayLabel()
	}
	return out, nil
}
