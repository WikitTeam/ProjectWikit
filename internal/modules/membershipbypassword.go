package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("membershipbypassword", renderMembershipByPassword) }

const membershipSubmitPath = "/-/membership/password"

func renderMembershipByPassword(env module.Env, params map[string]string, _ string) (string, error) {
	// A site that never turned this on should not advertise that it exists,
	// so the module renders nothing at all rather than a disabled form.
	if env.Site == nil || !env.Site.MembershipPasswordEnabled {
		return "", nil
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	var b strings.Builder
	if notice := submittedNotice(env, pc, "membership"); notice != "" {
		b.WriteString(notice)
	}
	if env.User == nil {
		b.WriteString(noticeBlock(env.Text("module-membershipbypassword-login")))
		return b.String(), nil
	}

	source := ""
	if pc.Article != nil {
		source = pc.Article.FullName()
	}
	label := strings.TrimSpace(firstNonBlank(params["label"], params["text"]))
	if label == "" {
		label = env.Text("module-membershipbypassword-password")
	}

	b.WriteString(`<form class="w-membership-password" method="post" action="` + membershipSubmitPath + `">` + "\n")
	b.WriteString(hiddenField("csrfmiddlewaretoken", pc.CSRF))
	b.WriteString(hiddenField("page", source))
	b.WriteString(`<table class="form">` + "\n")
	b.WriteString(`<tr><td>` + escape.HTML(label) + `</td>` +
		`<td><input class="text" type="password" name="password" autocomplete="off"></td></tr>` + "\n")
	b.WriteString("</table>\n")
	b.WriteString(`<div class="buttons"><input class="button" type="submit" value="` +
		escape.HTML(env.Text("module-membershipbypassword-submit")) + `"></div>` + "\n")
	b.WriteString("</form>")
	return b.String(), nil
}
