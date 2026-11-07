package modules

import (
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/timezone"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	ind8  = "        "
	ind12 = "            "
	ind16 = "                "
	ind20 = "                    "
	ind24 = "                        "
	ind28 = "                            "
	ind32 = "                                "
	ind36 = "                                    "
	ind40 = "                                        "
)

func forumSectionURL(id int64, name string) string {
	return "/forum/s-" + strconv.FormatInt(id, 10) + "/" + wikidot.Normalize(name)
}

func forumCategoryURL(id int64, name string) string {
	return "/forum/c-" + strconv.FormatInt(id, 10) + "/" + wikidot.Normalize(name)
}

func forumThreadURL(id int64, name string) string {
	return "/forum/t-" + strconv.FormatInt(id, 10) + "/" + wikidot.Normalize(name)
}

func forumPostURL(threadID int64, threadName string, postID int64) string {
	return forumThreadURL(threadID, threadName) + "#post-" + strconv.FormatInt(postID, 10)
}

func renderDate(env module.Env, at time.Time) string {
	return `<span class="odate w-date" style="display: inline" data-timestamp="` +
		strconv.FormatInt(at.UnixMilli(), 10) + `" data-format="` +
		escape.HTML(env.Text("module-date-format-js")) + `" data-ago-tip>` +
		escape.HTML(serverDate(env, at)) + `</span>`
}

func serverDate(env module.Env, at time.Time) string {
	at = at.In(siteZone(env))
	pad := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	return env.Text("module-date-format",
		"year", strconv.Itoa(at.Year()),
		"month", pad(int(at.Month())),
		"day", pad(at.Day()),
		"hour", pad(at.Hour()),
		"minute", pad(at.Minute()),
		"zone", timezone.Label(at))
}

func siteZone(env module.Env) *time.Location {
	if env.Site == nil {
		return time.UTC
	}
	return timezone.Load(env.Site.TimeZone)
}

func forumFailed(env module.Env) error {
	return &module.Error{Message: env.Text("module-failed", "name", env.Name)}
}

func setTitle(env module.Env, title string) {
	if env.Page != nil {
		env.Page.Title = title
	}
}

func setStatus(env module.Env, code int) {
	if env.Page != nil {
		env.Page.Status = code
	}
}

func sectionGrants(env module.Env, subject perms.Subject) (map[int64]perms.Set, error) {
	sections, err := env.Data.ForumSections()
	if err != nil {
		return nil, err
	}
	out := make(map[int64]perms.Set, len(sections))
	for i := range sections {
		out[sections[i].ID] = perms.Resolve(subject, env.Data.ForumSectionObject(&sections[i]))
	}
	return out, nil
}

// A category answers with the overrides of its section. The one holding page
// comments answers site-wide, since each comment thread follows its own page.
func categoryAllows(grants map[int64]perms.Set, sitewide perms.Set, c *db.ForumCategory, names ...string) bool {
	granted := sitewide
	if !c.IsForComments {
		granted = grants[c.SectionID]
	}
	for _, name := range names {
		if !granted.Has(name) {
			return false
		}
	}
	return true
}

func categoryGrants(env module.Env, subject perms.Subject, c *db.ForumCategory) (perms.Set, error) {
	if c.IsForComments {
		return perms.Resolve(subject, nil), nil
	}
	section, err := env.Data.ForumSection(c.SectionID)
	if err != nil {
		return perms.Set{}, err
	}
	return perms.Resolve(subject, env.Data.ForumSectionObject(section)), nil
}
