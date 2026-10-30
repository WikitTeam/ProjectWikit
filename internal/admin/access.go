package admin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	accessSlug       = "access"
	accessRoleTab    = "role"
	accessHoldersTab = "holders"
	accessLintTab    = "lint"
	accessLimit      = 300
)

func init() {
	register(screen{
		slug:  accessSlug,
		label: "admin.access",
		anyOf: []string{perms.ManageUsers, perms.ManageRoles, perms.ManagePermissions},
		serve: (*Handler).accessScreen,
	})
}

func (h *Handler) accessScreen(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	tab := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+accessSlug), "/")
	x, err := h.newExplainer(r.Context(), loc)
	if err != nil {
		return err
	}
	data := map[string]any{
		"Base":  Prefix + accessSlug + "/",
		"Tab":   tab,
		"Perms": permOptions(loc),
		"Roles": x.choices,
	}
	switch tab {
	case "":
		err = h.accessUser(r, x, data)
	case accessRoleTab:
		err = h.accessRole(r, x, data)
	case accessHoldersTab:
		err = h.accessHolders(r, x, data)
	case accessLintTab:
		err = h.accessLint(r, x, data)
	default:
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.access"), "access.html", data)
}

type permOption struct {
	Name  string
	Label string
}

type permGroup struct {
	Label string
	Perms []permOption
}

func permOptions(loc *i18n.Localizer) []permGroup {
	var out []permGroup
	for _, g := range perms.Catalog {
		group := permGroup{Label: loc.T("admin.perm-group-" + g.Key)}
		for _, name := range g.Names {
			group.Perms = append(group.Perms, permOption{Name: name, Label: loc.T("admin.perm-" + name)})
		}
		out = append(out, group)
	}
	return out
}

// Other screens offer a link here only to someone the link would not turn
// away.
func (h *Handler) screenOpens(ctx context.Context, slug string) bool {
	for _, s := range screens {
		if s.slug == slug {
			return s.opensFor(grantsFrom(ctx), instanceSuper(ctx))
		}
	}
	return false
}

func (h *Handler) accessHref(ctx context.Context, query string) string {
	if !h.screenOpens(ctx, accessSlug) {
		return ""
	}
	return Prefix + accessSlug + "/" + query
}

func (h *Handler) findUser(ctx context.Context, raw string) (*db.User, error) {
	name := strings.TrimPrefix(raw, "wd:")
	u, err := h.deps.DB.UserByUsername(ctx, name)
	if errors.Is(err, db.ErrNotFound) {
		u, err = h.deps.DB.UserByAnyName(ctx, wikidot.CanonicalizeUsername(name), name)
	}
	return u, err
}

type fact struct {
	Label string
	Value string
	Tone  string
}

type accessTarget struct {
	kind     string
	label    string
	link     string
	category string
	name     string
	article  *db.Article
	thread   *db.ForumThread
	post     *db.ForumThreadPost
	section  *db.ForumSection
}

var (
	threadRef  = regexp.MustCompile(`^t-(\d+)$`)
	sectionRef = regexp.MustCompile(`^s-(\d+)$`)
	postRef    = regexp.MustCompile(`^#?post-(\d+)$`)
)

// The field takes whatever a moderator has at hand, which is usually a link
// copied from the address bar.
func parseTarget(raw string) (kind string, id int64, name string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, ""
	}
	path, fragment := raw, ""
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		path, fragment = u.Path, u.Fragment
	} else if before, after, ok := strings.Cut(raw, "#"); ok && before != "" {
		path, fragment = before, after
	}
	path = strings.Trim(path, "/")
	if m := postRef.FindStringSubmatch(fragment); m != nil {
		path = "post-" + m[1]
	}
	first, _, _ := strings.Cut(strings.TrimPrefix(path, "forum/"), "/")
	for _, ref := range []struct {
		kind string
		re   *regexp.Regexp
	}{{"post", postRef}, {"thread", threadRef}, {"section", sectionRef}} {
		if m := ref.re.FindStringSubmatch(first); m != nil {
			n, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				return "", 0, ""
			}
			return ref.kind, n, ""
		}
	}
	return "page", 0, wikidot.Normalize(first)
}

