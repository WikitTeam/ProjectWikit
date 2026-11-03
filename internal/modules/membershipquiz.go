package modules

import (
	"crypto/rand"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/quiz"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

func init() { module.Register("membershipquiz", renderMembershipQuiz) }

const membershipQuizPath = "/-/membership/quiz"

func renderMembershipQuiz(env module.Env, _ map[string]string, _ string) (string, error) {
	q, err := env.Data.MembershipQuiz()
	if err != nil {
		return "", err
	}
	questions := quiz.Parse(q.Questions)
	// Like the password module, a site that never turned this on shows nothing.
	if !q.Enabled || q.RoleID == nil || len(questions) == 0 {
		return "", nil
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	var b strings.Builder
	b.WriteString(submittedNotice(env, pc, "membership-quiz"))
	if env.User == nil {
		b.WriteString(noticeBlock(env.Text("module-membershipquiz-login")))
		return b.String(), nil
	}
	held, err := env.Data.RolesByUser(env.User.ID)
	if err != nil {
		return "", err
	}
	if slices.ContainsFunc(held, func(r roles.Role) bool { return r.ID == *q.RoleID }) {
		b.WriteString(`<div class="success-block"><p>` + escape.HTML(env.Text("module-membershipquiz-member")) + "</p></div>")
		return b.String(), nil
	}
	attempts, err := env.Data.QuizAttempts(env.User.ID)
	if err != nil {
		return "", err
	}
	left, until := quiz.State(attempts, q.Tries, quiz.Hours(q.CooldownHours), time.Now())
	if left == 0 {
		b.WriteString(noticeBlock(env.Text("module-membershipquiz-locked",
			"time", until.In(siteZone(env)).Format("2006-01-02 15:04"))))
		return b.String(), nil
	}

	source := ""
	if pc.Article != nil {
		source = pc.Article.FullName()
	}
	b.WriteString(`<form class="w-membership-quiz" method="post" action="` + membershipQuizPath + `">` + "\n")
	b.WriteString(hiddenField("csrfmiddlewaretoken", pc.CSRF))
	b.WriteString(hiddenField("page", source))
	b.WriteString(`<p class="w-membership-quiz-tries">` + escape.HTML(env.Text("module-membershipquiz-tries", "count", left)) + "</p>\n")
	for n, qi := range shuffled(len(questions)) {
		question := questions[qi]
		kind := "radio"
		if question.Multiple() {
			kind = "checkbox"
		}
		name := "q" + strconv.Itoa(qi)
		b.WriteString(`<fieldset class="w-membership-quiz-question">` + "\n")
		b.WriteString(`<legend>` + strconv.Itoa(n+1) + ". " + escape.HTML(question.Text))
		b.WriteString("</legend>\n")
		for _, oi := range shuffled(len(question.Options)) {
			b.WriteString(`<label><input type="` + kind + `" name="` + name + `" value="` + strconv.Itoa(oi) + `"> ` +
				escape.HTML(question.Options[oi].Text) + "</label><br>\n")
		}
		b.WriteString("</fieldset>\n")
	}
	b.WriteString(`<div class="buttons"><input class="button" type="submit" value="` +
		escape.HTML(env.Text("module-membershipquiz-submit")) + `"></div>` + "\n")
	b.WriteString("</form>")
	return b.String(), nil
}

// Every reader gets their own order, so an answer key passed around by position
// is worth nothing.
func shuffled(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			continue
		}
		out[i], out[j.Int64()] = out[j.Int64()], out[i]
	}
	return out
}
