package sidecar

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

type fakeSidecar struct {
	t   *testing.T
	in  io.Reader
	out io.Writer
}

func (f *fakeSidecar) recv() map[string]any {
	f.t.Helper()
	var head [4]byte
	if _, err := io.ReadFull(f.in, head[:]); err != nil {
		f.t.Fatalf("fake sidecar read length err = %v, want nil", err)
	}
	buf := make([]byte, binary.BigEndian.Uint32(head[:]))
	if _, err := io.ReadFull(f.in, buf); err != nil {
		f.t.Fatalf("fake sidecar read body err = %v, want nil", err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		f.t.Fatalf("fake sidecar unmarshal err = %v, want nil", err)
	}
	return m
}

func (f *fakeSidecar) send(v any) {
	f.t.Helper()
	buf, err := json.Marshal(v)
	if err != nil {
		f.t.Fatalf("fake sidecar marshal err = %v, want nil", err)
	}
	var head [4]byte
	binary.BigEndian.PutUint32(head[:], uint32(len(buf)))
	f.out.Write(head[:])
	f.out.Write(buf)
}

func newPair(t *testing.T, serve func(f *fakeSidecar)) *Renderer {
	t.Helper()
	toServer, fromClient := io.Pipe()
	toClient, fromServer := io.Pipe()

	go func() {
		defer fromServer.Close()
		serve(&fakeSidecar{t: t, in: toServer, out: fromServer})
	}()

	t.Cleanup(func() { fromClient.Close() })
	return newOver(fromClient, toClient)
}

func resultMsg(extra map[string]any) map[string]any {
	m := map[string]any{"type": "result", "body": "<p>ok</p>"}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestRenderHTMLMapsResult(t *testing.T) {
	r := newPair(t, func(f *fakeSidecar) {
		f.recv()
		f.send(resultMsg(map[string]any{
			"included_pages": []string{"a"},
			"linked_pages":   []string{"b", "c"},
			"code":           [][2]string{{"rust", "fn main() {}"}},
			"html":           []string{"<b>x</b>"},
		}))
	})

	got, err := r.RenderHTML(context.Background(), "src", renderer.PageInfo{}, renderer.NopCallbacks{}, renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML() err = %v, want nil", err)
	}
	if got.Body != "<p>ok</p>" {
		t.Errorf("Body = %q, want %q", got.Body, "<p>ok</p>")
	}
	if len(got.LinkedPages) != 2 || got.LinkedPages[1] != "c" {
		t.Errorf("LinkedPages = %v, want [b c]", got.LinkedPages)
	}
	if len(got.Code) != 1 || got.Code[0].Language != "rust" || got.Code[0].Source != "fn main() {}" {
		t.Errorf("Code = %+v, want [{rust fn main() {}}]", got.Code)
	}
}

func TestRenderSendsRequestFields(t *testing.T) {
	var req map[string]any
	r := newPair(t, func(f *fakeSidecar) {
		req = f.recv()
		f.send(resultMsg(nil))
	})

	info := renderer.PageInfo{Page: "173", Category: "scp", Domain: "example.org", Tags: []string{"t"}}
	if _, err := r.RenderText(context.Background(), "源码", info, renderer.NopCallbacks{}, renderer.ModeSystem); err != nil {
		t.Fatalf("RenderText() err = %v, want nil", err)
	}

	if req["type"] != "render" {
		t.Errorf("type = %v, want render", req["type"])
	}
	if req["op"] != "text" {
		t.Errorf("op = %v, want text", req["op"])
	}
	if req["mode"] != "system" {
		t.Errorf("mode = %v, want system", req["mode"])
	}
	if req["source"] != "源码" {
		t.Errorf("source = %v, want %q", req["source"], "源码")
	}
	pi, _ := req["page_info"].(map[string]any)
	if pi["page"] != "173" || pi["category"] != "scp" || pi["domain"] != "example.org" {
		t.Errorf("page_info = %v, want page=173 category=scp domain=example.org", pi)
	}
}

func TestRenderRejectsInvalidMode(t *testing.T) {
	r := newPair(t, func(f *fakeSidecar) {})
	if _, err := r.RenderHTML(context.Background(), "s", renderer.PageInfo{}, renderer.NopCallbacks{}, "nonsense"); err == nil {
		t.Error("RenderHTML(mode=nonsense) err = nil, want non-nil")
	}
}

func TestRenderRejectsNilCallbacks(t *testing.T) {
	r := newPair(t, func(f *fakeSidecar) {})
	if _, err := r.RenderHTML(context.Background(), "s", renderer.PageInfo{}, nil, renderer.ModeArticle); err == nil {
		t.Error("RenderHTML(cb=nil) err = nil, want non-nil")
	}
}

func TestRenderPropagatesSidecarError(t *testing.T) {
	r := newPair(t, func(f *fakeSidecar) {
		f.recv()
		f.send(map[string]any{"type": "error", "message": "boom"})
	})
	_, err := r.RenderHTML(context.Background(), "s", renderer.PageInfo{}, renderer.NopCallbacks{}, renderer.ModeArticle)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want substring %q", err, "boom")
	}
}

func TestRenderRejectsUnknownMessageType(t *testing.T) {
	r := newPair(t, func(f *fakeSidecar) {
		f.recv()
		f.send(map[string]any{"type": "hello"})
	})
	_, err := r.RenderHTML(context.Background(), "s", renderer.PageInfo{}, renderer.NopCallbacks{}, renderer.ModeArticle)
	if err == nil || !strings.Contains(err.Error(), "hello") {
		t.Errorf("err = %v, want substring %q", err, "hello")
	}
}

func TestRenderContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := newPair(t, func(f *fakeSidecar) { f.recv() })

	_, err := r.RenderHTML(ctx, "s", renderer.PageInfo{}, renderer.NopCallbacks{}, renderer.ModeArticle)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

type recordingCallbacks struct {
	renderer.NopCallbacks
	moduleName string
	userName   string
	refs       []string
	includes   []renderer.IncludeRef
}

func (c *recordingCallbacks) RenderModule(name string, params map[string]string, body string) (string, error) {
	c.moduleName = name
	return "<div>" + name + "|" + params["k"] + "|" + body + "</div>", nil
}

func (c *recordingCallbacks) RenderUser(user string, avatar bool) (string, error) {
	c.userName = user
	return "<span>" + user + "</span>", nil
}

func (c *recordingCallbacks) GetPageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	c.refs = refs
	title := "标题"
	return []renderer.PartialPageInfo{{FullName: refs[0], Exists: true, Title: &title}}, nil
}

