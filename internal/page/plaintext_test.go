package page

import (
	"strings"
	"testing"
)

const sample = `[[module CSS]]
.x { color: red; }
[[/module]]
+ Item #: SCP-173

**Object Class:** //Euclid//, see [[[scp-172|the other one]]] and [http://example.com the site].

[[div class="note"]]
Second paragraph with ##red|colour## and [!-- hidden --] text.
[[/div]]`

func TestPlainText(t *testing.T) {
	got := collapse(plainText(sample))
	for _, gone := range []string{"color: red", "**", "//", "[[", "hidden", "##"} {
		if strings.Contains(got, gone) {
			t.Errorf("plainText() = %q, want no %q", got, gone)
		}
	}
	for _, kept := range []string{"Item #: SCP-173", "Object Class: Euclid", "the other one", "the site", "colour"} {
		if !strings.Contains(got, kept) {
			t.Errorf("plainText() = %q, want %q", got, kept)
		}
	}
}

func TestPreviewCutsAndMarks(t *testing.T) {
	if got := preview("one two three", 7); got != "one two..." {
		t.Errorf("preview(7) = %q, want %q", got, "one two...")
	}
	if got := preview("short", 200); got != "short" {
		t.Errorf("preview(200) = %q, want %q", got, "short")
	}
}

func TestFirstParagraphSkipsHeadings(t *testing.T) {
	want := "Object Class: Euclid, see the other one and the site."
	if got := firstParagraph(sample); got != want {
		t.Errorf("firstParagraph() = %q, want %q", got, want)
	}
}

func TestPreviewLengthOf(t *testing.T) {
	if n, ok := previewLengthOf("preview(50)"); !ok || n != 50 {
		t.Errorf("previewLengthOf(preview(50)) = %d, %v, want 50, true", n, ok)
	}
	if _, ok := previewLengthOf("preview(x)"); ok {
		t.Errorf("previewLengthOf(preview(x)) ok = true, want false")
	}
}

func TestListingVars(t *testing.T) {
	got := ListingVars("%%index%%/%%total%%/%%limit%%/%%total_or_limit%%", nil, 2, 40, "5", 5)
	if got != "2/40/5/5" {
		t.Errorf("ListingVars() = %q, want %q", got, "2/40/5/5")
	}
}

func TestListingVarsAliases(t *testing.T) {
	src := newFakeSource()
	src.source = "First line of text.\n\nSecond."
	v := NewVars(testArticle(), nil, src, nil)
	got := ListingVars("%%page_name%%|%%full_page_name%%|%%description%%|%%text%%", v, 1, 1, "", 1)
	want := "main|main|First line of text.|First line of text.\n\nSecond."
	if got != want {
		t.Errorf("ListingVars() = %q, want %q", got, want)
	}
	if got := ListingVars("%%summary%%|%%preview(5)%%", v, 1, 1, "", 1); got != "First line of text.|First..." {
		t.Errorf("ListingVars(summary, preview) = %q, want %q", got, "First line of text.|First...")
	}
}

func TestPlainTextDropsRules(t *testing.T) {
	if got := collapse(plainText("one\n----\ntwo\n=====\n")); got != "one two" {
		t.Errorf("plainText() = %q, want %q", got, "one two")
	}
}
