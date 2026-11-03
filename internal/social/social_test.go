package social

import "testing"

func TestHref(t *testing.T) {
	get := func(key string) Platform { p, _ := Find(key); return p }
	cases := []struct{ key, value, want string }{
		{"github", "octocat", "https://github.com/octocat"},
		{"x", "@someone", "https://x.com/someone"},
		{"bilibili", "12345", "https://space.bilibili.com/12345"},
		{"telegram", "https://t.me/abc", "https://t.me/abc"},
		{"qq", "123456789", ""},
		{"wechat", "wxid_abc", ""},
		{"email", "me@example.com", "mailto:me@example.com"},
		{"email", "not an email", ""},
		{"website", "example.com", "https://example.com"},
		{"other", "javascript:alert(1)", ""},
		{"github", "javascript:alert(1)", ""},
		{"website", "data:text/html,x", ""},
		{"github", "a/b", ""},
		{"github", "", ""},
	}
	for _, c := range cases {
		if got := Href(get(c.key), c.value); got != c.want {
			t.Errorf("Href(%s, %q) = %q, want %q", c.key, c.value, got, c.want)
		}
	}
}

func TestEveryIconedPlatformIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Platforms {
		if seen[p.Key] {
			t.Errorf("Platforms has %q twice", p.Key)
		}
		seen[p.Key] = true
		if p.Icon == "" && p.FA == "" {
			t.Errorf("Platforms[%q] has no icon", p.Key)
		}
	}
}
