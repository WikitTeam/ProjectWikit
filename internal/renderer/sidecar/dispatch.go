package sidecar

import (
	"encoding/json"
	"fmt"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

func dispatch(cb renderer.Callbacks, method string, raw json.RawMessage) (any, error) {
	decode := func(v any) error {
		if err := json.Unmarshal(raw, v); err != nil {
			return fmt.Errorf("decode arguments for callback %s: %w", method, err)
		}
		return nil
	}
	fail := func(err error) (any, error) {
		return nil, fmt.Errorf("callback %s: %w", method, err)
	}

	switch method {
	case "module_has_body":
		var a struct {
			Name string `json:"name"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.ModuleHasBody(a.Name)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "render_module":
		var a struct {
			Name   string            `json:"name"`
			Params map[string]string `json:"params"`
			Body   string            `json:"body"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.RenderModule(a.Name, a.Params, a.Body)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "render_user":
		var a struct {
			User   string `json:"user"`
			Avatar bool   `json:"avatar"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.RenderUser(a.User, a.Avatar)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "get_i18n_message":
		var a struct {
			ID string `json:"id"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.GetI18nMessage(a.ID)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "get_html_injected_code":
		var a struct {
			ID string `json:"id"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.GetHTMLInjectedCode(a.ID)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "get_page_info":
		var a struct {
			Refs []string `json:"refs"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		infos, err := cb.GetPageInfo(a.Refs)
		if err != nil {
			return fail(err)
		}
		out := make([]map[string]any, 0, len(infos))
		for _, in := range infos {
			out = append(out, map[string]any{
				"full_name": in.FullName,
				"exists":    in.Exists,
				"title":     in.Title,
			})
		}
		return out, nil

	case "evaluate_expression":
		var a struct {
			Expr string `json:"expr"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.EvaluateExpression(a.Expr)
		if err != nil {
			return fail(err)
		}
		return encodeExpression(v), nil

	case "normalize_page_name":
		var a struct {
			FullName string `json:"full_name"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.NormalizePageName(a.FullName)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "include_pages":
		var a struct {
			Includes []struct {
				FullName  string            `json:"full_name"`
				Variables map[string]string `json:"variables"`
			} `json:"includes"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		refs := make([]renderer.IncludeRef, 0, len(a.Includes))
		for _, in := range a.Includes {
			refs = append(refs, renderer.IncludeRef{FullName: in.FullName, Variables: in.Variables})
		}
		pages, err := cb.IncludePages(refs)
		if err != nil {
			return fail(err)
		}
		out := make([]map[string]any, 0, len(pages))
		for _, p := range pages {
			out = append(out, map[string]any{"full_name": p.FullName, "content": p.Content})
		}
		return out, nil

	case "no_such_include":
		var a struct {
			FullName string `json:"full_name"`
		}
		if err := decode(&a); err != nil {
			return nil, err
		}
		v, err := cb.NoSuchInclude(a.FullName)
		if err != nil {
			return fail(err)
		}
		return v, nil

	case "next_include_level":
		v, err := cb.NextIncludeLevel()
		if err != nil {
			return fail(err)
		}
		return v, nil
	}

	return nil, fmt.Errorf("sidecar asked for unknown callback %q", method)
}

func encodeExpression(v renderer.ExpressionResult) map[string]any {
	switch v.Kind {
	case renderer.ExprString:
		return map[string]any{"kind": "string", "str": v.Str}
	case renderer.ExprBool:
		return map[string]any{"kind": "bool", "bool": v.Bool}
	case renderer.ExprFloat:
		return map[string]any{"kind": "float", "float": v.Float}
	case renderer.ExprInt:
		return map[string]any{"kind": "int", "int": v.Int}
	default:
		return map[string]any{"kind": "none"}
	}
}
