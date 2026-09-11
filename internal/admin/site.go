package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/timezone"
)

const siteSlug = "site"

var (
	ratingModes = []string{"default", "disabled", "updown", "stars"}
	tagModes    = []string{"default", "disabled", "enabled"}
	emailPolicy = []string{db.EmailOptional, db.EmailRequired, db.EmailAtSignup}
)

func init() {
	register(screen{slug: siteSlug, label: "admin.site", need: perms.ManageSite, serve: (*Handler).site})
}

func (h *Handler) site(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if r.Method == http.MethodPost {
		return h.saveSite(w, r, loc)
	}
	return h.siteForm(w, r, loc, "", "")
}

func (h *Handler) siteForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem, done string) error {
	ctx := r.Context()
	current := site.FromContext(ctx)
	settings, err := h.deps.DB.SiteSettings(ctx, current.ID)
	if err != nil {
		return err
	}
	themes, err := h.deps.DB.Themes(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}

	return h.page(w, r, loc, loc.T("admin.site"), "site_form.html", map[string]any{
		"Site":        current,
		"Settings":    settings,
		"Themes":      themes,
		"Roles":       roleList,
		"RatingModes": ratingModes,
		"TagModes":    tagModes,
		"Policies":    emailPolicy,
		"Languages":   h.deps.Bundle.Choices(),
		"MayGrant":    granted.Has(perms.ManagePermissions),
		"CSRF":        csrf.Issue(w, r),
		"Error":       problem,
		"Done":        done,
		"Action":      Prefix + siteSlug + "/",
	})
}

func (h *Handler) saveSite(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}

	next := *current

	next.Slug = strings.TrimSpace(r.PostFormValue("slug"))
	next.Title = strings.TrimSpace(r.PostFormValue("title"))
	next.Headline = strings.TrimSpace(r.PostFormValue("headline"))
	next.Domain = strings.TrimSpace(r.PostFormValue("domain"))
	next.MediaDomain = strings.TrimSpace(r.PostFormValue("media_domain"))
	next.HomePage = strings.TrimSpace(r.PostFormValue("home_page"))
	next.Icon, err = h.pickIcon(r, "icon", current.Icon, siteIcons)
	if err != nil {
		return h.iconProblem(w, r, loc, err)
	}
	next.AuthIcon, err = h.pickIcon(r, "auth_icon", current.AuthIcon, siteIcons)
	if err != nil {
		return h.iconProblem(w, r, loc, err)
	}
	next.FooterLicense = r.PostFormValue("footer_license")
	next.SignupNotice = r.PostFormValue("signup_notice")
	next.PasswordHelp = r.PostFormValue("password_help")
	next.EmailPolicy = r.PostFormValue("email_policy")
	next.Language = r.PostFormValue("language")
	next.TimeZone = strings.TrimSpace(r.PostFormValue("time_zone"))
	next.ThemeID = optionalID(r.PostFormValue("active_theme"))
	next.SystemThemeID = optionalID(r.PostFormValue("system_theme"))

	if granted.Has(perms.ManagePermissions) {
		next.DefaultRoleID = optionalID(r.PostFormValue("default_role"))
		next.VerifiedRoleID = optionalID(r.PostFormValue("verified_role"))
		next.MembershipPasswordEnabled = r.PostFormValue("membership_password_enabled") != ""
		next.MembershipPassword = r.PostFormValue("membership_password")
		next.MembershipPasswordRoleID = optionalID(r.PostFormValue("membership_password_role"))
	}

	settings := db.SiteSettings{
		RatingMode: r.PostFormValue("rating_mode"),
		CreateTags: r.PostFormValue("can_user_create_tags"),
	}
	if problem := checkSite(loc, h.deps.Bundle, next, settings); problem != "" {
		return h.siteForm(w, r, loc, problem, "")
	}
	if err := h.deps.DB.SaveSite(ctx, &next, settings, granted.Has(perms.ManagePermissions)); err != nil {
		return err
	}
	h.note(r, db.AdminChanged, siteSlug, "", next.Title)
	redirect(w, Prefix+siteSlug+"/")
	return nil
}

func checkSite(loc *i18n.Localizer, bundle *i18n.Bundle, s db.Site, settings db.SiteSettings) string {
	switch {
	case !slugPattern.MatchString(s.Slug):
		return loc.T("admin.site-bad-slug")
	case s.Title == "":
		return loc.T("admin.site-no-title")
	case !site.ValidHost(s.Domain) || !site.ValidHost(s.MediaDomain):
		return loc.T("admin.site-bad-domain")
	case s.HomePage == "":
		return loc.T("admin.site-no-home")
	case !contains(emailPolicy, s.EmailPolicy):
		return loc.T("admin.site-bad-policy")
	case !bundle.Has(s.Language):
		return loc.T("admin.site-bad-language")
	case !timezone.Valid(s.TimeZone):
		return loc.T("admin.site-bad-time-zone")
	case !contains(ratingModes, settings.RatingMode) || !contains(tagModes, settings.CreateTags):
		return loc.T("admin.site-bad-mode")
	}
	return ""
}

func contains(list []string, want string) bool {
	for _, one := range list {
		if one == want {
			return true
		}
	}
	return false
}

func optionalID(raw string) *int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id == 0 {
		return nil
	}
	return &id
}
