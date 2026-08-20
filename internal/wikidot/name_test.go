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
		{"转小写", "SCP-173", "scp-173"},
		{"保留分类", "SCP:173", "scp:173"},
		{"去掉默认分类", "_default:foo", "foo"},
		{"空格变连字符", "Foo Bar", "foo-bar"},
		{"连续非法字符只出一个连字符", "foo   bar", "foo-bar"},
		{"首尾连字符去掉", "--foo--", "foo"},
		{"连续冒号折叠", "a::b", "a:b"},
		{"首尾冒号去掉", "::a::", "a"},
		{"只取第一个冒号做分隔", "a:b:c", "a:b:c"},
		{"空串", "", ""},
		{"拉丁重音字符去重音", "Café", "cafe"},
		{"下划线保留", "foo_bar", "foo_bar"},
		{"数字保留", "scp-001-ex", "scp-001-ex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q，期望 %q", tt.in, got, tt.want)
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
		{"逐字转写", "тест", "test"},
		{"ё 经 NFD 退化为 е", "Ёлка", "elka"},
		{"й 有映射", "йод", "iod"},
		{"软硬音符被删除", "тьма", "tma"},
		{"ж 与 з 都映射到 z", "жз", "zz"},
		{"ч 与 ц 都映射到 c", "чц", "cc"},
		{"映射表没有 ш，落到非法字符分支", "Школа", "kola"},
		{"映射表没有 щ", "щи", "i"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q，期望 %q", tt.in, got, tt.want)
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
				t.Errorf("Normalize(%q) = %q，期望 %q", tt.in, got, tt.want)
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
				t.Errorf("Split(%q) category = %q，期望 %q", tt.in, category, tt.wantCategory)
			}
			if name != tt.wantName {
				t.Errorf("Split(%q) name = %q，期望 %q", tt.in, name, tt.wantName)
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
				t.Errorf("Denormalize(%q) = %q，期望 %q", tt.in, got, tt.want)
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
		{"普通名字", "scp-173", true},
		{"带分类", "scp:173", true},
		{"大写也认", "SCP-173", true},
		{"空串", "", false},
		{"保留名 api", "api", false},
		{"保留名大写", "API", false},
		{"保留名 local--files", "local--files", false},
		{"中文", "测试", false},
		{"空格", "foo bar", false},
		{"斜杠", "foo/bar", false},
		{"分类为空", ":173", false},
		{"名字为空", "scp:", false},
		{"128 字符", strings.Repeat("a", 128), true},
		{"129 字符", strings.Repeat("a", 129), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NameAllowed(tt.in); got != tt.want {
				t.Errorf("NameAllowed(%q) = %v，期望 %v", tt.in, got, tt.want)
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
				t.Errorf("NameAllowed(Normalize(%q)) = NameAllowed(%q) = false，期望 true", in, got)
			}
		})
	}
}
