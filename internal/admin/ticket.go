package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const (
	ticketSlug     = "tickets"
	membershipSlug = "membership-applications"
	inviteSlug     = "invite-links"
)

var ticketStatuses = []string{db.TicketPending, db.TicketApproved, db.TicketRejected, db.TicketClosed}

func init() {
	register(screen{slug: ticketSlug, label: "admin.tickets", need: perms.ViewUserTickets, serve: (*Handler).tickets})
	register(screen{slug: membershipSlug, label: "admin.membership", need: perms.ReviewMembershipApplications, serve: (*Handler).membership})
	register(screen{slug: inviteSlug, label: "admin.invites", need: perms.ManageUsers, serve: (*Handler).invites})
}

func (h *Handler) tickets(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	return h.ticketScreen(w, r, loc, ticketSlug, db.TicketKind, "admin.tickets", false)
}

func (h *Handler) membership(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	return h.ticketScreen(w, r, loc, membershipSlug, db.MembershipApplyKind, "admin.membership", true)
}

func (h *Handler) ticketScreen(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, slug, kind, title string, grants bool) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+slug), "/")
	ctx := r.Context()

	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		stored, err := h.deps.DB.AdminTicket(ctx, siteID(ctx), id)
		if errors.Is(err, db.ErrNotFound) || (err == nil && stored.Kind != kind) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
		status := r.PostFormValue("status")
		if !contains(ticketStatuses, status) {
			return h.ticketForm(w, r, loc, slug, kind, title, grants, rest, loc.T("admin.report-bad-status"))
		}
		var role *int64
		if grants {
			granted, _, err := h.access(ctx)
			if err != nil {
				return err
			}
			if granted.Has(perms.ManagePermissions) {
				role = optionalID(r.PostFormValue("granted_role"))
			} else {
				role = stored.GrantedID
			}
		}
		mine := auth.FromContext(ctx)
		if mine == nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return nil
		}
		err = h.deps.DB.ReviewTicket(ctx, id, status, r.PostFormValue("admin_notes"), mine.ID, role, time.Now())
		if err != nil {
			return err
		}
		h.noteID(r, db.AdminChanged, slug, id, status)
		redirect(w, Prefix+slug+"/")
		return nil
	}

	if rest == "" {
		status := r.URL.Query().Get("status")
		if !contains(ticketStatuses, status) {
			status = ""
		}
		page := atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		found, total, err := h.deps.DB.AdminTickets(ctx, siteID(ctx), kind, status, perPage, (page-1)*perPage)
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T(title), "ticket_list.html", map[string]any{
			"Tickets":  found,
			"Status":   status,
			"Statuses": ticketStatuses,
			"Page":     page,
			"Pages":    (total + perPage - 1) / perPage,
			"Total":    total,
			"Base":     Prefix + slug + "/",
			"Action":   Prefix + slug + "/",
		})
	}
	return h.ticketForm(w, r, loc, slug, kind, title, grants, rest, "")
}

func (h *Handler) ticketForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, slug, kind, title string, grants bool, rest, problem string) error {
	ctx := r.Context()
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.AdminTicket(ctx, siteID(ctx), id)
	if errors.Is(err, db.ErrNotFound) || (err == nil && row.Kind != kind) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	var roleList []db.RoleChoice
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mayGrant := grants && granted.Has(perms.ManagePermissions)
	if mayGrant {
		roleList, err = h.deps.DB.AllRoles(ctx, siteID(ctx))
		if err != nil {
			return err
		}
	}
	return h.page(w, r, loc, loc.T(title), "ticket_form.html", map[string]any{
		"Ticket":   row,
		"Statuses": ticketStatuses,
		"Roles":    roleList,
		"MayGrant": mayGrant,
		"CSRF":     csrf.Issue(w, r),
		"Error":    problem,
		"Action":   Prefix + slug + "/" + rest,
		"Back":     Prefix + slug + "/",
	})
}

func (h *Handler) invites(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		if id := optionalID(r.PostFormValue("id")); id != nil && r.PostFormValue("delete") != "" {
			if err := h.deps.DB.DeleteInvite(ctx, siteID(ctx), *id); err != nil {
				return err
			}
			h.noteID(r, db.AdminDeleted, inviteSlug, *id, "")
		}
		redirect(w, Prefix+inviteSlug+"/")
		return nil
	}
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	found, total, err := h.deps.DB.AdminInvites(ctx, siteID(ctx), perPage, (page-1)*perPage)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.invites"), "invite_list.html", map[string]any{
		"Invites": found,
		"Page":    page,
		"Pages":   (total + perPage - 1) / perPage,
		"Total":   total,
		"CSRF":    csrf.Issue(w, r),
		"Action":  Prefix + inviteSlug + "/",
	})
}
