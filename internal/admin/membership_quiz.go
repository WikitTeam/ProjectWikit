package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/quiz"
)

const membershipQuizSlug = "membership-quiz"

func init() {
	register(screen{slug: membershipQuizSlug, label: "admin.membership-quiz", need: perms.ManageSite, serve: (*Handler).membershipQuiz})
}

func (h *Handler) membershipQuiz(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	current, err := h.deps.DB.MembershipQuiz(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	choices := assignableRoles(roleList)
	// The role is held to the same bar as the role fields on the site screen.
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mayGrant := granted.Has(perms.ManagePermissions)
	problem := ""
	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		current.Enabled = r.PostFormValue("enabled") != ""
		if mayGrant {
			current.RoleID = offered(optionalID(r.PostFormValue("role")), choices)
		}
		current.Tries = boundedInt(r.PostFormValue("tries"), 1, 100, 3)
		current.CooldownHours = boundedInt(r.PostFormValue("cooldown_hours"), 0, 24*365, 12)
		current.Questions = r.PostFormValue("questions")
		switch {
		case current.Enabled && current.RoleID == nil:
			problem = loc.T("admin.quiz-no-role")
		case current.Enabled && len(quiz.Parse(current.Questions)) == 0:
			problem = loc.T("admin.quiz-no-questions")
		default:
			if err := h.deps.DB.SaveMembershipQuiz(ctx, siteID(ctx), current); err != nil {
				return err
			}
			h.note(r, db.AdminChanged, membershipQuizSlug, "", "")
			redirect(w, Prefix+membershipQuizSlug+"/")
			return nil
		}
	}
	return h.page(w, r, loc, loc.T("admin.membership-quiz"), "membership_quiz.html", map[string]any{
		"Quiz":      current,
		"Roles":     choices,
		"MayGrant":  mayGrant,
		"RoleName":  roleLabel(choices, current.RoleID),
		"Questions": len(quiz.Parse(current.Questions)),
		"Error":     problem,
		"CSRF":      csrf.Issue(w, r),
		"Action":    Prefix + membershipQuizSlug + "/",
	})
}

func boundedInt(raw string, low, high, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return min(max(n, low), high)
}

func roleLabel(choices []db.RoleChoice, id *int64) string {
	if id == nil {
		return ""
	}
	for _, one := range choices {
		if one.ID == *id {
			return one.Label()
		}
	}
	return ""
}
