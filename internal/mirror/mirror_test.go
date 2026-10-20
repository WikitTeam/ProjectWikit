package mirror

import (
	"slices"
	"testing"
)

func TestRefs(t *testing.T) {
	source := "[[include :Backrooms-Wiki-CN:component:Level-Class\n|class=0]]\n" +
		"[[include :scp-wiki:theme:black-highlighter-theme]]\n" +
		"[[include component:local]]\n" +
		"[[ include  :scp-wiki:theme:black-highlighter-theme | x=1]]\n" +
		"[[include :ah:module:leg]] text [[include :bad wiki:x]]"
	got := Refs(source)
	want := []Target{
		{"backrooms-wiki-cn", "component:level-class"},
		{"scp-wiki", "theme:black-highlighter-theme"},
		{"ah", "module:leg"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Refs() = %v, want %v", got, want)
	}
}

func TestRefsSkipsInertText(t *testing.T) {
	source := "@@[[include :a:raw]]@@\n[[code]]\n[[include :a:code]]\n[[/code]]\n[!-- [[include :a:comment]] --]\n[[include :a:real]]"
	if got := Refs(source); !slices.Equal(got, []Target{{"a", "real"}}) {
		t.Errorf("Refs() = %v, want [{a real}]", got)
	}
}

func TestKeyMatchesTheRenderedName(t *testing.T) {
	if got := Key(" SCP-Wiki ", "Component:Foo Bar"); got != (Target{"scp-wiki", "component:foo-bar"}) {
		t.Errorf("Key() = %v, want {scp-wiki component:foo-bar}", got)
	}
}

func TestKeyFollowsWikidotUnderscores(t *testing.T) {
	tests := []struct{ in, want string }{
		{"component:foo_bar", "component:foo-bar"},
		{"_template", "_template"},
		{"component:_inc", "component:_inc"},
		{"a_b_c", "a-b-c"},
	}
	for _, tt := range tests {
		if got := Key("w", tt.in).Page; got != tt.want {
			t.Errorf("Key(w, %q).Page = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTargetString(t *testing.T) {
	if got := (Target{"scp-wiki", "component:x"}).String(); got != ":scp-wiki:component:x" {
		t.Errorf("String() = %q, want :scp-wiki:component:x", got)
	}
}
