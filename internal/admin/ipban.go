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
	"github.com/WikitTeam/ProjectWikit/internal/ipban"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const ipBanSlug = "ip-bans"

func init() {
	register(screen{slug: ipBanSlug, label: "admin.ip-bans", need: perms.BanMembers, serve: (*Handler).ipBans})
}

func (h *Handler) ipBans(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	problem := ""
	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		if r.PostFormValue("delete") != "" {
			id, err := strconv.ParseInt(r.PostFormValue("id"), 10, 64)
			if err != nil {
				notFound(w)
				return nil
			}
			network, err := h.deps.DB.DeleteIPBan(ctx, siteID(ctx), id)
			if err != nil && !errors.Is(err, db.ErrNotFound) {
				return err
			}
			if err == nil {
				h.noteID(r, db.AdminDeleted, ipBanSlug, id, network)
			}
			redirect(w, Prefix+ipBanSlug+"/")
			return nil
		}
		network, ok := ipban.Parse(r.PostFormValue("network"))
		if !ok {
			problem = loc.T("admin.ip-ban-bad-network")
		} else {
			var by *int64
			if mine := auth.FromContext(ctx); mine != nil {
				by = &mine.ID
			}
			until := optionalTime(r.PostFormValue("until"), editorZone(r))
			reason := strings.TrimSpace(r.PostFormValue("reason"))
			id, err := h.deps.DB.SetIPBan(ctx, siteID(ctx), network, until, reason, by, time.Now().UTC())
			if err != nil {
				return err
			}
			h.noteID(r, db.AdminCreated, ipBanSlug, id, network.String())
			redirect(w, Prefix+ipBanSlug+"/")
			return nil
		}
	}

	bans, err := h.deps.DB.IPBans(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	now := time.Now()
	return h.page(w, r, loc, loc.T("admin.ip-bans"), "ip_ban_list.html", map[string]any{
		"Bans":     bans,
		"Now":      now,
		"Error":    problem,
		"Network":  r.PostFormValue("network"),
		"Reason":   r.PostFormValue("reason"),
		"CSRF":     csrf.Issue(w, r),
		"Action":   Prefix + ipBanSlug + "/",
		"ZoneName": site.Zone(ctx).String(),
	})
}
