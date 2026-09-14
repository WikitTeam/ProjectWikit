package modules

import (
	"errors"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/module"
)

const darkModeBody = `# dark-mode
 * title: 暗色模式
 * type: checkbox`

func renderedInput(t *testing.T, body string) string {
	t.Helper()
	out, err := module.Render(module.Env{}, "input", nil, body)
	if err != nil {
		t.Fatalf("Render() err = %v, want nil", err)
	}
	return out
}

func TestInputCheckbox(t *testing.T) {
	got := renderedInput(t, darkModeBody)

	want := `<div class="mailform-box input-box">
<table class="form">
<tr><td>暗色模式</td><td><div class="field-error-message"></div><input class="checkbox" type="checkbox" name="dark-mode"></td></tr>
</table>
</div>`
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestInputCheckboxDefault(t *testing.T) {
	got := renderedInput(t, "# on\n * type: checkbox\n * default: 1")
	if !strings.Contains(got, `type="checkbox" name="on" checked>`) {
		t.Errorf("Render() = %q, want a checked box", got)
	}
}

func TestInputFallsBackToText(t *testing.T) {
	got := renderedInput(t, "# who\n * title: Name")
	if !strings.Contains(got, `<input class="text" type="text" name="who" size="30" value="">`) {
		t.Errorf("Render() = %q, want a text field", got)
	}
}

func TestInputTextSizeAndDefault(t *testing.T) {
	got := renderedInput(t, "# who\n * size: 8\n * default: kakushi")
	if !strings.Contains(got, `size="8" value="kakushi"`) {
		t.Errorf("Render() = %q, want size 8 and a default", got)
	}
}

func TestInputTextarea(t *testing.T) {
	got := renderedInput(t, "# note\n * type: textarea\n * default: hello")
	if !strings.Contains(got, `<textarea class="textarea" name="note">hello</textarea>`) {
		t.Errorf("Render() = %q, want a textarea", got)
	}
}

func TestInputSelect(t *testing.T) {
	got := renderedInput(t, "# pick\n * type: select\n * options: a: One\n * options: b: Two\n * default: b")
	want := `<select name="pick"><option value="a">One</option><option value="b" selected>Two</option></select>`
	if !strings.Contains(got, want) {
		t.Errorf("Render() = %q, want substring %q", got, want)
	}
}

func TestInputLabelFallsBackToTheName(t *testing.T) {
	got := renderedInput(t, "# dark-mode\n * type: checkbox")
	if !strings.Contains(got, `<td>dark-mode</td>`) {
		t.Errorf("Render() = %q, want the name as the label", got)
	}
}

func TestInputHint(t *testing.T) {
	got := renderedInput(t, "# who\n * hint: your name")
	if !strings.Contains(got, `<div class="sub">your name</div>`) {
		t.Errorf("Render() = %q, want a hint", got)
	}
}

func TestInputSeveralFields(t *testing.T) {
	got := renderedInput(t, "# one\n * type: checkbox\n# two\n * type: checkbox")
	if n := strings.Count(got, "<tr>"); n != 2 {
		t.Errorf("Render() rows = %d, want 2", n)
	}
	if n := strings.Count(got, `class="mailform-box input-box"`); n != 1 {
		t.Errorf("Render() boxes = %d, want 1", n)
	}
}

func TestInputEscapesTheAuthorsText(t *testing.T) {
	got := renderedInput(t, `# a"b
 * title: <b>x</b>`)
	if !strings.Contains(got, `name="a&quot;b"`) {
		t.Errorf("Render() = %q, want the name escaped", got)
	}
	if !strings.Contains(got, `<td>&lt;b&gt;x&lt;/b&gt;</td>`) {
		t.Errorf("Render() = %q, want the label escaped", got)
	}
}

func TestInputWithoutFields(t *testing.T) {
	_, err := module.Render(module.Env{}, "input", nil, "nothing here")

	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) {
		t.Fatalf("Render() err = %v, want a module error", err)
	}
	if moduleErr.Message != "module-input-no-fields" {
		t.Errorf("Render() message = %q, want %q", moduleErr.Message, "module-input-no-fields")
	}
}
