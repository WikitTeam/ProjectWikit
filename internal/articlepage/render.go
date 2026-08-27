package articlepage

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/article"
	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pageconfig"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/pyjson"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	templateName = "_template"
	excerptLimit = 384
)

type body struct {
	html      string
	status    int
	excerpt   string
	image     string
	title     string
	rev       int
	updatedAt time.Time
	style     string
	redirect  string
}

func (h *Handler) render(req *request) (*result, error) {
	navTop, topStyle, err := h.nav(req, "nav:top")
	if err != nil {
		return nil, err
	}
	navSide, sideStyle, err := h.nav(req, "nav:side")
	if err != nil {
		return nil, err
	}

	canonical := h.canonicalURL(req)
	out, err := h.body(req, canonical)
	if err != nil {
		return nil, err
	}
	if out.redirect != "" {
		return &result{Status: http.StatusFound, Location: out.redirect}, nil
	}
	out.style = topStyle + sideStyle + out.style

	data, err := h.shellData(req, out, canonical, navTop, navSide)
	if err != nil {
		return nil, err
	}

	var page strings.Builder
	if err := h.shell(req.loc).Page(&page, data); err != nil {
		return nil, err
	}
	return &result{Status: out.status, Body: page.String()}, nil
}

func (h *Handler) body(req *request, canonical string) (body, error) {
	switch {
	case req.forbidden:
		html, err := h.shell(req.loc).Forbidden(req.name)
		return body{html: html, status: http.StatusForbidden}, err
	case req.article != nil:
		return h.articleBody(req, canonical)
	default:
		return h.missingBody(req)
	}
}

func (h *Handler) articleBody(req *request, canonical string) (body, error) {
	source, err := h.source(req)
	if err != nil {
		return body{}, err
	}
	vars := h.vars(req, req.article)
	source = page.PageVars(source, vars, 1, 1)
	source = page.ApplyTemplate(source, article.ThisPage(req.params, canonical))
	source = page.ThisVars(source, vars)

	info, err := h.pageInfo(req, req.article)
	if err != nil {
		return body{}, err
	}

	pc := h.context(req, req.article)
	html, err := h.deps.Engine.RenderHTML(req.ctx, source, info, h.callbacks(req, vars, pc), renderer.ModeArticle)
	if err != nil {
		return body{}, err
	}
	// The text pass gets a context of its own. Sharing one would let a module
	// that writes to it do so twice.
	text, err := h.deps.Engine.RenderText(req.ctx, source, info,
		h.callbacks(req, vars, h.context(req, req.article)), renderer.ModeArticle)
	if err != nil {
		return body{}, err
	}

	rev, err := h.deps.DB.LatestRevNumber(req.ctx, req.article.ID)
	if err != nil {
		return body{}, err
	}
	out := body{
		html:      html.Body,
		status:    pc.Status,
		excerpt:   excerpt(text.Body),
		title:     pc.Title,
		rev:       rev,
		updatedAt: req.article.UpdatedAt,
		style:     pc.ComputedStyle,
		redirect:  pc.RedirectTo,
		image:     pc.OGImage,
	}
	if pc.OGDescription != "" {
		out.excerpt = pc.OGDescription
	}
	return out, nil
}

func (h *Handler) context(req *request, source *db.Article) *page.Context {
	return page.NewContext(req.article, source, paramsMap(req.params), req.user)
}

func paramsMap(params article.Params) map[string]string {
	out := make(map[string]string, len(params))
	for _, param := range params {
		out[param.Key] = param.Value
	}
	return out
}

// A page named _template is its own content, which is what keeps the template
// from wrapping itself.
func (h *Handler) source(req *request) (string, error) {
	if req.article.Name == templateName {
		return "%%content%%", nil
	}
	found, err := h.deps.DB.ArticleByName(req.ctx, req.article.Category+":"+templateName)
	if errors.Is(err, db.ErrNotFound) {
		return "%%content%%", nil
	}
	if err != nil {
		return "", err
	}
	source, err := h.deps.DB.LatestSource(req.ctx, found.ID)
	if errors.Is(err, db.ErrNotFound) {
		return "%%content%%", nil
	}
	if err != nil {
		return "", err
	}
	return source, nil
}

func (h *Handler) missingBody(req *request) (body, error) {
	options, err := pyjson.Marshal(pyjson.Object{
		{Key: "page_id", Value: req.name},
		{Key: "pathParams", Value: pathParams(req.params)},
	})
	if err != nil {
		return body{}, err
	}
	category, _ := wikidot.Split(req.name)
	object, err := repo.NewPerms(req.ctx, h.deps.DB).Category(category)
	if err != nil {
		return body{}, err
	}
	subject, err := h.subject(req)
	if err != nil {
		return body{}, err
	}
	allow := wikidot.NameAllowed(req.name) && perms.Resolve(subject, object).Has(perms.CreateArticles)

	// The name is left out because the view never puts it in this template's
	// context, so the message it renders has an empty slot where it goes.
	html, err := h.shell(req.loc).NotFound(shell.NotFound{
		AllowCreate: allow,
		Options:     options,
	})
	return body{html: html, status: http.StatusNotFound}, err
}

