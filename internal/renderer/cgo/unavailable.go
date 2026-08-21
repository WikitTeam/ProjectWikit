//go:build !cgo || nocgo

package cgo

import (
	"context"
	"errors"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

var ErrRenderFailed = errors.New("cgo: built without ftml, use the sidecar renderer")

type Renderer struct{}

var _ renderer.Renderer = (*Renderer)(nil)

func New() *Renderer {
	return &Renderer{}
}

func Version() string {
	return ""
}

func (r *Renderer) RenderHTML(context.Context, string, renderer.PageInfo, renderer.Callbacks, renderer.Mode) (renderer.Result, error) {
	return renderer.Result{}, ErrRenderFailed
}

func (r *Renderer) RenderText(context.Context, string, renderer.PageInfo, renderer.Callbacks, renderer.Mode) (renderer.Result, error) {
	return renderer.Result{}, ErrRenderFailed
}

func (r *Renderer) CollectBacklinks(context.Context, string, renderer.PageInfo, renderer.Callbacks, renderer.Mode) (renderer.Result, error) {
	return renderer.Result{}, ErrRenderFailed
}

func (r *Renderer) CollectCodeAndHTML(context.Context, string, renderer.PageInfo, renderer.Callbacks, renderer.Mode) (renderer.Parts, error) {
	return renderer.Parts{}, ErrRenderFailed
}
