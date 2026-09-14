package htmlsource

import (
	"os"
	"strings"
	"testing"
)

func split(body string) []struct{ Name, Text string } {
	var out []struct{ Name, Text string }
	name := ""
	var lines []string
	flush := func() {
		if name == "" {
			return
		}
		out = append(out, struct{ Name, Text string }{name, strings.Join(lines, "\n")})
	}
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			flush()
			name = strings.TrimSuffix(strings.TrimPrefix(line, "=== "), " ===")
			lines = nil
			continue
		}
		if name != "" {
			lines = append(lines, line)
		}
	}
	flush()
	return out
}

func TestConvertMatchesGolden(t *testing.T) {
	cases, err := os.ReadFile("testdata/cases.html")
	if err != nil {
		t.Fatalf("ReadFile(cases) err = %v, want nil", err)
	}
	golden, err := os.ReadFile("testdata/html.golden")
	if err != nil {
		t.Fatalf("ReadFile(golden) err = %v, want nil", err)
	}

	want := map[string]string{}
	for _, one := range split(string(golden)) {
		want[one.Name] = one.Text
	}
	for _, one := range split(string(cases)) {
		expected, ok := want[one.Name]
		if !ok {
			t.Errorf("golden has no case %q, want one", one.Name)
			continue
		}
		got := Convert(strings.Trim(one.Text, "\n"))
		if strings.TrimRight(got, "\n") != strings.TrimRight(expected, "\n") {
			t.Errorf("Convert(%s) = %q, want %q", one.Name, got, expected)
		}
	}
}

func TestGoldenCoversEveryCase(t *testing.T) {
	cases, err := os.ReadFile("testdata/cases.html")
	if err != nil {
		t.Fatalf("ReadFile(cases) err = %v, want nil", err)
	}
	if got := len(split(string(cases))); got < 30 {
		t.Errorf("len(cases) = %d, want at least 30", got)
	}
}
