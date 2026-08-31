package modules

import (
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("time", renderTime) }

// Each field becomes its own date element so the reader's browser resolves it
// in their own zone, which the server cannot know.
var timeFields = map[string]string{
	"currentyear":   "%Y",
	"currentmonth":  "%m",
	"currentday":    "%d",
	"currenthour":   "%H",
	"currentminute": "%M",
	"currentsecond": "%S",
}

func renderTime(env module.Env, _ map[string]string, body string) (string, error) {
	if env.Render == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "time")}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	now := time.Now()
	stamps := map[string]string{}
	source := page.ApplyTemplate(strings.TrimSpace(body), func(name string) (string, bool) {
		format, ok := timeFields[normalizeTimeField(name)]
		if !ok {
			return "", false
		}
		token := timeToken(format)
		stamps[token] = dateSpan(now, format)
		return token, true
	})

	html, err := env.Render(source, pc)
	if err != nil {
		return "", err
	}
	return putChipsBack(html, stamps), nil
}

func timeToken(format string) string { return "pwikittime" + format[1:] + "stamp" }

func dateSpan(at time.Time, format string) string {
	return `<span class="odate w-date" data-timestamp="` +
		strconv.FormatInt(at.UnixMilli(), 10) + `" data-format="` + format +
		`" style="display: inline">` + escape.HTML(timeField(at, format)) + `</span>`
}

func timeField(at time.Time, format string) string {
	switch format {
	case "%Y":
		return strconv.Itoa(at.Year())
	case "%m":
		return pad2(int(at.Month()))
	case "%d":
		return pad2(at.Day())
	case "%H":
		return pad2(at.Hour())
	case "%M":
		return pad2(at.Minute())
	case "%S":
		return pad2(at.Second())
	}
	return ""
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// The spelling without the second r reaches the same field, so a page using it
// does not silently render the variable name instead.
func normalizeTimeField(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	if rest, ok := strings.CutPrefix(lowered, "curent"); ok {
		return "current" + rest
	}
	return lowered
}