func (c *recordingCallbacks) IncludePages(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	c.includes = refs
	body := "内容"
	return []renderer.FetchedPage{{FullName: refs[0].FullName, Content: &body}}, nil
}

func (c *recordingCallbacks) EvaluateExpression(string) (renderer.ExpressionResult, error) {
	return renderer.IntExpr(42), nil
}

func (c *recordingCallbacks) NextIncludeLevel() (bool, error) { return false, nil }

func TestDispatchRoundTrips(t *testing.T) {
	cb := &recordingCallbacks{}

	tests := []struct {
		method string
		args   map[string]any
		want   string
	}{
		{"module_has_body", map[string]any{"name": "Rate"}, `false`},
		{"render_module", map[string]any{"name": "Rate", "params": map[string]string{"k": "v"}, "body": "b"}, `"<div>Rate|v|b</div>"`},
		{"render_user", map[string]any{"user": "kakushi", "avatar": true}, `"<span>kakushi</span>"`},
		{"get_page_info", map[string]any{"refs": []string{"scp:173"}}, `[{"exists":true,"full_name":"scp:173","title":"标题"}]`},
		{"evaluate_expression", map[string]any{"expr": "1+1"}, `{"int":42,"kind":"int"}`},
		{"include_pages", map[string]any{"includes": []map[string]any{{"full_name": "a", "variables": map[string]string{}}}}, `[{"content":"内容","full_name":"a"}]`},
		{"next_include_level", map[string]any{}, `false`},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			raw, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("marshal err = %v, want nil", err)
			}
			got, err := dispatch(cb, tt.method, raw)
			if err != nil {
				t.Fatalf("dispatch(%s) err = %v, want nil", tt.method, err)
			}
			encoded, err := marshal(got)
			if err != nil {
				t.Fatalf("marshal err = %v, want nil", err)
			}
			if string(encoded) != tt.want {
				t.Errorf("dispatch(%s) = %s, want %s", tt.method, encoded, tt.want)
			}
		})
	}

	if cb.moduleName != "Rate" {
		t.Errorf("moduleName = %q, want Rate", cb.moduleName)
	}
	if len(cb.includes) != 1 || cb.includes[0].FullName != "a" {
		t.Errorf("includes = %+v, want [{a}]", cb.includes)
	}
}

func TestDispatchUnknownMethod(t *testing.T) {
	_, err := dispatch(renderer.NopCallbacks{}, "no_such_callback", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "no_such_callback") {
		t.Errorf("err = %v, want substring %q", err, "no_such_callback")
	}
}

func TestEncodeExpression(t *testing.T) {
	tests := []struct {
		in   renderer.ExpressionResult
		want string
	}{
		{renderer.ExpressionResult{}, `{"kind":"none"}`},
		{renderer.StringExpr("x"), `{"kind":"string","str":"x"}`},
		{renderer.BoolExpr(true), `{"bool":true,"kind":"bool"}`},
		{renderer.FloatExpr(1.5), `{"float":1.5,"kind":"float"}`},
		{renderer.IntExpr(-3), `{"int":-3,"kind":"int"}`},
	}
	for _, tt := range tests {
		got, err := marshal(encodeExpression(tt.in))
		if err != nil {
			t.Fatalf("marshal err = %v, want nil", err)
		}
		if string(got) != tt.want {
			t.Errorf("encodeExpression(%+v) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
