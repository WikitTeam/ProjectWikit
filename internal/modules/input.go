package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("input", renderInput) }

const (
	inputTypeText     = "text"
	inputTypeTextarea = "textarea"
	inputTypeSelect   = "select"
	inputTypeCheckbox = "checkbox"

	inputDefaultSize = 30
)

type inputField struct {
	name    string
	title   string
	kind    string
	size    string
	value   string
	hint    string
	options []inputOption
}

type inputOption struct {
	Key   string
	Label string
}

func renderInput(env module.Env, _ map[string]string, body string) (string, error) {
	fields := parseInputFields(body)
	if len(fields) == 0 {
		return "", &module.Error{Message: env.Text("module-input-no-fields")}
	}

	var b strings.Builder
	// The first name is what a theme written against wikidot selects on, and the
	// second is the one to write against once nothing does.
	b.WriteString(`<div class="mailform-box input-box">` + "\n")
	b.WriteString(`<table class="form">` + "\n")
	for i := range fields {
		b.WriteString(inputRow(&fields[i]))
	}
	b.WriteString("</table>\n")
	b.WriteString("</div>")
	return b.String(), nil
}

// The definitions are written as one wikitext list, which is why a field opens
// on a # line and its properties are bullets under it.
func parseInputFields(body string) []inputField {
	var fields []inputField
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "#"):
			name := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if name != "" {
				fields = append(fields, inputField{name: name, kind: inputTypeText})
			}
		case strings.HasPrefix(line, "*") && len(fields) > 0:
			key, value, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(line, "*")), ":")
			if ok {
				readInputProperty(&fields[len(fields)-1], strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(value))
			}
		}
	}
	return fields
}

func readInputProperty(field *inputField, key, value string) {
	switch key {
	case "title":
		field.title = value
	case "type":
		field.kind = strings.ToLower(value)
	case "size":
		field.size = value
	case "default":
		field.value = value
	case "hint":
		field.hint = value
	case "options":
		field.options = append(field.options, parseInputOption(value))
	}
}

func parseInputOption(value string) inputOption {
	key, label, ok := strings.Cut(value, ":")
	if !ok {
		return inputOption{Key: value, Label: value}
	}
	return inputOption{Key: strings.TrimSpace(key), Label: strings.TrimSpace(label)}
}

func inputRow(field *inputField) string {
	label := field.title
	if label == "" {
		label = field.name
	}
	hint := ""
	if field.hint != "" {
		hint = `<div class="sub">` + escape.HTML(field.hint) + `</div>`
	}
	// The empty error slot is styled away until a row is marked invalid, and it
	// is kept so a theme written against wikidot still finds it.
	return `<tr><td>` + escape.HTML(label) + `</td>` +
		`<td><div class="field-error-message"></div>` + inputControl(field) + hint + `</td></tr>` + "\n"
}

func inputControl(field *inputField) string {
	name := ` name="` + escape.HTML(field.name) + `"`
	switch field.kind {
	case inputTypeCheckbox:
		checked := ""
		if inputChecked(field.value) {
			checked = ` checked`
		}
		return `<input class="checkbox" type="checkbox"` + name + checked + `>`
	case inputTypeTextarea:
		return `<textarea class="textarea"` + name + `>` + escape.HTML(field.value) + `</textarea>`
	case inputTypeSelect:
		var b strings.Builder
		b.WriteString(`<select` + name + `>`)
		for _, option := range field.options {
			selected := ""
			if option.Key == field.value {
				selected = ` selected`
			}
			b.WriteString(`<option value="` + escape.HTML(option.Key) + `"` + selected + `>` +
				escape.HTML(option.Label) + `</option>`)
		}
		b.WriteString(`</select>`)
		return b.String()
	}
	return `<input class="text" type="text"` + name + ` size="` + strconv.Itoa(inputSize(field.size)) +
		`" value="` + escape.HTML(field.value) + `">`
}

func inputChecked(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "yes", "true", "checked":
		return true
	}
	return false
}

func inputSize(raw string) int {
	size, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || size <= 0 {
		return inputDefaultSize
	}
	return size
}
