//go:build cgo && !nocgo

package cgo

import (
	"context"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func TestRenderHTMLComment(t *testing.T) {
	cases := []struct{ source, want string }{
		{"a [!-- x --] b", "<p>a  b</p>"},
		{"a [!--x--] b", "<p>a  b</p>"},
		{"a [!-- x--] b", "<p>a  b</p>"},
		{"a [!--http://example.com--] b", "<p>a  b</p>"},
		{"a [!--x@y.com--] b", "<p>a  b</p>"},
		{"a [!--x---] b", "<p>a  b</p>"},
		{"a [!--x--] b --] c", "<p>a  b --] c</p>"},
	}
	for _, c := range cases {
		got, err := New().RenderHTML(context.Background(), c.source, info(), newHost(), renderer.ModeArticle)
		if err != nil {
			t.Fatalf("RenderHTML(%q) err = %v, want nil", c.source, err)
		}
		if got.Body != c.want {
			t.Errorf("RenderHTML(%q) = %q, want %q", c.source, got.Body, c.want)
		}
	}
}

func TestRenderHTMLFormattingAcrossBlocks(t *testing.T) {
	cases := []struct{ source, want string }{
		{"**[[size 150%]]x**[[/size]]", `<p><strong><span style="font-size: 150%;">x</span></strong></p>`},
		{"**a [[span]]b** c[[/span]]", `<p><strong>a <span>b</span></strong> c</p>`},
		{`//[[span class="a"]]it// rest[[/span]]`, `<p><em><span class="a">it</span></em> rest</p>`},
		{"##red|[[size 150%]]red## rest[[/size]]", `<p><span style="color: red;"><span style="font-size: 150%;">red</span></span> rest</p>`},
		{"[[size 150%]]**x[[/size]]**", `<p><span style="font-size: 150%;"><strong>x</strong></span></p>`},
		{"[[size 150%]]//x[[/size]]//", `<p><span style="font-size: 150%;"><em>x</em></span></p>`},
		{"[[size 150%]]__x[[/size]]__", `<p><span style="font-size: 150%;"><u>x</u></span></p>`},
		{"[[size 150%]]--x[[/size]]--", `<p><span style="font-size: 150%;"><s>x</s></span></p>`},
		{`[[span class="a"]]##blue|x[[/span]]##`, `<p><span class="a"><span style="color: blue;">x</span></span></p>`},
		{"**[[span]]a[[/span]]**", `<p><strong><span>a</span></strong></p>`},
		{"[[span]]**a**[[/span]]", `<p><span><strong>a</strong></span></p>`},
		{"**a** [[span]]b[[/span]] **c**", `<p><strong>a</strong> <span>b</span> <strong>c</strong></p>`},
	}
	for _, c := range cases {
		got, err := New().RenderHTML(context.Background(), c.source, info(), newHost(), renderer.ModeArticle)
		if err != nil {
			t.Fatalf("RenderHTML(%q) err = %v, want nil", c.source, err)
		}
		if got.Body != c.want {
			t.Errorf("RenderHTML(%q) = %q, want %q", c.source, got.Body, c.want)
		}
	}
}