// nav renders one of the two navigation pages. It gets its own callbacks and
// its own PageInfo, since the page it decorates is a different row.
func (h *Handler) nav(req *request, name string) (string, string, error) {
	found, err := h.deps.DB.ArticleByName(req.ctx, name)
	if errors.Is(err, db.ErrNotFound) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	source, err := h.deps.DB.LatestSource(req.ctx, found.ID)
	if errors.Is(err, db.ErrNotFound) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}

	vars := h.vars(req, req.article)
	info, err := h.pageInfo(req, found)
	if err != nil {
		return "", "", err
	}
	pc := h.context(req, found)
	html, err := h.deps.Engine.RenderHTML(req.ctx, page.ThisVars(source, vars), info,
		h.callbacks(req, vars, pc), renderer.ModeArticle)
	if err != nil {
		return "", "", err
	}
	return html.Body, pc.ComputedStyle, nil
}

func (h *Handler) callbacks(req *request, vars *page.Vars, pc *page.Context) *callbacks.Callbacks {
	users := printuser.New(req.loc, h.deps.Icons)
	store := repo.New(req.ctx, h.deps.DB, users, repo.Options{
		Loc:  req.loc,
		Site: req.site,
		User: req.user,
	})
	cb := callbacks.New(req.loc, store)
	cb.SetPageVars(vars)
	cb.SetContext(pc)
	return cb
}

func (h *Handler) vars(req *request, of *db.Article) *page.Vars {
	if of == nil {
		return nil
	}
	return page.NewVars(of, req.user, repo.NewVarSource(req.ctx, h.deps.DB, req.site.ID), req.loc)
}

func (h *Handler) pageInfo(req *request, source *db.Article) (renderer.PageInfo, error) {
	info := renderer.PageInfo{
		Site:        req.site.Slug,
		Domain:      req.site.Domain,
		MediaDomain: req.site.MediaDomain,
	}
	if source == nil {
		return info, nil
	}
	info.Page = source.Name
	info.Category = source.Category
	tags, err := h.deps.DB.ArticleTagNames(req.ctx, source.ID)
	if err != nil {
		return renderer.PageInfo{}, err
	}
	info.Tags = tags
	return info, nil
}

func (h *Handler) subject(req *request) (perms.Subject, error) {
	return repo.NewPerms(req.ctx, h.deps.DB).Subject(req.user, h.now())
}

var newlines = regexp.MustCompile(`\n+`)

// excerpt trims every line and collapses the blank ones, which is the shape
// og:description carries.
func excerpt(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	joined := newlines.ReplaceAllString(strings.Join(lines, "\n"), "\n")
	runes := []rune(joined)
	if len(runes) > excerptLimit {
		return string(runes[:excerptLimit]) + "..."
	}
	return joined
}