func (h *Handler) findTarget(ctx context.Context, raw string) (*accessTarget, bool, error) {
	kind, id, name := parseTarget(raw)
	siteID := siteID(ctx)
	switch kind {
	case "":
		return nil, strings.TrimSpace(raw) == "", nil
	case "post":
		post, err := h.deps.DB.ForumPost(ctx, siteID, id)
		if err != nil {
			return nil, false, ignoreMissing(err)
		}
		thread, err := h.deps.DB.ForumThread(ctx, siteID, post.ThreadID)
		if err != nil {
			return nil, false, ignoreMissing(err)
		}
		t, err := h.threadTarget(ctx, thread)
		if err != nil {
			return nil, false, err
		}
		t.kind, t.post = "post", post
		t.label = "#post-" + strconv.FormatInt(post.ID, 10) + " · " + thread.Name
		t.link += "#post-" + strconv.FormatInt(post.ID, 10)
		return t, true, nil
	case "thread":
		thread, err := h.deps.DB.ForumThread(ctx, siteID, id)
		if err != nil {
			return nil, false, ignoreMissing(err)
		}
		t, err := h.threadTarget(ctx, thread)
		return t, err == nil, err
	case "section":
		section, err := h.deps.DB.ForumSection(ctx, siteID, id)
		if err != nil {
			return nil, false, ignoreMissing(err)
		}
		return &accessTarget{kind: "section", label: section.Name, link: "/forum/s-" + strconv.FormatInt(section.ID, 10), section: section}, true, nil
	}
	if name == "" {
		return nil, false, nil
	}
	article, err := h.deps.DB.ArticleByName(ctx, siteID, name)
	if errors.Is(err, db.ErrNotFound) {
		category, _ := wikidot.Split(name)
		return &accessTarget{kind: "new-page", label: name, link: "/" + name, category: category, name: name}, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &accessTarget{kind: "page", label: article.FullName(), link: "/" + article.FullName(), category: article.Category, name: article.FullName(), article: article}, true, nil
}

func ignoreMissing(err error) error {
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	return err
}

func (h *Handler) threadTarget(ctx context.Context, thread *db.ForumThread) (*accessTarget, error) {
	t := &accessTarget{kind: "thread", label: thread.Name, link: "/forum/t-" + strconv.FormatInt(thread.ID, 10), thread: thread}
	if thread.ArticleID != nil {
		article, err := h.deps.DB.ArticleByID(ctx, siteID(ctx), *thread.ArticleID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		if article != nil {
			t.category = article.Category
			t.article = article
		}
	}
	return t, nil
}

func (t *accessTarget) object(p *repo.Perms, u *db.User) (*perms.Object, error) {
	if t == nil {
		return nil, nil
	}
	switch t.kind {
	case "page":
		return p.Article(t.article, u)
	case "new-page":
		return p.Category(t.category)
	case "section":
		return p.ForumSection(t.section), nil
	case "thread":
		return p.ForumThread(t.thread, u)
	case "post":
		thread, err := p.ForumThread(t.thread, u)
		if err != nil {
			return nil, err
		}
		return p.ForumPost(t.post, thread, u), nil
	}
	return nil, nil
}

type explainer struct {
	loc     *i18n.Localizer
	site    *db.Site
	roles   map[int64]db.RoleChoice
	choices []db.RoleChoice
	cats    map[string]int64

	target    *accessTarget
	user      *db.User
	superHere bool
	sanctions map[perms.Sanction]*time.Time
}

func (h *Handler) newExplainer(ctx context.Context, loc *i18n.Localizer) (*explainer, error) {
	choices, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return nil, err
	}
	cats, err := h.deps.DB.AdminCategories(ctx, siteID(ctx))
	if err != nil {
		return nil, err
	}
	x := &explainer{loc: loc, site: site.FromContext(ctx), choices: choices, roles: map[int64]db.RoleChoice{}, cats: map[string]int64{}}
	for _, c := range choices {
		x.roles[c.ID] = c
	}
	for _, c := range cats {
		if c.ID != 0 {
			x.cats[c.Name] = c.ID
		}
	}
	return x, nil
}

func (x *explainer) roleName(id int64) string {
	if c, ok := x.roles[id]; ok {
		return escape.HTML(c.Label())
	}
	return "#" + strconv.FormatInt(id, 10)
}

func (x *explainer) roleHref(id int64) string {
	return Prefix + roleSlug + "/" + strconv.FormatInt(id, 10)
}

func (x *explainer) categoryName() string {
	if x.target == nil {
		return ""
	}
	if x.target.category == db.DefaultCategory {
		return "_default"
	}
	return escape.HTML(x.target.category)
}

func (x *explainer) perm(name string) string { return escape.HTML(x.loc.T("admin.perm-" + name)) }

func (x *explainer) until(at *time.Time, plain, timed string, args ...any) string {
	if at == nil {
		return x.loc.T(plain, args...)
	}
	return x.loc.T(timed, append(args, "when", localTime(*at))...)
}

func (x *explainer) cause(c perms.Cause) string {
	t := x.loc.T
	switch c.Reason {
	case perms.ReasonSuperuser:
		if x.user != nil && x.user.IsSuperuser {
			return t("admin.why-superuser-instance")
		}
		return t("admin.why-superuser-site")
	case perms.ReasonInactive:
		return x.until(x.user.InactiveUntil, "admin.why-inactive", "admin.why-inactive-until")
	case perms.ReasonNoGrant:
		return t("admin.why-no-grant")
	case perms.ReasonRoleAllow:
		return t("admin.why-role-allow", "role", x.roleName(c.RoleID))
	case perms.ReasonRoleDeny:
		return t("admin.why-role-deny", "role", x.roleName(c.RoleID))
	case perms.ReasonOverrideAllow:
		return t("admin.why-override-allow", "role", x.roleName(c.RoleID), "category", x.categoryName())
	case perms.ReasonOverrideDeny:
		return t("admin.why-override-deny", "role", x.roleName(c.RoleID), "category", x.categoryName())
	case perms.ReasonPageLocked:
		return t("admin.why-page-locked", "perm", x.perm(perms.LockArticles))
	case perms.ReasonPageAuthor:
		return t("admin.why-page-author")
	case perms.ReasonHiddenSection:
		return t("admin.why-hidden-section", "perm", x.perm(perms.ViewHiddenForumSections))
	case perms.ReasonThreadAuthor:
		return t("admin.why-thread-author")
	case perms.ReasonThreadLocked:
		return t("admin.why-thread-locked", "perm", x.perm(perms.LockForumThreads))
	case perms.ReasonPostAuthor:
		return t("admin.why-post-author")
	case perms.ReasonForumBarred:
		return x.until(x.user.ForumInactiveUntil, "admin.why-forum-barred", "admin.why-forum-barred-until")
	case perms.ReasonUnverified:
		return t("admin.why-unverified")
	case perms.ReasonSanction:
		kind := escape.HTML(t("admin.sanction-" + string(c.Sanction)))
		return x.until(x.sanctions[c.Sanction], "admin.why-sanction", "admin.why-sanction-until", "kind", kind)
	}
	return string(c.Reason)
}

type fix struct {
	Text string
	Href string
	Link string
}

func (x *explainer) fix(c perms.Cause, name string) fix {
	t := x.loc.T
	switch c.Reason {
	case perms.ReasonInactive, perms.ReasonForumBarred:
		return fix{Text: t("admin.fix-instance-account"), Href: x.userHref(), Link: t("admin.fix-open-user")}
	case perms.ReasonNoGrant:
		return fix{Text: t("admin.fix-no-grant", "perm", x.perm(name)), Href: Prefix + roleSlug + "/", Link: t("admin.roles")}
	case perms.ReasonRoleDeny:
		return fix{Text: t("admin.fix-role-deny", "role", x.roleName(c.RoleID)), Href: x.roleHref(c.RoleID), Link: t("admin.fix-open-role")}
	case perms.ReasonOverrideDeny:
		f := fix{Text: t("admin.fix-override-deny", "role", x.roleName(c.RoleID), "category", x.categoryName())}
		if x.target != nil {
			if id, ok := x.cats[x.target.category]; ok {
				f.Href, f.Link = Prefix+pageCategorySlug+"/"+strconv.FormatInt(id, 10), t("admin.fix-open-category")
			}
		}
		return f
	case perms.ReasonPageLocked:
		return fix{Text: t("admin.fix-page-locked", "perm", x.perm(perms.LockArticles))}
	case perms.ReasonHiddenSection:
		return fix{Text: t("admin.fix-hidden-section", "perm", x.perm(perms.ViewHiddenForumSections)), Href: Prefix + sectionSlug + "/", Link: t("admin.forum-sections")}
	case perms.ReasonThreadLocked:
		return fix{Text: t("admin.fix-thread-locked", "perm", x.perm(perms.LockForumThreads))}
	case perms.ReasonUnverified:
		return fix{Text: t("admin.fix-unverified"), Href: Prefix + siteSlug + "/", Link: t("admin.site")}
	case perms.ReasonSanction:
		return fix{Text: t("admin.fix-sanction"), Href: x.userHref(), Link: t("admin.fix-open-user")}
	}
	return fix{}
}

func (x *explainer) userHref() string {
	if x.user == nil {
		return ""
	}
	return Prefix + userSlug + "/" + strconv.FormatInt(x.user.ID, 10)
}

type roleLine struct {
	Name     string
	Href     string
	Base     string
	Override string
	Result   string
	Tone     string
}

type stepView struct {
	Number   int
	Title    string
	State    string
	Label    string
	Decisive bool
	Lines    []string
	Roles    []roleLine
}

type verdictView struct {
	Name    string
	Label   string
	Allowed bool
	Step    string
	Summary string
	Also    []string
	Fix     fix
	Steps   []stepView
	Detail  string
}

var effectState = map[perms.Effect]string{
	perms.EffectSkip:  "skip",
	perms.EffectNone:  "none",
	perms.EffectAllow: "allow",
	perms.EffectDeny:  "deny",
}

func (x *explainer) value(v perms.Value) string {
	switch v {
	case perms.Allow:
		return x.loc.T("admin.perm-allow")
	case perms.Deny:
		return x.loc.T("admin.perm-deny")
	}
	return x.loc.T("admin.perm-inherit")
}

// Only the causes that pushed the step's way are kept, so a thread whose
// author is also locked out reads as the lock alone.
func (x *explainer) stepText(s perms.StepResult) string {
	var parts []string
	for _, c := range s.Causes {
		if s.Effect == perms.EffectNone || c.Effect == s.Effect {
			parts = append(parts, x.cause(c))
		}
	}
	return strings.Join(parts, x.loc.T("admin.access-join"))
}

func (x *explainer) verdict(d perms.Decision, full bool) verdictView {
	main, also := d.Decisive()
	v := verdictView{
		Name:    d.Name,
		Label:   x.loc.T("admin.perm-" + d.Name),
		Allowed: d.Allowed,
		Step:    x.loc.T("admin.access-step-" + strconv.Itoa(int(main.Step))),
		Summary: x.stepText(main),
	}
	for _, s := range also {
		v.Also = append(v.Also, x.stepText(s))
	}
	if !d.Allowed && len(main.Causes) > 0 {
		for _, c := range main.Causes {
			if c.Effect == perms.EffectDeny || c.Reason == perms.ReasonNoGrant {
				v.Fix = x.fix(c, d.Name)
				break
			}
		}
	}
	if !full {
		return v
	}
	settled := 0
	if d.Step(perms.StepAccount).Effect == perms.EffectDeny {
		settled = int(perms.StepAccount)
	} else if d.Step(perms.StepSuperuser).Effect == perms.EffectAllow {
		settled = int(perms.StepSuperuser)
	}
	for _, s := range d.Steps {
		n := int(s.Step)
		view := stepView{
			Number:   n,
			Title:    x.loc.T("admin.access-step-" + strconv.Itoa(n)),
			State:    effectState[s.Effect],
			Label:    x.loc.T("admin.access-effect-" + effectState[s.Effect]),
			Decisive: s.Step == main.Step,
		}
		if s.Step == perms.StepRoles && s.Effect == perms.EffectNone {
			view.Label = x.loc.T("admin.access-effect-unset")
		}
		for _, c := range s.Causes {
			view.Lines = append(view.Lines, x.cause(c))
		}
		if s.Step == perms.StepRoles {
			view.Lines = nil
			for _, rv := range d.Roles {
				line := roleLine{Name: x.roleName(rv.RoleID), Href: x.roleHref(rv.RoleID), Base: x.value(rv.Base), Override: "—", Result: x.value(rv.Result), Tone: toneOf(rv.Result)}
				if rv.HasOverride {
					line.Override = x.value(rv.Override)
				}
				view.Roles = append(view.Roles, line)
			}
			if s.Effect == perms.EffectNone && len(d.Roles) > 0 {
				view.Lines = []string{x.loc.T("admin.why-no-grant")}
			}
		}
		if len(view.Lines) == 0 && len(view.Roles) == 0 {
			switch {
			case settled != 0 && n > settled:
				view.Lines = []string{x.loc.T("admin.access-settled")}
			case s.Effect == perms.EffectSkip:
				view.Lines = []string{x.loc.T("admin.access-skip-" + strconv.Itoa(n))}
			}
		}
		v.Steps = append(v.Steps, view)
	}
	return v
}

type accessForm struct {
	User      string
	Perm      string
	Target    string
	Anonymous bool
	Role      int64
	Category  string
}

func (h *Handler) accessUser(r *http.Request, x *explainer, data map[string]any) error {
	ctx := r.Context()
	q := r.URL.Query()
	form := accessForm{
		User:      strings.TrimSpace(q.Get("user")),
		Perm:      q.Get("perm"),
		Target:    strings.TrimSpace(q.Get("target")),
		Anonymous: q.Get("anonymous") != "",
	}
	data["Form"] = form
	if form.User == "" && !form.Anonymous {
		if form.Perm != "" || form.Target != "" {
			data["Problem"] = x.loc.T("admin.access-need-user")
		}
		return nil
	}
	if form.Perm != "" && perms.GroupOf(form.Perm) == "" {
		data["Problem"] = x.loc.T("admin.access-bad-perm")
		return nil
	}
	var u *db.User
	if !form.Anonymous {
		found, err := h.findUser(ctx, form.User)
		if errors.Is(err, db.ErrNotFound) {
			data["Problem"] = x.loc.T("admin.access-no-user", "name", form.User)
			return nil
		}
		if err != nil {
			return err
		}
		u = found
	}
	target, ok, err := h.findTarget(ctx, form.Target)
	if err != nil {
		return err
	}
	if !ok {
		data["Problem"] = x.loc.T("admin.access-no-target", "target", form.Target)
		return nil
	}
	now := time.Now()
	p := repo.NewPerms(ctx, h.deps.DB)
	subject, err := p.Subject(u, now)
	if err != nil {
		return err
	}
	object, err := target.object(p, u)
	if err != nil {
		return err
	}
	if err := h.prepareFor(ctx, x, target, u, subject, now); err != nil {
		return err
	}
	data["Who"] = h.whoFacts(ctx, x, u, subject, now)
	data["What"] = x.whatFacts(object)
	if form.Perm != "" {
		data["Verdict"] = x.verdict(perms.Explain(subject, object, form.Perm), true)
		return nil
	}
	data["Table"] = x.table(subject, object, r.URL.Query())
	return nil
}

func (h *Handler) prepareFor(ctx context.Context, x *explainer, target *accessTarget, u *db.User, subject perms.Subject, now time.Time) error {
	x.target, x.user, x.superHere, x.sanctions = target, u, false, map[perms.Sanction]*time.Time{}
	if u == nil {
		return nil
	}
	x.superHere = subject.Superuser && !u.IsSuperuser
	list, err := h.deps.DB.MemberSanctions(ctx, siteID(ctx), u.ID)
	if err != nil {
		return err
	}
	for _, s := range list {
		if s.Until == nil || s.Until.After(now) {
			x.sanctions[perms.Sanction(s.Kind)] = s.Until
		}
	}
	return nil
}

type tableGroup struct {
	Label string
	Rows  []verdictView
}

func (x *explainer) table(subject perms.Subject, object *perms.Object, q url.Values) []tableGroup {
	var out []tableGroup
	for _, g := range perms.Catalog {
		group := tableGroup{Label: x.loc.T("admin.perm-group-" + g.Key)}
		for _, name := range g.Names {
			v := x.verdict(perms.Explain(subject, object, name), false)
			link := url.Values{}
			for k, vals := range q {
				link[k] = vals
			}
			link.Set("perm", name)
			v.Detail = Prefix + accessSlug + "/?" + link.Encode()
			group.Rows = append(group.Rows, v)
		}
		out = append(out, group)
	}
	return out
}

func yesNo(loc *i18n.Localizer, yes bool) string {
	if yes {
		return loc.T("admin.yes")
	}
	return loc.T("admin.no")
}

func (h *Handler) whoFacts(ctx context.Context, x *explainer, u *db.User, subject perms.Subject, now time.Time) []fact {
	t := x.loc.T
	var out []fact
	if u == nil {
		out = append(out, fact{Label: t("admin.access-who"), Value: t("admin.access-anonymous")})
	} else {
		out = append(out, fact{Label: t("admin.access-who"), Value: `<a href="` + escape.HTML(x.userHref()) + `">` + escape.HTML(u.DisplayLabel()) + `</a>`})
		super := t("admin.no")
		tone := ""
		switch {
		case u.IsSuperuser:
			super, tone = t("admin.access-super-instance"), "allow"
		case x.superHere:
			super, tone = t("admin.access-super-site"), "allow"
		}
		out = append(out, fact{Label: t("admin.access-super"), Value: super, Tone: tone})
		state, tone := t("admin.access-normal"), ""
		if !u.ActiveAt(now) {
			state, tone = x.until(u.InactiveUntil, "admin.why-inactive", "admin.why-inactive-until"), "deny"
		}
		out = append(out, fact{Label: t("admin.access-account"), Value: state, Tone: tone})
		state, tone = t("admin.access-normal"), ""
		if !u.ForumActiveAt(now) {
			state, tone = x.until(u.ForumInactiveUntil, "admin.why-forum-barred", "admin.why-forum-barred-until"), "deny"
		}
		out = append(out, fact{Label: t("admin.access-forum"), Value: state, Tone: tone})
		state, tone = t("admin.access-normal"), ""
		if !u.MessagesAt(now) {
			state, tone = x.until(u.DirectMessagesUntil, "admin.access-dm-off", "admin.access-dm-off-until"), "deny"
		}
		out = append(out, fact{Label: t("admin.access-dm"), Value: state, Tone: tone})
	}
	var roles []string
	for _, role := range subject.Roles {
		roles = append(roles, `<a href="`+escape.HTML(x.roleHref(role.ID))+`">`+x.roleName(role.ID)+`</a>`)
	}
	value := strings.Join(roles, t("admin.access-list-join"))
	if value == "" {
		value = t("admin.access-none")
	}
	out = append(out, fact{Label: t("admin.roles"), Value: value})
	if u != nil {
		var list []string
		for _, kind := range subject.Sanctions {
			label := escape.HTML(t("admin.sanction-" + string(kind)))
			list = append(list, x.until(x.sanctions[kind], "admin.access-sanction", "admin.access-sanction-until", "kind", label))
		}
		value, tone := strings.Join(list, t("admin.access-list-join")), "deny"
		if value == "" {
			value, tone = t("admin.access-none"), ""
		}
		out = append(out, fact{Label: t("admin.sanctions"), Value: value, Tone: tone})
	}
	return out
}

func (x *explainer) whatFacts(object *perms.Object) []fact {
	t := x.loc.T
	target := x.target
	if target == nil {
		return []fact{{Label: t("admin.access-target"), Value: t("admin.access-sitewide")}}
	}
	link := `<a href="` + escape.HTML(target.link) + `" target="_blank" rel="noreferrer">` + escape.HTML(target.label) + `</a>`
	out := []fact{{Label: t("admin.access-target"), Value: t("admin.access-kind-"+target.kind, "name", link)}}
	if target.category != "" {
		value := x.categoryName()
		if id, ok := x.cats[target.category]; ok {
			value = `<a href="` + escape.HTML(Prefix+pageCategorySlug+"/"+strconv.FormatInt(id, 10)) + `">` + value + `</a>`
		}
		overrides := 0
		if object != nil {
			overrides = len(object.Overrides)
		}
		out = append(out, fact{Label: t("admin.access-category"), Value: t("admin.access-overrides", "category", value, "n", overrides)})
	}
	switch target.kind {
	case "page":
		out = append(out, fact{Label: t("admin.access-locked"), Value: yesNo(x.loc, target.article.Locked), Tone: map[bool]string{true: "deny"}[target.article.Locked]})
		if x.user != nil && object != nil {
			out = append(out, fact{Label: t("admin.access-author"), Value: yesNo(x.loc, object.Author)})
		}
	case "new-page":
		if !wikidot.NameAllowed(target.name) {
			out = append(out, fact{Label: t("admin.access-note"), Value: t("admin.access-bad-name"), Tone: "deny"})
		}
	case "section":
		out = append(out, fact{Label: t("admin.access-hidden"), Value: yesNo(x.loc, target.section.IsHiddenForUsers)})
	case "thread", "post":
		out = append(out, fact{Label: t("admin.access-locked"), Value: yesNo(x.loc, target.thread.IsLocked), Tone: map[bool]string{true: "deny"}[target.thread.IsLocked]})
		if x.user != nil && object != nil {
			out = append(out, fact{Label: t("admin.access-author"), Value: yesNo(x.loc, object.Author)})
		}
		if target.kind == "post" {
			out = append(out, fact{Label: t("admin.access-note"), Value: t("admin.access-post-note")})
		} else if target.article != nil {
			out = append(out, fact{Label: t("admin.access-note"), Value: t("admin.access-comments-note", "perm", x.perm(perms.CommentArticles))})
		}
	}
	return out
}
