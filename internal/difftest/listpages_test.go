package difftest

import "testing"

func TestSortListPagesParamsIsOrderBlind(t *testing.T) {
	a := `data-list-pages-params="{&quot;zebra&quot;: &quot;1&quot;, &quot;alpha&quot;: &quot;2&quot;}"`
	b := `data-list-pages-params="{&quot;alpha&quot;: &quot;2&quot;, &quot;zebra&quot;: &quot;1&quot;}"`

	got, want := string(sortListPagesParams([]byte(a))), string(sortListPagesParams([]byte(b)))
	if got != want {
		t.Errorf("sortListPagesParams(a) = %q, want %q", got, want)
	}
	if want != b {
		t.Errorf("sortListPagesParams(b) = %q, want %q", want, b)
	}
}

func TestSortListPagesParamsKeepsADifferentValue(t *testing.T) {
	a := `data-list-pages-params="{&quot;alpha&quot;: &quot;1&quot;}"`
	b := `data-list-pages-params="{&quot;alpha&quot;: &quot;2&quot;}"`

	if string(sortListPagesParams([]byte(a))) == string(sortListPagesParams([]byte(b))) {
		t.Error("sortListPagesParams(a) = sortListPagesParams(b), want them to differ")
	}
}

func TestSortListPagesParamsLeavesJunkAlone(t *testing.T) {
	in := `data-list-pages-params="not json"`
	if got := string(sortListPagesParams([]byte(in))); got != in {
		t.Errorf("sortListPagesParams(%q) = %q, want it unchanged", in, got)
	}
}

func TestSortListPagesParamsKeepsANull(t *testing.T) {
	in := `data-list-pages-params="{&quot;alpha&quot;: null}"`
	if got := string(sortListPagesParams([]byte(in))); got != in {
		t.Errorf("sortListPagesParams(%q) = %q, want it unchanged", in, got)
	}
}
