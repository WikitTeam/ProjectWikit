package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("comments", renderComments) }

func renderComments(env module.Env, _ map[string]string, _ string) (string, error) {
	if env.Page == nil || env.Page.Article == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}
	info, err := env.Data.CommentInfo(env.Page.Article.ID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<div class="comments-box">` + "\n" + ind8 +
		`<div class="options" id="comments-options-hidden">` +
		`<a href="javascript:;" onclick="pwikit.comments(event)">` +
		escape.HTML(env.Text("module-comments-show")) +
		`</a></div>` + "\n" + ind8 +
		`<div id="thread-container" class="thread-container" style="margin-top: 1em" data-thread="` +
		strconv.FormatInt(info.ThreadID, 10) + `"></div></div>`)
	return b.String(), nil
}
