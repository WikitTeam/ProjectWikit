package modules

import (
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() { module.Register("newpage", renderNewPage) }

func renderNewPage(env module.Env, params map[string]string, _ string) (string, error) {
	// An empty submit= is the caller asking for a blank button, so a missing
	// key and a present one are not the same thing here.
	submit, ok := params["submit"]
	if !ok {
		submit = env.Text("module-newpage-submit")
	}
	config, err := wikijson.Marshal(wikijson.Object{{Key: "category", Value: params["category"]}})
	if err != nil {
		return "", err
	}

	// new-page-box appears nowhere else in the tree because the base stylesheet
	// reserves it for exactly this form.
	return `<div class="new-page-box new-page-form w-newpage-module" data-config="` + escape.HTML(config) + `">
  <form method="get" action="">
    <input class="text" name="new_fullname" type="text" size="30" placeholder="` +
		escape.HTML(params["example"]) + `" required="true">
    <input class="button" value="` + escape.HTML(submit) + `" type="submit">
  </form>
</div>`, nil
}
