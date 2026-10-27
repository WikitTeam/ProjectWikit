package sitemap

import (
	"strconv"
	"testing"
)

func entriesOf(n int) []entry {
	out := make([]entry, n)
	for i := range out {
		out[i] = entry{Loc: "https://a.test/p-" + strconv.Itoa(i)}
	}
	return out
}

func TestDocumentSmallIsOneURLSet(t *testing.T) {
	doc, ok := document(entriesOf(3), "https://a.test", "")
	set, isSet := doc.(urlSet)
	if !ok || !isSet {
		t.Fatalf("document(3, \"\") = %T, %v, want urlSet, true", doc, ok)
	}
	if len(set.URLs) != 3 {
		t.Errorf("len(URLs) = %d, want 3", len(set.URLs))
	}
}

func TestDocumentLargeIsIndex(t *testing.T) {
	doc, ok := document(entriesOf(maxPerFile+1), "https://a.test", "")
	idx, isIndex := doc.(index)
	if !ok || !isIndex {
		t.Fatalf("document(max+1, \"\") = %T, %v, want index, true", doc, ok)
	}
	if len(idx.Sitemaps) != 2 {
		t.Fatalf("len(Sitemaps) = %d, want 2", len(idx.Sitemaps))
	}
	if got, want := idx.Sitemaps[1].Loc, "https://a.test/sitemap.xml?part=2"; got != want {
		t.Errorf("Sitemaps[1].Loc = %q, want %q", got, want)
	}
}

func TestDocumentPart(t *testing.T) {
	doc, ok := document(entriesOf(maxPerFile+1), "https://a.test", "2")
	set, isSet := doc.(urlSet)
	if !ok || !isSet {
		t.Fatalf("document(max+1, \"2\") = %T, %v, want urlSet, true", doc, ok)
	}
	if len(set.URLs) != 1 {
		t.Errorf("len(URLs) = %d, want 1", len(set.URLs))
	}
}

func TestDocumentPartOutOfRange(t *testing.T) {
	for _, part := range []string{"0", "3", "x"} {
		if _, ok := document(entriesOf(maxPerFile+1), "https://a.test", part); ok {
			t.Errorf("document(max+1, %q) ok = true, want false", part)
		}
	}
}
