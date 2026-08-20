package renderer

import (
	"context"
	"fmt"
)

type FakeCall struct {
	Op     string
	Source string
	Info   PageInfo
	Mode   Mode
}

type Fake struct {
	Result Result
	Parts  Parts
	Err    error
	Calls  []FakeCall
}

var _ Renderer = (*Fake)(nil)

func (f *Fake) RenderHTML(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Result, error) {
	return f.record("RenderHTML", ctx, source, info, mode)
}

func (f *Fake) RenderText(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Result, error) {
	return f.record("RenderText", ctx, source, info, mode)
}

func (f *Fake) CollectBacklinks(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Result, error) {
	return f.record("CollectBacklinks", ctx, source, info, mode)
}

func (f *Fake) CollectCodeAndHTML(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Parts, error) {
	if _, err := f.record("CollectCodeAndHTML", ctx, source, info, mode); err != nil {
		return Parts{}, err
	}
	return f.Parts, nil
}

func (f *Fake) record(op string, ctx context.Context, source string, info PageInfo, mode Mode) (Result, error) {
	f.Calls = append(f.Calls, FakeCall{Op: op, Source: source, Info: info, Mode: mode})
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if f.Err != nil {
		return Result{}, f.Err
	}
	if !mode.Valid() {
		return Result{}, fmt.Errorf("unknown render mode %q", mode)
	}
	return f.Result, nil
}

type NopCallbacks struct{}

var _ Callbacks = NopCallbacks{}

func (NopCallbacks) ModuleHasBody(string) (bool, error) { return false, nil }

func (NopCallbacks) RenderModule(string, map[string]string, string) (string, error) {
	return "", nil
}

func (NopCallbacks) RenderUser(string, bool) (string, error)         { return "", nil }
func (NopCallbacks) GetI18nMessage(string) (string, error)           { return "", nil }
func (NopCallbacks) GetHTMLInjectedCode(string) (string, error)      { return "", nil }
func (NopCallbacks) GetPageInfo([]string) ([]PartialPageInfo, error) { return nil, nil }

func (NopCallbacks) EvaluateExpression(string) (ExpressionResult, error) {
	return ExpressionResult{}, nil
}

func (NopCallbacks) NormalizePageName(name string) (string, error) { return name, nil }

func (NopCallbacks) IncludePages([]IncludeRef) ([]FetchedPage, error) { return nil, nil }
func (NopCallbacks) NoSuchInclude(string) (string, error)             { return "", nil }
func (NopCallbacks) NextIncludeLevel() (bool, error)                  { return true, nil }
