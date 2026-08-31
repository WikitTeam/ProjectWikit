package modules

import (
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
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
		escape.HTML(env.Text("module-date-format-js")) + `">` +
		escape.HTML(serverDate(env, at)) + `</span>`
}

// The stamp is formatted in UTC and not the site's zone, which is what the
// stored value carries.
func serverDate(env module.Env, at time.Time) string {
	at = at.UTC()
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
		"minute", pad(at.Minute()))
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
