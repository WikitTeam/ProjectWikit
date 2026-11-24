package quota

import "testing"

func ptr(n int64) *int64 { return &n }

func TestForDefaults(t *testing.T) {
	var p Policy
	got := p.For("main", Rule{})
	if got.MaxFile != DefaultMaxFile {
		t.Errorf("For().MaxFile = %d, want %d", got.MaxFile, DefaultMaxFile)
	}
	if got.SiteQuota != 0 {
		t.Errorf("For().SiteQuota = %d, want 0", got.SiteQuota)
	}
}

func TestForLayers(t *testing.T) {
	p := Policy{
		Default: Rule{MaxFile: ptr(10), SiteQuota: ptr(100)},
		Sites:   map[string]Rule{"big": {SiteQuota: ptr(1000)}},
	}
	if got := p.For("small", Rule{}); got != (Limits{MaxFile: 10, SiteQuota: 100}) {
		t.Errorf("For(small) = %+v, want {10 100}", got)
	}
	if got := p.For("big", Rule{}); got != (Limits{MaxFile: 10, SiteQuota: 1000}) {
		t.Errorf("For(big) = %+v, want {10 1000}", got)
	}
	if got := p.For("big", Rule{MaxFile: ptr(0)}); got != (Limits{MaxFile: 0, SiteQuota: 1000}) {
		t.Errorf("For(big, override 0) = %+v, want {0 1000}", got)
	}
}

func TestParseRuleLeavesEmptyUnset(t *testing.T) {
	r, err := ParseRule("", "2GB")
	if err != nil {
		t.Fatalf("ParseRule() err = %v, want nil", err)
	}
	if r.MaxFile != nil {
		t.Errorf("ParseRule(\"\").MaxFile = %v, want nil", *r.MaxFile)
	}
	if r.SiteQuota == nil || *r.SiteQuota != 2<<30 {
		t.Errorf("ParseRule(2GB).SiteQuota = %v, want %d", r.SiteQuota, int64(2<<30))
	}
}

func TestFormat(t *testing.T) {
	for n, want := range map[int64]string{35 << 20: "35 MB", 2 << 30: "2 GB", 500: "500 B", 1536 << 10: "1.5 MB"} {
		if got := Format(n); got != want {
			t.Errorf("Format(%d) = %q, want %q", n, got, want)
		}
	}
}
