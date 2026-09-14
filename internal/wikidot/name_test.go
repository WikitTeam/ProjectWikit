package wikidot

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"lowercased", "SCP-173", "scp-173"},
		{"category kept", "SCP:173", "scp:173"},
		{"default category stripped", "_default:foo", "foo"},
		{"space becomes a hyphen", "Foo Bar", "foo-bar"},
		{"a run of disallowed chars yields one hyphen", "foo   bar", "foo-bar"},
		{"leading and trailing hyphens stripped", "--foo--", "foo"},
		{"repeated colons collapsed", "a::b", "a:b"},
		{"leading and trailing colons stripped", "::a::", "a"},
		{"splits on the first colon only", "a:b:c", "a:b:c"},
		{"empty string", "", ""},
		{"latin accents stripped", "Café", "cafe"},
		{"underscore kept", "foo_bar", "foo_bar"},
		{"digits kept", "scp-001-ex", "scp-001-ex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeCyrillic(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"transliterated letter by letter", "тест", "test"},
		{"ё decomposes to е via NFD", "Ёлка", "elka"},
		{"й has a mapping", "йод", "iod"},
		{"soft and hard signs dropped", "тьма", "tma"},
		{"zh and z both map to z", "жз", "zz"},
		{"ch and ts both map to c", "чц", "cc"},
		{"ш has no mapping and falls into the disallowed branch", "Школа", "kola"},
		{"щ has no mapping", "щи", "i"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeDropsCJK(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"测试", ""},
		{"测试page", "page"},
		{"page 测试", "page"},
		{"前测试后", ""},
		{"a测b", "a-b"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		in           string
		wantCategory string
		wantName     string
	}{
		{"scp:173", "scp", "173"},
		{"173", DefaultCategory, "173"},
		{"a:b:c", "a", "b:c"},
		{"", DefaultCategory, ""},
		{":x", "", "x"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			category, name := Split(tt.in)
			if category != tt.wantCategory {
				t.Errorf("Split(%q) category = %q, want %q", tt.in, category, tt.wantCategory)
			}
			if name != tt.wantName {
				t.Errorf("Split(%q) name = %q, want %q", tt.in, name, tt.wantName)
			}
		})
	}
}

func TestDenormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"173", "_default:173"},
		{"scp:173", "scp:173"},
		{"", "_default:"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := Denormalize(tt.in); got != tt.want {
				t.Errorf("Denormalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNameAllowed(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"plain name", "scp-173", true},
		{"with category", "scp:173", true},
		{"uppercase accepted", "SCP-173", true},
		{"empty string", "", false},
		{"reserved name api", "api", false},
		{"reserved name uppercased", "API", false},
		{"reserved name pw-api", "pw-api", false},
		{"reserved name local--files", "local--files", false},
		{"chinese", "测试", false},
		{"space", "foo bar", false},
		{"slash", "foo/bar", false},
		{"empty category", ":173", false},
		{"empty name", "scp:", false},
		{"128 chars", strings.Repeat("a", 128), true},
		{"129 chars", strings.Repeat("a", 129), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NameAllowed(tt.in); got != tt.want {
				t.Errorf("NameAllowed(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeOutputIsAllowed(t *testing.T) {
	inputs := []string{"SCP-173", "Foo Bar", "тест", "Café", "a::b"}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			got := Normalize(in)
			if !NameAllowed(got) {
				t.Errorf("NameAllowed(Normalize(%q)) = NameAllowed(%q) = false, want true", in, got)
			}
		})
	}
}
