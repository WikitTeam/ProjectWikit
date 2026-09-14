package admin

import (
	"net/http"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

// Every sanction a site can put on a member, in the order the screen shows
// them. A ban already carries the other three, which is why it comes first.
var sanctionKinds = []struct {
	Kind perms.Sanction
	Need string
	Text string
}{
	{perms.SanctionBan, perms.BanMembers, "admin.sanction-ban"},
	{perms.SanctionMute, perms.MuteMembers, "admin.sanction-mute"},
	{perms.SanctionEdit, perms.RestrictMemberEditing, "admin.sanction-edit"},
	{perms.SanctionRating, perms.RestrictMemberRating, "admin.sanction-rating"},
}

type sanctionRow struct {
	Kind    string
	Label   string
	InForce bool
	Until   *time.Time
	Reason  string
	SetAt   time.Time
}

func sanctionRows(stored []db.MemberSanction, granted perms.Set) []sanctionRow {
	held := make(map[string]db.MemberSanction, len(stored))
	for _, one := range stored {
		held[one.Kind] = one
	}
	out := make([]sanctionRow, 0, len(sanctionKinds))
	for _, kind := range sanctionKinds {
		if !granted.Has(kind.Need) {
			continue
		}
		row := sanctionRow{Kind: string(kind.Kind), Label: kind.Text}
		if one, ok := held[string(kind.Kind)]; ok {
			row.InForce, row.Until, row.Reason, row.SetAt = true, one.Until, one.Reason, one.SetAt
		}
		out = append(out, row)
	}
	return out
}

func neededForAction(what string) string {
	if what == actionBot {
		return perms.ManageBots
	}
	return perms.InviteMembers
}

func (h *Handler) allowed(w http.ResponseWriter, r *http.Request, need string) bool {
	if grantsFrom(r.Context()).Has(need) {
		return true
	}
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	return false
}

// A sanction the operator was never shown is left alone rather than lifted.
func (h *Handler) saveSanctions(r *http.Request, target int64) error {
	ctx := r.Context()
	granted := grantsFrom(ctx)
	var by *int64
	if mine := auth.FromContext(ctx); mine != nil {
		by = &mine.ID
	}
	zone := editorZone(r)
	now := time.Now().UTC()

	for _, kind := range sanctionKinds {
		if !granted.Has(kind.Need) {
			continue
		}
		name := string(kind.Kind)
		if r.PostFormValue("sanction_"+name) == "" {
			if err := h.deps.DB.ClearSanction(ctx, siteID(ctx), target, name); err != nil {
				return err
			}
			continue
		}
		until := optionalTime(r.PostFormValue("sanction_"+name+"_until"), zone)
		reason := r.PostFormValue("sanction_" + name + "_reason")
		if err := h.deps.DB.SetSanction(ctx, siteID(ctx), target, name, until, reason, by, now); err != nil {
			return err
		}
	}
	return nil
}
