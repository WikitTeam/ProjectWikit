//go:build cgo && !nocgo

package cgo

/*
#include <stdlib.h>
#include "ftml.h"
*/
import "C"

import (
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

// Callers owe the borrowed string a runtime.KeepAlive: nothing else holds it
// while C reads through the pointer.
func borrow(s string) C.FtmlStr {
	if len(s) == 0 {
		return C.FtmlStr{}
	}
	return C.FtmlStr{
		ptr: (*C.char)(unsafe.Pointer(unsafe.StringData(s))),
		len: C.size_t(len(s)),
	}
}

func goString(s C.FtmlStr) string {
	if s.ptr == nil || s.len == 0 {
		return ""
	}
	return C.GoStringN(s.ptr, C.int(s.len))
}

// Required wherever an FtmlStr is stored in a struct C holds a pointer to:
// cgo forbids C memory from containing Go pointers, so borrowing is not an
// option there.
func cCopy(s string) C.FtmlStr {
	if len(s) == 0 {
		return C.FtmlStr{}
	}
	buf := C.CBytes([]byte(s))
	return C.FtmlStr{ptr: (*C.char)(buf), len: C.size_t(len(s))}
}

func cFree(s C.FtmlStr) {
	if s.ptr != nil {
		C.free(unsafe.Pointer(s.ptr))
	}
}

func host(h C.uintptr_t) renderer.Callbacks {
	return cgo.Handle(h).Value().(renderer.Callbacks)
}

//export pwikitModuleHasBody
func pwikitModuleHasBody(h C.uintptr_t, name C.FtmlStr) C.int {
	ok, err := host(h).ModuleHasBody(goString(name))
	if err != nil || !ok {
		return 0
	}
	return 1
}

//export pwikitRenderModule
func pwikitRenderModule(h C.uintptr_t, name C.FtmlStr, params *C.FtmlKeyValue, count C.size_t, body C.FtmlStr, sink *C.FtmlStringSink) {
	values := make(map[string]string, int(count))
	if params != nil && count > 0 {
		for _, kv := range unsafe.Slice(params, int(count)) {
			values[goString(kv.key)] = goString(kv.value)
		}
	}
	out, err := host(h).RenderModule(goString(name), values, goString(body))
	if err != nil {
		return
	}
	C.ftml_sink_string(sink, borrow(out))
	runtime.KeepAlive(out)
}

//export pwikitRenderUser
func pwikitRenderUser(h C.uintptr_t, user C.FtmlStr, avatar C.int, sink *C.FtmlStringSink) {
	out, err := host(h).RenderUser(goString(user), avatar != 0)
	if err != nil {
		return
	}
	C.ftml_sink_string(sink, borrow(out))
	runtime.KeepAlive(out)
}

//export pwikitGetI18nMessage
func pwikitGetI18nMessage(h C.uintptr_t, id C.FtmlStr, sink *C.FtmlStringSink) {
	out, err := host(h).GetI18nMessage(goString(id))
	if err != nil {
		return
	}
	C.ftml_sink_string(sink, borrow(out))
	runtime.KeepAlive(out)
}

//export pwikitGetHTMLInjectedCode
func pwikitGetHTMLInjectedCode(h C.uintptr_t, id C.FtmlStr, sink *C.FtmlStringSink) {
	out, err := host(h).GetHTMLInjectedCode(goString(id))
	if err != nil {
		return
	}
	C.ftml_sink_string(sink, borrow(out))
	runtime.KeepAlive(out)
}

//export pwikitGetPageInfo
func pwikitGetPageInfo(h C.uintptr_t, refs *C.FtmlStr, count C.size_t, sink *C.FtmlPageInfoSink) {
	names := make([]string, 0, int(count))
	if refs != nil && count > 0 {
		for _, ref := range unsafe.Slice(refs, int(count)) {
			names = append(names, goString(ref))
		}
	}
	infos, err := host(h).GetPageInfo(names)
	if err != nil {
		return
	}
	for _, info := range infos {
		title := ""
		hasTitle := C.int(0)
		if info.Title != nil {
			title = *info.Title
			hasTitle = 1
		}
		exists := C.int(0)
		if info.Exists {
			exists = 1
		}
		C.ftml_sink_page_info(sink, borrow(info.FullName), borrow(title), hasTitle, exists)
		runtime.KeepAlive(info.FullName)
		runtime.KeepAlive(title)
	}
}

//export pwikitEvaluateExpression
func pwikitEvaluateExpression(h C.uintptr_t, expr C.FtmlStr, out *C.FtmlExpressionResult, sink *C.FtmlStringSink) {
	result, err := host(h).EvaluateExpression(goString(expr))
	if err != nil {
		return
	}
	switch result.Kind {
	case renderer.ExprString:
		out.kind = C.FTML_EXPR_STRING
		C.ftml_sink_string(sink, borrow(result.Str))
		runtime.KeepAlive(result.Str)
	case renderer.ExprBool:
		out.kind = C.FTML_EXPR_BOOL
		if result.Bool {
			out.int_value = 1
		}
	case renderer.ExprFloat:
		out.kind = C.FTML_EXPR_FLOAT
		out.float_value = C.double(result.Float)
	case renderer.ExprInt:
		out.kind = C.FTML_EXPR_INT
		out.int_value = C.int64_t(result.Int)
	}
}

//export pwikitNormalizePageName
func pwikitNormalizePageName(h C.uintptr_t, fullName C.FtmlStr, sink *C.FtmlStringSink) {
	out, err := host(h).NormalizePageName(goString(fullName))
	if err != nil {
		return
	}
	C.ftml_sink_string(sink, borrow(out))
	runtime.KeepAlive(out)
}

//export pwikitIncludePages
func pwikitIncludePages(h C.uintptr_t, refs *C.FtmlIncludeRef, count C.size_t, sink *C.FtmlFetchedPageSink) {
	includes := make([]renderer.IncludeRef, 0, int(count))
	if refs != nil && count > 0 {
		for _, ref := range unsafe.Slice(refs, int(count)) {
			variables := make(map[string]string, int(ref.variable_count))
			if ref.variables != nil && ref.variable_count > 0 {
				for _, kv := range unsafe.Slice(ref.variables, int(ref.variable_count)) {
					variables[goString(kv.key)] = goString(kv.value)
				}
			}
			includes = append(includes, renderer.IncludeRef{
				FullName:  goString(ref.full_name),
				Variables: variables,
			})
		}
	}
	pages, err := host(h).IncludePages(includes)
	if err != nil {
		return
	}
	for _, page := range pages {
		content := ""
		hasContent := C.int(0)
		if page.Content != nil {
			content = *page.Content
			hasContent = 1
		}
		C.ftml_sink_fetched_page(sink, borrow(page.FullName), borrow(content), hasContent)
		runtime.KeepAlive(page.FullName)
		runtime.KeepAlive(content)
	}
}

//export pwikitNoSuchInclude
func pwikitNoSuchInclude(h C.uintptr_t, fullName C.FtmlStr, sink *C.FtmlStringSink) {
	out, err := host(h).NoSuchInclude(goString(fullName))
	if err != nil {
		return
	}
	C.ftml_sink_string(sink, borrow(out))
	runtime.KeepAlive(out)
}

//export pwikitNextIncludeLevel
func pwikitNextIncludeLevel(h C.uintptr_t) C.int {
	ok, err := host(h).NextIncludeLevel()
	if err != nil || !ok {
		return 0
	}
	return 1
}
