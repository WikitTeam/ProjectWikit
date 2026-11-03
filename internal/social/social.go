// Package social knows the platforms a profile can link to and turns what a
// member typed into something safe to put on the page.
package social

import (
	"net/url"
	"strings"
)

type Platform struct {
	Key   string
	Name  string
	Icon  string
	FA    string
	Color string
	// Handle builds the address from a bare handle. Empty means a handle alone
	// leads nowhere, so the page offers to copy it instead.
	Handle string
}

var Platforms = []Platform{
	{Key: "telegram", Name: "Telegram", Icon: "telegram", Color: "#26A5E4", Handle: "https://t.me/"},
	{Key: "discord", Name: "Discord", Icon: "discord", Color: "#5865F2"},
	{Key: "qq", Name: "QQ", Icon: "qq", Color: "#1EBAFC"},
	{Key: "wechat", Name: "WeChat", Icon: "wechat", Color: "#07C160"},
	{Key: "bilibili", Name: "Bilibili", Icon: "bilibili", Color: "#00A1D6", Handle: "https://space.bilibili.com/"},
	{Key: "github", Name: "GitHub", Icon: "github", Color: "#181717", Handle: "https://github.com/"},
	{Key: "whatsapp", Name: "WhatsApp", Icon: "whatsapp", Color: "#25D366"},
	{Key: "x", Name: "X", Icon: "x", Color: "#000000", Handle: "https://x.com/"},
	{Key: "facebook", Name: "Facebook", Icon: "facebook", Color: "#0866FF", Handle: "https://facebook.com/"},
	{Key: "steam", Name: "Steam", Icon: "steam", Color: "#000000", Handle: "https://steamcommunity.com/id/"},
	{Key: "instagram", Name: "Instagram", Icon: "instagram", Color: "#E4405F", Handle: "https://instagram.com/"},
	{Key: "youtube", Name: "YouTube", Icon: "youtube", Color: "#FF0000", Handle: "https://youtube.com/@"},
	{Key: "reddit", Name: "Reddit", Icon: "reddit", Color: "#FF4500", Handle: "https://reddit.com/user/"},
	{Key: "weibo", Name: "Weibo", Icon: "sinaweibo", Color: "#E6162D", Handle: "https://weibo.com/"},
	{Key: "zhihu", Name: "Zhihu", Icon: "zhihu", Color: "#0084FF", Handle: "https://www.zhihu.com/people/"},
	{Key: "tiktok", Name: "TikTok", Icon: "tiktok", Color: "#000000", Handle: "https://tiktok.com/@"},
	{Key: "twitch", Name: "Twitch", Icon: "twitch", Color: "#9146FF", Handle: "https://twitch.tv/"},
	{Key: "patreon", Name: "Patreon", Icon: "patreon", Color: "#000000", Handle: "https://patreon.com/"},
	{Key: "line", Name: "LINE", Icon: "line", Color: "#00C300"},
	{Key: "pixiv", Name: "pixiv", Icon: "pixiv", Color: "#0096FA", Handle: "https://www.pixiv.net/users/"},
	{Key: "email", Name: "Email", FA: "fas fa-envelope", Color: "#5F6B7A"},
	{Key: "website", Name: "Website", FA: "fas fa-globe", Color: "#5F6B7A"},
	{Key: "other", Name: "", FA: "fas fa-link", Color: "#5F6B7A"},
}

func Find(key string) (Platform, bool) {
	for _, p := range Platforms {
		if p.Key == key {
			return p, true
		}
	}
	return Platform{}, false
}

// Only http, https and mailto ever come out. Empty means the value is only
// shown and copied.
func Href(p Platform, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if p.Key == "email" {
		if strings.Contains(value, "@") && !strings.ContainsAny(value, " <>\"'") {
			return "mailto:" + value
		}
		return ""
	}
	if u, err := url.Parse(value); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		return u.String()
	}
	if p.Key == "website" && !strings.ContainsAny(value, " <>\"'") && strings.Contains(value, ".") && !strings.Contains(value, ":") {
		return "https://" + value
	}
	if p.Handle != "" {
		handle := strings.TrimPrefix(value, "@")
		if handle != "" && !strings.ContainsAny(handle, " /?#:<>\"'") {
			return p.Handle + url.PathEscape(handle)
		}
	}
	return ""
}
