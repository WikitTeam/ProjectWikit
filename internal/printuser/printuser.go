// Package printuser renders the user chip that appears inside wikitext output.
package printuser

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	AnonAvatar    = "/-/static/images/anon_avatar.png"
	DefaultAvatar = "/-/static/images/default_avatar.png"
	WikidotAvatar = "/-/static/images/wikidot_avatar.png"
)

const (
	TypeNormal  = "normal"
	TypeWikidot = "wikidot"
	TypeSystem  = "system"
	TypeBot     = "bot"
)

type User struct {
	ID              int64
	Type            string
	Username        string
	WikidotUsername string
	DisplayName     string
	Avatar          string
	IsActive        bool
}

type Options struct {
	Avatar bool
	Hover  bool
}

type Renderer struct {
	loc  *i18n.Localizer
	icon roles.IconLoader
}

func New(loc *i18n.Localizer, icon roles.IconLoader) *Renderer {
	return &Renderer{loc: loc, icon: icon}
}

func (r *Renderer) text(id string) string {
	if r.loc == nil {
		return id
	}
	return r.loc.T(id)
}

func (r *Renderer) System(opts Options) string {
	return `<span class="printuser` + classAdd(opts) + `"><strong>` + r.text("user-system") + `</strong></span>`
}

func (r *Renderer) Anonymous(opts Options) string {
	name := r.text("user-anonymous")
	var b strings.Builder
	b.WriteString(`<span class="printuser` + classAdd(opts) + `">` + "\n                ")
	if opts.Avatar {
		b.WriteString("\n                    " + `<a onclick="return false;"><img class="small" src="` +
			escape.HTML(AnonAvatar) + `" alt="` + name + `"></a>` + "\n                ")
	}
	b.WriteString("\n                " + `<a onclick="return false;">` + name + `</a></span>`)
	return b.String()
}

// External is the wd: user nobody imported. The name is normalized as a page
// name rather than a user name, and the displayed form keeps the original.
func (r *Renderer) External(username string, opts Options) string {
	display := escape.HTML(username)
	name := escape.HTML(wikidot.Normalize(username))
	href := `https://www.wikidot.com/user:info/` + name

	var b strings.Builder
	b.WriteString(`<span class="printuser w-user` + classAdd(opts) +
		`" data-user-id="-1" data-user-name="` + name + `">` + "\n            ")
	if opts.Avatar {
		b.WriteString("\n                " + `<a href="` + href + `" target="_blank"><img class="small" src="` +
			escape.HTML(WikidotAvatar) + `" alt="` + display + `"></a>` + "\n            ")
	}
	b.WriteString("\n            " + `<a href="` + href + `" target="_blank">` + display + `</a></span>`)
	return b.String()
}

func (r *Renderer) User(u User, rs []roles.Role, opts Options) (string, error) {
	tails, err := r.Tails(u, rs)
	if err != nil {
		return "", err
	}

	avatar := WikidotAvatar
	display := "wd:" + firstNonEmpty(u.DisplayName, u.WikidotUsername)
	if u.Type != TypeWikidot {
		avatar = DefaultAvatar
		if u.Avatar != "" {
			avatar = "/local--files/" + u.Avatar
		}
		display = firstNonEmpty(u.DisplayName, u.Username)
	}
	display = escape.HTML(display)
	name := escape.HTML(u.URLName())

	var b strings.Builder
	b.WriteString(`<span class="printuser w-user` + classAdd(opts) + `" data-user-id="` +
		strconv.FormatInt(u.ID, 10) + `" data-user-name="` + name + `">` + "\n            ")
	if opts.Avatar {
		b.WriteString("\n                " + `<a href="/-/users/` + name + `"><img class="small" src="` +
			escape.HTML(avatar) + `" alt="` + display + `"></a>` + "\n            ")
	}
	b.WriteString("\n            " + `<a href="/-/users/` + name + `">` + display + `</a>` + "\n            ")
	if opts.Avatar {
		b.WriteString("\n                ")
		for _, icon := range tails.Icons {
			b.WriteString("\n                    " + `<span class="icon" ` + title(icon.Tooltip) +
				`><img src="data:image/svg+xml,` + escape.HTML(icon.Icon) + `"/></span>` + "\n                ")
		}
		b.WriteString("\n                ")
		for _, badge := range tails.Badges {
			border := ""
			if badge.ShowBorder {
				border = "border: solid 1px " + badge.TextColor
			}
			b.WriteString("\n                    " + `<span class="badge" ` + title(badge.Tooltip) +
				` style="background: ` + badge.Bg + `; color: ` + badge.TextColor + `; ` + border + `">` +
				badge.Text + `</span>` + "\n                ")
		}
		b.WriteString("\n            ")
	}
	b.WriteString("\n        </span>")
	return b.String(), nil
}

// Tails answers the banned and bot cases before roles are consulted at all, so
// neither state can be hidden by a role configuration.
func (r *Renderer) Tails(u User, rs []roles.Role) (roles.Tails, error) {
	switch {
	case !u.IsActive && u.Type != TypeWikidot:
		return roles.Tails{Badges: []roles.Badge{{
			Text:      r.text("user-banned"),
			Bg:        "#000000",
			TextColor: "#FFFFFF",
			Tooltip:   r.text("user-banned-tooltip"),
		}}}, nil
	case u.Type == TypeBot:
		return roles.Tails{Badges: []roles.Badge{{
			Text:      r.text("user-bot"),
			Bg:        "#77A",
			TextColor: "#FFFFFF",
			Tooltip:   r.text("user-bot-tooltip"),
		}}}, nil
	}
	return roles.NameTails(rs, r.icon)
}

func (u User) URLName() string {
	if u.Type == TypeWikidot {
		return firstNonEmpty(u.WikidotUsername, u.Username)
	}
	return u.Username
}

func classAdd(opts Options) string {
	if opts.Hover {
		return " avatarhover"
	}
	return ""
}

// The tooltip is dropped rather than emitted empty, which leaves two spaces
// behind.
func title(tooltip string) string {
	if tooltip == "" {
		return ""
	}
	return `title="` + tooltip + `"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
