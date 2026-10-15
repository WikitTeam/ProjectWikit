// Package form answers what a data form category stores and how one stored
// value reaches the page.
package form

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
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
	TypePassword = "password"
)

type Option struct {
	Key   string
	Label string
}

type Field struct {
	Name      string
	Type      string
	Label     string
	Hint      string
	Default   string
	Value     string
	Options   []Option
	Height    int
	MinLength int
	MaxLength int
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
	body = strings.ReplaceAll(body, "\u00a0", " ")
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
		case "height":
			field.Height, _ = strconv.Atoi(strings.TrimSpace(value.Value))
		case "minlength":
			field.MinLength, _ = strconv.Atoi(strings.TrimSpace(value.Value))
		case "maxlength":
			field.MaxLength, _ = strconv.Atoi(strings.TrimSpace(value.Value))
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
	if err := yaml.Unmarshal([]byte(plainIndent(source)), &root); err != nil {
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

func plainIndent(source string) string {
	if !strings.Contains(source, "\u00a0") {
		return source
	}
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		rest := strings.TrimLeft(line, " \t\u00a0")
		indent := line[:len(line)-len(rest)]
		lines[i] = strings.ReplaceAll(indent, "\u00a0", " ") + rest
	}
	return strings.Join(lines, "\n")
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
	case TypeCheckbox:
		if Checked(raw) {
			return "1", true
		}
		return "0", true
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

func Checked(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (d *Definition) Serialize(values map[string]string) string {
	var b strings.Builder
	written := map[string]bool{}
	for _, field := range d.Fields {
		written[field.Name] = true
		value, ok := values[field.Name]
		switch field.Type {
		case TypeStatic:
			continue
		case TypeHidden:
			value, ok = field.Value, true
		case TypeCheckbox:
			value, ok = "0", true
			if Checked(values[field.Name]) {
				value = "1"
			}
		}
		if !ok {
			value = field.Default
		}
		writeEntry(&b, field.Name, value, field.Type == TypeSelect || field.Type == TypeCheckbox)
	}
	extra := make([]string, 0)
	for name := range values {
		if !written[name] {
			extra = append(extra, name)
		}
	}
	slices.Sort(extra)
	for _, name := range extra {
		writeEntry(&b, name, values[name], false)
	}
	return b.String()
}

func writeEntry(b *strings.Builder, name, value string, quoted bool) {
	b.WriteString(name + ": ")
	switch {
	case quoted && !strings.ContainsAny(value, "\n\r"):
		b.WriteString("'" + strings.ReplaceAll(value, "'", "''") + "'")
	case plainScalar(value):
		b.WriteString(value)
	default:
		b.WriteString(doubleQuoted(value))
	}
	b.WriteString("\n")
}

func plainScalar(value string) bool {
	if value == "" || strings.ContainsAny(value, "\n\r\t\u00a0") || strings.TrimSpace(value) != value {
		return false
	}
	var node yaml.Node
	if yaml.Unmarshal([]byte("v: "+value), &node) != nil {
		return false
	}
	mapping := documentMapping(&node)
	if mapping == nil || len(mapping.Content) != 2 {
		return false
	}
	got := mapping.Content[1]
	return got.Kind == yaml.ScalarNode && got.Tag == "!!str" && got.Style == 0 && got.Value == value
}

func doubleQuoted(value string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
	return strings.TrimSuffix(buf.String(), "\n")
}