func pathParams(params article.Params) pyjson.Object {
	out := make(pyjson.Object, 0, len(params))
	for _, param := range params {
		var value any
		if !param.Bare {
			value = param.Value
		}
		out = append(out, pyjson.Field{Key: param.Key, Value: value})
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) shellData(req *request, out body, canonical, navTop, navSide string) (shell.Data, error) {
	theme, err := h.themeURL(req)
	if err != nil {
		return shell.Data{}, err
	}
	indexed, err := h.indexed(req)
	if err != nil {
		return shell.Data{}, err
	}
	crumbs, err := h.breadcrumbs(req)
	if err != nil {
		return shell.Data{}, err
	}
	tags, err := h.tagBlock(req)
	if err != nil {
		return shell.Data{}, err
	}
	login, err := h.loginStatus(req)
	if err != nil {
		return shell.Data{}, err
	}
	options, err := h.options(req)
	if err != nil {
		return shell.Data{}, err
	}

	title := firstNonEmpty(out.title, req.site.Title)
	return shell.Data{
		SiteName:          req.site.Title,
		SiteHeadline:      req.site.Headline,
		SiteTitle:         title,
		SiteIcon:          req.site.Icon,
		OGTitle:           title,
		OGDescription:     out.excerpt,
		OGImage:           out.image,
		OGURL:             canonical,
		NoIndex:           !indexed,
		GoogleTagID:       h.deps.GoogleTagID,
		ThemeURL:          theme,
		ComputedStyle:     out.style,
		NavTop:            navTop,
		NavSide:           navSide,
		Title:             out.title,
		Content:           out.html,
		Breadcrumbs:       crumbs,
		TagCategories:     tags,
		RevNumber:         out.rev,
		UpdatedAt:         out.updatedAt,
		LoginStatusConfig: login,
		OptionsConfig:     options,
	}, nil
}

func (h *Handler) themeURL(req *request) (string, error) {
	if req.site.ThemeID == nil {
		return site.DefaultThemeURL, nil
	}
	theme, err := h.deps.DB.ThemeByID(req.ctx, *req.site.ThemeID)
	if errors.Is(err, db.ErrNotFound) {
		return site.DefaultThemeURL, nil
	}
	if err != nil {
		return "", err
	}
	return site.ThemeURL(theme), nil
}

func (h *Handler) indexed(req *request) (bool, error) {
	category, _ := wikidot.Split(req.name)
	return h.deps.DB.CategoryIndexed(req.ctx, category)
}

func (h *Handler) breadcrumbs(req *request) ([]shell.Breadcrumb, error) {
	if req.article == nil {
		return nil, nil
	}
	chain, err := h.deps.DB.Breadcrumbs(req.ctx, req.article.ID)
	if err != nil {
		return nil, err
	}
	out := make([]shell.Breadcrumb, 0, len(chain))
	for i := range chain {
		out = append(out, shell.Breadcrumb{URL: "/" + chain[i].FullName(), Title: chain[i].Title})
	}
	return out, nil
}

func (h *Handler) tagBlock(req *request) ([]shell.TagCategory, error) {
	if req.article == nil {
		return nil, nil
	}
	categories, err := h.deps.DB.ArticleTagCategories(req.ctx, req.article.ID)
	if err != nil {
		return nil, err
	}
	out := make([]shell.TagCategory, 0, len(categories))
	for _, category := range categories {
		one := shell.TagCategory{Name: category.Name}
		for _, tag := range category.Tags {
			one.Tags = append(one.Tags, shell.Tag{Name: tag.Name, FullName: tag.FullName})
		}
		out = append(out, one)
	}
	return out, nil
}

func (h *Handler) loginStatus(req *request) (string, error) {
	status := pageconfig.LoginStatus{User: req.user}
	if req.user != nil {
		userRoles, err := h.deps.DB.RolesByUser(req.ctx, req.user.ID)
		if err != nil {
			return "", err
		}
		count, err := h.deps.DB.UnreadNotifications(req.ctx, req.user.ID)
		if err != nil {
			return "", err
		}
		subject, err := h.subject(req)
		if err != nil {
			return "", err
		}
		status.Roles = userRoles
		status.NotificationCount = count
		status.CanEditArticles = perms.Resolve(subject, nil).Has(perms.EditArticles)
	}
	return status.JSON(req.loc)
}

func (h *Handler) options(req *request) (string, error) {
	options := pageconfig.Options{
		PageID:         req.name,
		NormalizedName: req.name,
		HasArticle:     req.article != nil,
		Anonymous:      req.user == nil,
		Perms:          req.perms,
		PathParams:     req.params,
		Rating:         page.DisabledRating(),
	}

	tags, err := h.deps.DB.SiteCanCreateTags(req.ctx, req.site.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	options.CanCreateTags = tags == db.CreateTagsEnabled

	if req.user != nil {
		raw, err := h.deps.DB.UserPreference(req.ctx, req.user.ID,
			pageconfig.PreferenceSection, pageconfig.PreferenceAdvancedSourceEditor)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return "", err
		}
		options.Preferences.AdvancedSourceEditor = pageconfig.PreferenceEnabled(raw)
	}

	if req.article != nil {
		rating, err := h.rating(req)
		if err != nil {
			return "", err
		}
		options.Rating = rating

		info, err := h.deps.DB.CommentInfo(req.ctx, req.article.ID)
		if err != nil {
			return "", err
		}
		options.CommentCount = info.Count

		watching, err := h.watching(req, info.ThreadID)
		if err != nil {
			return "", err
		}
		options.IsWatching = watching
	}
	return options.JSON()
}

func (h *Handler) rating(req *request) (page.Rating, error) {
	siteMode, err := h.deps.DB.SiteRatingMode(req.ctx, req.site.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return page.Rating{}, err
	}
	categoryMode, err := h.deps.DB.CategoryRatingMode(req.ctx, req.article.Category)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return page.Rating{}, err
	}
	mode := page.RatingMode(siteMode, categoryMode)
	if mode == page.RatingModeDisabled {
		return page.RatingOf(mode, db.VoteStats{}), nil
	}
	stats, err := h.deps.DB.VoteStats(req.ctx, req.article.ID)
	if err != nil {
		return page.Rating{}, err
	}
	return page.RatingOf(mode, stats), nil
}

// watching asks about the page and about the thread the path names. A t that is
// not a number leaves the second question unasked, where Django raises.
func (h *Handler) watching(req *request, threadID int64) (bool, error) {
	if req.user == nil {
		return false, nil
	}
	onArticle, err := h.deps.DB.SubscribedToArticle(req.ctx, req.user.ID, req.article.ID)
	if err != nil {
		return false, err
	}
	if onArticle {
		return true, nil
	}
	fromPath, err := strconv.ParseInt(req.params.Get("t"), 10, 64)
	if err != nil {
		return false, nil
	}
	return h.deps.DB.SubscribedToThread(req.ctx, req.user.ID, fromPath)
}
