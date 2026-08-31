package modules

import (
	"errors"
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
	want := `<a class="button" href="/scp:missing/edit/true">编辑</a>`
	if got != want {
		t.Errorf("renderButton(edit) = %q, want %q", got, want)
	}
}

func TestRenderButtonWithItsOwnText(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "edit", "text": "创建此页"}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="button" href="/missing/edit/true">创建此页</a>`
	if got != want {
		t.Errorf("renderButton(text) = %q, want %q", got, want)
	}
}

func TestRenderButtonWithEmptyText(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "missing"})

	got, err := renderButton(env, map[string]string{"type": "edit", "text": ""}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="button" href="/missing/edit/true"></a>`
	if got != want {
		t.Errorf("renderButton(text=\"\") = %q, want %q", got, want)
	}
}

func TestRenderButtonEscapes(t *testing.T) {
	env := buttonEnv(t, &db.Article{Category: "_default", Name: "a b&c"})

	got, err := renderButton(env, map[string]string{"type": "edit", "text": `<b>"x"</b>`}, "")
	if err != nil {
		t.Fatalf("renderButton() err = %v, want nil", err)
	}
	want := `<a class="button" href="/a b&amp;c/edit/true">&lt;b&gt;&quot;x&quot;&lt;/b&gt;</a>`
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
