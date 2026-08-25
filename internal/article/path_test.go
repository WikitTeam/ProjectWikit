package article

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/page"
)

var update = flag.Bool("update", false, "rewrite the golden files and export the corpus")

var pathCorpus = []string{
	"",
	"/",
	"main",
	"main/",
	"main/a/1",
	"main/a/1/",
	"main/a/",
	"main/norender",
	"main/a/1/b",
	"main/A/1/B/2",
	"main/b/2/a/1",
	"main/a/1/a/2",
	"main//x",
	" main ",
	"MAIN/A/B",
	"main/a/%2Fb",
	"main/%20x/%20y",
	"main/uni/%E4%B8%AD",
	"main/bad/%FF%FF",
	"main/bad/%E4%B8",
	"main/bad/%ZZ",
	"scp-173/norender/true",
	"%D1%80%D1%83%D1%81/a/1",
	"forum/start",
	"forum/start/x/1",
	"forum/c-12/x/1",
	"forum/t-7",
	"forum/s-3/a/b",
	"forum/other",
}

var thisPageParams = Params{
	{Key: "a", Value: "1"},
	{Key: "norender", Bare: true},
	{Key: "q", Value: "x y/z"},
	{Key: "uni", Value: "中"},
}

var thisPageNames = []string{
	"path|a",
	"path|A",
	"path|missing",
	"path|norender",
	"path|",
	"path_expr|a",
	"path_expr|missing",
	"path_expr|norender",
	"path_expr|uni",
	"path_expr|q",
	"path_url|q",
	"path_url|missing",
	"path_url|norender",
	"path_url|uni",
	"canonical_url",
	"PATH|a",
	"other",
}

var redirectCorpus = []string{
	"main",
	"main/a/1",
	"Main",
	"Main/",
	"Main/a/1/",
	"Main/a/",
	"Main/b/2/a/1",
	"Main/A/1/B/2",
	"Main/norender/true/foo",
	"Main/x/y%20z",
	"Main//x",
	"Main/a/%2Fb",
	"Main/%20x/1",
	"%D1%80%D1%83%D1%81/a/1",
	"Some%20Page/a/1",
	"SCP-173",
}

const canonicalURL = "https://wiki.example/main/a/1"

func TestParsePathMatchesGolden(t *testing.T) {
	var b strings.Builder
	for _, raw := range pathCorpus {
		name, params := ParsePath(raw, "")
		fmt.Fprintf(&b, "=== %s\nname %q\n", raw, name)
		for _, param := range params {
			if param.Bare {
				fmt.Fprintf(&b, "param %q = <none>\n", param.Key)
				continue
			}
			fmt.Fprintf(&b, "param %q = %q\n", param.Key, param.Value)
		}
	}
	checkGolden(t, "path.golden", b.String())
	if *update {
		writeCorpus(t, "path_corpus.json", pathCorpus)
	}
}

func TestThisPageMatchesGolden(t *testing.T) {
	resolve := ThisPage(thisPageParams, canonicalURL)
	var b strings.Builder
	for _, name := range thisPageNames {
		fmt.Fprintf(&b, "%s -> %q\n", name, page.ApplyTemplate("%%"+name+"%%", resolve))
	}
	checkGolden(t, "thispage.golden", b.String())
	if *update {
		corpus := map[string]any{"params": paramsForCorpus(thisPageParams), "names": thisPageNames, "canonical_url": canonicalURL}
		writeCorpus(t, "thispage_corpus.json", corpus)
	}
}

func TestRedirectTargetMatchesGolden(t *testing.T) {
	var b strings.Builder
	for _, raw := range redirectCorpus {
		name, params := ParsePath(raw, "")
		target, ok := RedirectTarget(name, params)
		if !ok {
			target = "<none>"
		}
		fmt.Fprintf(&b, "%s -> %s\n", raw, target)
	}
	checkGolden(t, "redirect.golden", b.String())
	if *update {
		writeCorpus(t, "redirect_corpus.json", redirectCorpus)
	}
}

func paramsForCorpus(params Params) []map[string]any {
	out := make([]map[string]any, 0, len(params))
	for _, param := range params {
		entry := map[string]any{"key": param.Key, "value": param.Value}
		if param.Bare {
			entry["value"] = nil
		}
		out = append(out, entry)
	}
	return out
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) = %v, want nil", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) = %v, want nil", path, err)
	}
	if got != string(want) {
		t.Errorf("%s differs from the golden file; first difference at %s", name, firstDiff(got, string(want)))
	}
}

func firstDiff(got, want string) string {
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
		if gotLines[i] != wantLines[i] {
			return fmt.Sprintf("line %d: got %q, want %q", i+1, gotLines[i], wantLines[i])
		}
	}
	return fmt.Sprintf("line count: got %d, want %d", len(gotLines), len(wantLines))
}

func writeCorpus(t *testing.T, name string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() = %v, want nil", err)
	}
	path := filepath.Join("testdata", name)
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) = %v, want nil", path, err)
	}
}
