package modules

import (
	"errors"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func buttonEnv(t *testing.T, article *db.Article) module.Env {
	t.Helper()
	env := forumEnv(t)
	env.Name = "Button"
	env.Page = page.NewContext(article, article, nil, nil)
	return env
}

func TestRenderButtonEdit(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "scp", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "edit"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="edit" href="javascript:;" onclick="pwikit.edit(event)">edit</a>`
	if got != want {
		t.Errorf("renderButton(edit) = %q, want %q", got, want)
	}
}

func TestRenderButtonSetTags(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "set-tags", "tags": "-发现 +丢失", "text": "丢失"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="set-tags" href="javascript:;" onclick="pwikit.setTags(event, &#x27;-发现 +丢失&#x27;)">丢失</a>`
	if got != want {
		t.Errorf("renderButton(set-tags) = %q, want %q", got, want)
	}
}

func TestRenderButtonSetTagsWithoutTags(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "set-tags"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="set-tags" href="javascript:;" onclick="pwikit.setTags(event, &#x27;&#x27;)">set-tags</a>`
	if got != want {
		t.Errorf("renderButton(set-tags without tags) = %q, want %q", got, want)
	}
}

func TestRenderButtonSetTagsEscapesTheOperations(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "set-tags", "tags": `+it's`, "text": "x"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="set-tags" href="javascript:;" onclick="pwikit.setTags(event, &#x27;+it\&#x27;s&#x27;)">x</a>`
	if got != want {
		t.Errorf("renderButton(quote in tags) = %q, want %q", got, want)
	}
}

func TestRenderButtonWithItsOwnText(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "edit", "text": "创建此页"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="edit" href="javascript:;" onclick="pwikit.edit(event)">创建此页</a>`
	if got != want {
		t.Errorf("renderButton(text) = %q, want %q", got, want)
	}
}

func TestRenderButtonTrimsTheLabel(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "set-tags", "tags": "+a", "text": "丢失 "}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	if !strings.HasSuffix(got, ">丢失</a>") {
		t.Errorf("renderButton(text=%q) = %q, want the label without its trailing space", "丢失 ", got)
	}
}

func TestRenderButtonWithEmptyText(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "edit", "text": ""}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="edit" href="javascript:;" onclick="pwikit.edit(event)"></a>`
	if got != want {
		t.Errorf("renderButton(text=\"\") = %q, want %q", got, want)
	}
}

func TestRenderButtonLabelsWithTheTypeAsWritten(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "Set-Tags"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="set-tags" href="javascript:;" onclick="pwikit.setTags(event, &#x27;&#x27;)">Set-Tags</a>`
	if got != want {
		t.Errorf("renderButton(Set-Tags) = %q, want %q", got, want)
	}
}

func TestRenderButtonEscapesTheLabel(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "edit", "text": `<b>"x"</b>`}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="wiki-standalone-button" data-button-type="edit" href="javascript:;" onclick="pwikit.edit(event)">&lt;b&gt;&quot;x&quot;&lt;/b&gt;</a>`
	if got != want {
		t.Errorf("renderButton(escapes) = %q, want %q", got, want)
	}
}

func TestRenderButtonOfAnUnknownType(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	_, err := renderButton(env, map[string]string{"type": "print"}, "")
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) {
		t.Fatalf("renderButton(print) err = %v, want a module error", err)
	}
}

func TestRenderButtonWithoutAPage(t *testing.T) {
	env := forumEnv(t)
	env.Name = "Button"

	_, err := renderButton(env, map[string]string{"type": "edit"}, "")
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) {
		t.Fatalf("renderButton() err = %v, want a module error", err)
	}
}
