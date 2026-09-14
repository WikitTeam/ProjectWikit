package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("files", renderFiles) }

// Wikidot wraps the table in a div named after a random number it uses for its
// own paging, which no stylesheet can select, so nothing here answers to it.
func renderFiles(env module.Env, _ map[string]string, _ string) (string, error) {
	if env.Page == nil || env.Page.Article == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}
	files, err := env.Data.ArticleFiles(env.Page.Article.ID)
	if err != nil {
		return "", err
	}

	base := "/local--files/" + env.Page.Article.FullName() + "/"

	var b strings.Builder
	b.WriteString(`<table class="page-files"><tr><th>` + escape.HTML(env.Text("module-files-name")) +
		`</th><th>` + escape.HTML(env.Text("module-files-type")) +
		`</th><th>` + escape.HTML(env.Text("module-files-size")) + `</th><th></th></tr>`)
	// The last column waits on a file reporting more than the table shows, and
	// stays present so a theme sizing four columns still lines up.
	for _, f := range files {
		b.WriteString(`<tr><td><a href="` + escape.HTML(base+f.Name) + `">` + escape.HTML(f.Name) +
			`</a></td><td>` + escape.HTML(f.MimeType) +
			`</td><td>` + escape.HTML(fileSize(f.Size)) + `</td><td></td></tr>`)
	}
	b.WriteString("</table>\n" + ind8 + `<p style="text-align: center;" class="manage-attachments-link">` +
		`<a href="javascript:;" onclick="pwikit.files(event)">` +
		escape.HTML(env.Text("module-files-manage")) + `</a></p>`)
	return b.String(), nil
}

// Wikidot divides by 1024 and still writes kB, and a theme lining these up in a
// column would notice a digit more or fewer.
func fileSize(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " bytes"
	}
	v := float64(n) / unit
	if v < unit {
		return strconv.FormatFloat(v, 'f', 2, 64) + " kB"
	}
	return strconv.FormatFloat(v/unit, 'f', 2, 64) + " MB"
}
