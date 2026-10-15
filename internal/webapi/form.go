package webapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const tailForm = "form"

type formOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type formField struct {
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Label     string       `json:"label"`
	Hint      string       `json:"hint"`
	Default   string       `json:"default"`
	Value     string       `json:"value"`
	Options   []formOption `json:"options"`
	Height    int          `json:"height"`
	MinLength int          `json:"minLength"`
	MaxLength int          `json:"maxLength"`
}

type formAnswer struct {
	Fields []formField       `json:"fields"`
	Values map[string]string `json:"values"`
	Broken bool              `json:"broken"`
}

func (h *Articles) definition(r *http.Request, name string) (*form.Definition, error) {
	category, pageName := wikidot.Split(strings.ToLower(name))
	if pageName == templateName {
		return nil, nil
	}
	ctx := r.Context()
	template, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), category+":"+templateName)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	source, err := h.deps.DB.LatestSource(ctx, template.ID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	def, found, err := form.Parse(source)
	if err != nil || !found || len(def.Fields) == 0 {
		return nil, nil
	}
	return def, nil
}

func (h *Articles) formFields(r *http.Request, _ *i18n.Localizer, name string) (string, int, error) {
	def, err := h.definition(r, name)
	if err != nil {
		return "", 0, err
	}
	answer := formAnswer{Values: map[string]string{}}
	if def == nil {
		return marshalAnswer(answer)
	}
	for _, field := range def.Fields {
		out := formField{
			Name: field.Name, Type: field.Type, Label: field.Label, Hint: field.Hint,
			Default: field.Default, Value: field.Value, Options: []formOption{},
			Height: field.Height, MinLength: field.MinLength, MaxLength: field.MaxLength,
		}
		for _, option := range field.Options {
			out.Options = append(out.Options, formOption{Key: option.Key, Label: option.Label})
		}
		answer.Fields = append(answer.Fields, out)
	}

	article, err := h.article(r, name)
	if errors.Is(err, errNotFound) {
		return marshalAnswer(answer)
	}
	if err != nil {
		return "", 0, err
	}
	source, err := h.deps.DB.LatestSource(r.Context(), article.ID)
	if errors.Is(err, db.ErrNotFound) {
		return marshalAnswer(answer)
	}
	if err != nil {
		return "", 0, err
	}
	values, err := form.ParseData(source)
	if err != nil || (len(values) == 0 && strings.TrimSpace(source) != "") {
		answer.Broken = true
		return marshalAnswer(answer)
	}
	answer.Values = values
	return marshalAnswer(answer)
}

func (h *Articles) formSource(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	var input struct {
		Values map[string]string `json:"values"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, bodyLimit)).Decode(&input); err != nil {
		return field("error", loc.T("api-bad-request")), http.StatusBadRequest, nil
	}
	def, err := h.definition(r, name)
	if err != nil {
		return "", 0, err
	}
	if def == nil {
		return field("error", loc.T("api-bad-request")), http.StatusBadRequest, nil
	}
	for _, f := range def.Fields {
		if f.Type != form.TypeText && f.Type != form.TypeWiki {
			continue
		}
		label := f.Label
		if label == "" {
			label = f.Name
		}
		count := utf8.RuneCountInString(strings.TrimSpace(input.Values[f.Name]))
		if f.MinLength > 0 && count < f.MinLength {
			return field("error", loc.T("form-field-too-short", "label", label, "count", f.MinLength)), http.StatusBadRequest, nil
		}
		if f.MaxLength > 0 && count > f.MaxLength {
			return field("error", loc.T("form-field-too-long", "label", label, "count", f.MaxLength)), http.StatusBadRequest, nil
		}
	}
	return field("source", def.Serialize(input.Values)), http.StatusOK, nil
}

func marshalAnswer(answer formAnswer) (string, int, error) {
	if answer.Fields == nil {
		answer.Fields = []formField{}
	}
	out, err := json.Marshal(answer)
	return string(out), http.StatusOK, err
}
