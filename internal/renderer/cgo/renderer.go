//go:build cgo && !nocgo

package cgo

/*
#include <stdlib.h>
#include "ftml.h"
*/
import "C"

import (
	"context"
	"errors"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

var ErrRenderFailed = errors.New("cgo: ftml render failed")

type Renderer struct{}

var _ renderer.Renderer = (*Renderer)(nil)

func New() *Renderer {
	return &Renderer{}
}

func Version() string {
	return goString(C.ftml_version())
}

// Lives entirely in C memory: it is passed by pointer, and cgo forbids memory
// C can reach from holding Go pointers.
type pageInfo struct {
	value *C.FtmlPageInfo
	tags  []C.FtmlStr
}

func newPageInfo(info renderer.PageInfo) *pageInfo {
	value := (*C.FtmlPageInfo)(C.calloc(1, C.sizeof_FtmlPageInfo))
	value.page = cCopy(info.Page)
	value.category = cCopy(info.Category)
	value.site = cCopy(info.Site)
	value.title = cCopy(info.Title)
	value.domain = cCopy(info.Domain)
	value.media_domain = cCopy(info.MediaDomain)
	value.language = cCopy(info.Language)
	value.rating = C.double(info.Rating)

	held := make([]C.FtmlStr, 0, len(info.Tags))
	if len(info.Tags) > 0 {
		array := (*C.FtmlStr)(C.calloc(C.size_t(len(info.Tags)), C.sizeof_FtmlStr))
		slice := unsafe.Slice(array, len(info.Tags))
		for i, tag := range info.Tags {
			slice[i] = cCopy(tag)
			held = append(held, slice[i])
		}
		value.tags = array
		value.tag_count = C.size_t(len(info.Tags))
	}
	return &pageInfo{value: value, tags: held}
}

func (p *pageInfo) free() {
	for _, tag := range p.tags {
		cFree(tag)
	}
	if p.value.tags != nil {
		C.free(unsafe.Pointer(p.value.tags))
	}
	cFree(p.value.page)
	cFree(p.value.category)
	cFree(p.value.site)
	cFree(p.value.title)
	cFree(p.value.domain)
	cFree(p.value.media_domain)
	cFree(p.value.language)
	C.free(unsafe.Pointer(p.value))
}

type entryPoint func(C.FtmlStr, *C.FtmlCallbacks, C.uintptr_t, *C.FtmlPageInfo, C.FtmlStr) *C.FtmlResult

func (r *Renderer) call(entry entryPoint, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (*C.FtmlResult, func(), error) {
	var callbacks C.FtmlCallbacks
	C.pwikit_fill_callbacks(&callbacks)

	handle := cgo.NewHandle(cb)
	defer handle.Delete()

	page := newPageInfo(info)
	defer page.free()

	modeText := string(mode)
	result := entry(borrow(source), &callbacks, C.uintptr_t(handle), page.value, borrow(modeText))
	runtime.KeepAlive(source)
	runtime.KeepAlive(modeText)

	if result == nil {
		return nil, nil, ErrRenderFailed
	}
	return result, func() { C.ftml_result_free(result) }, nil
}

func collectStrings(count func() C.size_t, at func(C.size_t) C.FtmlStr) []string {
	total := int(count())
	if total == 0 {
		return nil
	}
	out := make([]string, 0, total)
	for i := 0; i < total; i++ {
		out = append(out, goString(at(C.size_t(i))))
	}
	return out
}

func buildResult(result *C.FtmlResult) renderer.Result {
	return renderer.Result{
		Body: goString(C.ftml_result_body(result)),
		IncludedPages: collectStrings(
			func() C.size_t { return C.ftml_result_included_len(result) },
			func(i C.size_t) C.FtmlStr { return C.ftml_result_included_at(result, i) }),
		LinkedPages: collectStrings(
			func() C.size_t { return C.ftml_result_linked_len(result) },
			func(i C.size_t) C.FtmlStr { return C.ftml_result_linked_at(result, i) }),
		Code: buildCode(result),
		HTML: collectStrings(
			func() C.size_t { return C.ftml_result_html_len(result) },
			func(i C.size_t) C.FtmlStr { return C.ftml_result_html_at(result, i) }),
	}
}

func buildCode(result *C.FtmlResult) []renderer.CodeBlock {
	total := int(C.ftml_result_code_len(result))
	if total == 0 {
		return nil
	}
	out := make([]renderer.CodeBlock, 0, total)
	for i := 0; i < total; i++ {
		out = append(out, renderer.CodeBlock{
			Language: goString(C.ftml_result_code_language_at(result, C.size_t(i))),
			Source:   goString(C.ftml_result_code_contents_at(result, C.size_t(i))),
		})
	}
	return out
}

func (r *Renderer) render(entry entryPoint, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	result, free, err := r.call(entry, source, info, cb, mode)
	if err != nil {
		return renderer.Result{}, err
	}
	defer free()
	return buildResult(result), nil
}

func (r *Renderer) RenderHTML(_ context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return r.render(entryRenderHTML, source, info, cb, mode)
}

func (r *Renderer) RenderText(_ context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return r.render(entryRenderText, source, info, cb, mode)
}

func (r *Renderer) CollectBacklinks(_ context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return r.render(entryCollectBacklinks, source, info, cb, mode)
}

func (r *Renderer) CollectCodeAndHTML(_ context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Parts, error) {
	result, free, err := r.call(entryCollectCodeAndHTML, source, info, cb, mode)
	if err != nil {
		return renderer.Parts{}, err
	}
	defer free()
	return renderer.Parts{
		Code: buildCode(result),
		HTML: collectStrings(
			func() C.size_t { return C.ftml_result_html_len(result) },
			func(i C.size_t) C.FtmlStr { return C.ftml_result_html_at(result, i) }),
	}, nil
}

func entryRenderHTML(source C.FtmlStr, cb *C.FtmlCallbacks, host C.uintptr_t, info *C.FtmlPageInfo, mode C.FtmlStr) *C.FtmlResult {
	return C.ftml_render_html(source, cb, host, info, mode)
}

func entryRenderText(source C.FtmlStr, cb *C.FtmlCallbacks, host C.uintptr_t, info *C.FtmlPageInfo, mode C.FtmlStr) *C.FtmlResult {
	return C.ftml_render_text(source, cb, host, info, mode)
}

func entryCollectBacklinks(source C.FtmlStr, cb *C.FtmlCallbacks, host C.uintptr_t, info *C.FtmlPageInfo, mode C.FtmlStr) *C.FtmlResult {
	return C.ftml_collect_backlinks(source, cb, host, info, mode)
}

func entryCollectCodeAndHTML(source C.FtmlStr, cb *C.FtmlCallbacks, host C.uintptr_t, info *C.FtmlPageInfo, mode C.FtmlStr) *C.FtmlResult {
	return C.ftml_collect_code_and_html(source, cb, host, info, mode)
}
