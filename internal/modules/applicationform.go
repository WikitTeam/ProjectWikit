package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("applicationform", renderApplicationForm) }

const (
	kindTicket          = "ticket"
	kindMembershipApply = "membershipapply"

	ticketSubmitPath = "/-/tickets/submit"
	ticketMaxSubject = 200
)

func renderApplicationForm(env module.Env, params map[string]string, _ string) (string, error) {
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	kind := kindTicket
	if strings.EqualFold(strings.TrimSpace(firstNonBlank(params["type"], params["kind"])), kindMembershipApply) {
		kind = kindMembershipApply
	}

	var b strings.Builder
	if notice := submittedNotice(env, pc, "applied"); notice != "" {
		b.WriteString(notice)
	}
	if env.User == nil {
		b.WriteString(noticeBlock(env.Text("module-applicationform-login")))
		return b.String(), nil
	}

	source := ""
	if pc.Article != nil {
		source = pc.Article.FullName()
	}

	b.WriteString(`<form class="w-application-form" method="post" action="` + ticketSubmitPath + `">` + "\n")
	b.WriteString(hiddenField("csrfmiddlewaretoken", pc.CSRF))
	b.WriteString(hiddenField("kind", kind))
	b.WriteString(hiddenField("page", source))
	b.WriteString(`<table class="form">` + "\n")
	b.WriteString(subjectRow(env, params))
	b.WriteString(`<tr><td>` + escape.HTML(env.Text("module-applicationform-body")) + `</td>` +
		`<td><textarea name="body" rows="10" cols="60"></textarea></td></tr>` + "\n")
	b.WriteString("</table>\n")
	b.WriteString(`<div class="buttons"><input class="button" type="submit" value="` +
		escape.HTML(env.Text("module-applicationform-submit")) + `"></div>` + "\n")
	b.WriteString("</form>")
	return b.String(), nil
}

// One parameter carries both readings. A word a switch is spelled with turns
// the field on or off, and anything else is the subject itself.
func subjectRow(env module.Env, params map[string]string) string {
	value, given := params["title"]
	if !given {
		value, given = params["subject"]
	}
	if !given {
		return subjectInput(env)
	}
	if wanted, isSwitch := module.ParseBool(value); isSwitch {
		if wanted {
			return subjectInput(env)
		}
		return ""
	}
	return hiddenField("subject", strings.TrimSpace(value))
}

func subjectInput(env module.Env) string {
	return `<tr><td>` + escape.HTML(env.Text("module-applicationform-subject")) + `</td>` +
		`<td><input class="text" type="text" name="subject" maxlength="` +
		strconv.Itoa(ticketMaxSubject) + `"></td></tr>` + "\n"
}

func hiddenField(name, value string) string {
	return `<input type="hidden" name="` + name + `" value="` + escape.HTML(value) + `">` + "\n"
}

func noticeBlock(text string) string {
	return `<div class="error-block"><p>` + escape.HTML(text) + `</p></div>` + "\n"
}

// The outcome comes back as a path parameter rather than a query string
// because that is the only part of the URL a page render can see.
func submittedNotice(env module.Env, pc *page.Context, key string) string {
	switch pc.PathParams.Get(key) {
	case "ok":
		return `<div class="success-block"><p>` +
			escape.HTML(env.Text("module-"+key+"-ok")) + `</p></div>` + "\n"
	case "":
		return ""
	default:
		return noticeBlock(env.Text("module-" + key + "-failed"))
	}
}
