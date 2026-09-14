// Package form answers what a data form category stores and how one stored
// value reaches the page.
package form

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	TypeText     = "text"
	TypeWiki     = "wiki"
	TypeSelect   = "select"
	TypeCheckbox = "checkbox"
	TypeStatic   = "static"
	TypeHidden   = "hidden"
)

type Option struct {
	Key   string
	Label string
}

type Field struct {
	Name    string
	Type    string
	Label   string
	Hint    string
	Default string
	Value   string
	Options []Option
}

type Definition struct {
	Fields []Field
}

var blockPattern = regexp.MustCompile(`(?is)\[\[\s*form\s*]](.*?)\[\[\s*/\s*form\s*]]`)

// A category template carries the definition for this package, not for the
// reader.
func Strip(source string) string {
	return blockPattern.ReplaceAllString(source, "")
}

func Parse(source string) (*Definition, bool, error) {
	match := blockPattern.FindStringSubmatch(source)
	if match == nil {
		return nil, false, nil
	}
	def, err := parseDefinition(match[1])
	if err != nil {
		return nil, true, err
	}
	return def, true, nil
}

func parseDefinition(body string) (*Definition, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(body), &root); err != nil {
		return nil, fmt.Errorf("parse form definition: %w", err)
	}
	mapping := documentMapping(&root)
	if mapping == nil {
		return &Definition{}, nil
	}
	fields := valueFor(mapping, "fields")
	if fields == nil || fields.Kind != yaml.MappingNode {
		return &Definition{}, nil
	}

	out := &Definition{}
	for i := 0; i+1 < len(fields.Content); i += 2 {
		field := Field{Name: fields.Content[i].Value, Type: TypeText}
		if body := fields.Content[i+1]; body.Kind == yaml.MappingNode {
			readField(&field, body)
		}
		out.Fields = append(out.Fields, field)
	}
	return out, nil
}

func readField(field *Field, body *yaml.Node) {
	for i := 0; i+1 < len(body.Content); i += 2 {
		key := strings.ToLower(strings.TrimSpace(body.Content[i].Value))
		value := body.Content[i+1]
		switch key {
		case "type":
			field.Type = strings.ToLower(strings.TrimSpace(value.Value))
		case "label":
			field.Label = value.Value
		case "hint":
			field.Hint = value.Value
		case "default":
			field.Default = value.Value
		case "value":
			field.Value = value.Value
		case "values":
			field.Options = readOptions(value)
		}
	}
}

// Every option is read as the scalar's text, which keeps a key of 08 out of
// octal and one of Yes out of boolean.
func readOptions(node *yaml.Node) []Option {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	out := make([]Option, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		out = append(out, Option{Key: node.Content[i].Value, Label: node.Content[i+1].Value})
	}
	return out
}

func ParseData(source string) (map[string]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(source), &root); err != nil {
		return nil, fmt.Errorf("parse form data: %w", err)
	}
	mapping := documentMapping(&root)
	if mapping == nil {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(mapping.Content)/2)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if value := mapping.Content[i+1]; value.Kind == yaml.ScalarNode {
			out[mapping.Content[i].Value] = value.Value
		}
	}
	return out, nil
}

func documentMapping(root *yaml.Node) *yaml.Node {
	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	return node
}

func valueFor(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if strings.EqualFold(strings.TrimSpace(mapping.Content[i].Value), key) {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// Field matches without regard to case because a module argument reaches the
// query lowercased while the definition keeps whatever the author typed.
func (d *Definition) Field(name string) (*Field, bool) {
	if d == nil {
		return nil, false
	}
	for i := range d.Fields {
		if strings.EqualFold(d.Fields[i].Name, name) {
			return &d.Fields[i], true
		}
	}
	return nil, false
}

func (d *Definition) Raw(values map[string]string, name string) (string, bool) {
	field, ok := d.Field(name)
	if !ok {
		return "", false
	}
	if field.Type == TypeStatic || field.Type == TypeHidden {
		return field.Value, true
	}
	if stored, ok := values[field.Name]; ok {
		return stored, true
	}
	return field.Default, true
}

func (d *Definition) Data(values map[string]string, name string) (string, bool) {
	field, ok := d.Field(name)
	if !ok {
		return "", false
	}
	raw, _ := d.Raw(values, name)
	switch field.Type {
	case TypeWiki, TypeStatic:
		return raw, true
	case TypeSelect:
		for _, option := range field.Options {
			if option.Key == raw {
				// The label comes from the template rather than from whoever
				// filled the page in, so it keeps its markup.
				return option.Label, true
			}
		}
		return EscapeMarkup(raw), true
	}
	return EscapeMarkup(raw), true
}

func (d *Definition) Label(name string) (string, bool) {
	field, ok := d.Field(name)
	if !ok {
		return "", false
	}
	return field.Label, true
}

func (d *Definition) Hint(name string) (string, bool) {
	field, ok := d.Field(name)
	if !ok {
		return "", false
	}
	return field.Hint, true
}

// A value is substituted into the source before the renderer sees it, so a
// field that promises no wiki syntax has to carry its own escape.
func EscapeMarkup(value string) string {
	if value == "" {
		return ""
	}
	// A closing @@ inside the value would end the span early. Reopening around
	// a literal pair puts it back as text.
	return "@@" + strings.ReplaceAll(value, "@@", strings.Repeat("@", 10)) + "@@"
}
