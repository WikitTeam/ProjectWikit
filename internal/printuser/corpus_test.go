package printuser

type roleSpec struct {
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	ShortName         string `json:"short_name"`
	Index             int    `json:"index"`
	Category          string `json:"category"`
	InlineVisualMode  string `json:"inline_visual_mode"`
	ProfileVisualMode string `json:"profile_visual_mode"`
	Color             string `json:"color"`
	Icon              string `json:"icon"`
	BadgeText         string `json:"badge_text"`
	BadgeBg           string `json:"badge_bg"`
	BadgeTextColor    string `json:"badge_text_color"`
	BadgeShowBorder   bool   `json:"badge_show_border"`
}

type userSpec struct {
	ID              int64  `json:"id"`
	Type            string `json:"type"`
	Username        string `json:"username"`
	WikidotUsername string `json:"wikidot_username"`
	DisplayName     string `json:"display_name"`
	Avatar          string `json:"avatar"`
	IsActive        bool   `json:"is_active"`
}

type caseSpec struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Avatar   bool      `json:"avatar"`
	Hover    bool      `json:"hover"`
	External string    `json:"external"`
	User     *userSpec `json:"user"`
	Roles    []string  `json:"roles"`
}

type corpusFile struct {
	Roles []roleSpec        `json:"roles"`
	Icons map[string]string `json:"icons"`
	Cases []caseSpec        `json:"cases"`
}

const iconPath = "-/roles/probe.svg"

// The leading <?xml?> line is there on purpose: get_name_tail starts at the
// first <svg, so everything before it must disappear.
const iconSVG = "<?xml version=\"1.0\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 16 16\"><path d=\"M1 2 L3 4\"/></svg>\n"

// Every field is spelled out on both sides of the comparison so a Django model
// default can never quietly become the Go default.
func newRole(slug string, index int) roleSpec {
	return roleSpec{
		Slug:              slug,
		Index:             index,
		InlineVisualMode:  "hidden",
		ProfileVisualMode: "hidden",
		Color:             "#000000",
		BadgeBg:           "#808080",
		BadgeTextColor:    "#ffffff",
	}
}

func newUser(id int64, username string) *userSpec {
	return &userSpec{ID: id, Type: "normal", Username: username, IsActive: true}
}

func badgeRole(slug string, index int, category string) roleSpec {
	r := newRole(slug, index)
	r.InlineVisualMode = "badge"
	r.Category = category
	return r
}

func iconRole(slug string, index int, category string) roleSpec {
	r := newRole(slug, index)
	r.InlineVisualMode = "icon"
	r.Category = category
	r.Icon = iconPath
	return r
}

func corpus() corpusFile {
	staff := badgeRole("staff", 0, "")
	staff.Name = "工作人员"
	staff.BadgeText = "STAFF"
	staff.BadgeBg = "#112233"
	staff.BadgeTextColor = "#ffeedd"
	staff.BadgeShowBorder = true

	noText := badgeRole("no-text", 1, "")

	sameCatFirst := badgeRole("cat-first", 2, "status")
	sameCatFirst.Name = "先来的"
	sameCatFirst.BadgeText = "FIRST"

	sameCatSecond := badgeRole("cat-second", 3, "status")
	sameCatSecond.Name = "后来的"
	sameCatSecond.BadgeText = "SECOND"

	catIcon := iconRole("cat-icon", 4, "status")
	catIcon.Name = "同类图标"
	catIcon.Color = "#ff8800"

	plainIcon := iconRole("plain-icon", 5, "")
	plainIcon.Name = "图标"
	plainIcon.Color = "#00aa55"

	iconNoFile := newRole("icon-no-file", 6)
	iconNoFile.InlineVisualMode = "icon"
	iconNoFile.Name = "没有图标文件"

	hidden := newRole("hidden-role", 7)
	hidden.Name = "隐藏"
	hidden.ProfileVisualMode = "status"

	displayNamed := newUser(102, "displayed")
	displayNamed.DisplayName = "显示名"
	displayNamed.Avatar = "-/users/x.png"

	wd := newUser(103, "claimed")
	wd.Type = "wikidot"
	wd.WikidotUsername = "wd-name"

	wdNoClaim := newUser(104, "wd-only")
	wdNoClaim.Type = "wikidot"
	wdNoClaim.WikidotUsername = "wd-only-name"
	wdNoClaim.DisplayName = "维基点"

	banned := newUser(105, "banned")
	banned.IsActive = false

	bannedWikidot := newUser(106, "banned-wd")
	bannedWikidot.Type = "wikidot"
	bannedWikidot.WikidotUsername = "banned-wd-name"
	bannedWikidot.IsActive = false

	bot := newUser(107, "bot-account")
	bot.Type = "bot"

	bannedBot := newUser(108, "banned-bot")
	bannedBot.Type = "bot"
	bannedBot.IsActive = false

	escaped := newUser(109, "escaper")
	escaped.DisplayName = `a<b>&"'`

	decorated := newUser(110, "decorated")

	return corpusFile{
		Roles: []roleSpec{staff, noText, sameCatFirst, sameCatSecond, catIcon, plainIcon, iconNoFile, hidden},
		Icons: map[string]string{iconPath: iconSVG},
		Cases: []caseSpec{
			{Name: "system", Kind: "system", Avatar: true, Hover: true},
			{Name: "system_bare", Kind: "system"},
			{Name: "anonymous", Kind: "anonymous", Avatar: true, Hover: true},
			{Name: "anonymous_bare", Kind: "anonymous"},
			{Name: "external", Kind: "external", External: "Some User", Avatar: true, Hover: true},
			{Name: "external_bare", Kind: "external", External: "Some User"},
			{Name: "external_escaped", Kind: "external", External: `Tom & "Jerry"`, Avatar: true, Hover: true},
			{Name: "plain", Kind: "user", Avatar: true, Hover: true, User: newUser(101, "plain")},
			{Name: "plain_bare", Kind: "user", User: newUser(101, "plain")},
			{Name: "display_name_and_avatar", Kind: "user", Avatar: true, Hover: true, User: displayNamed},
			{Name: "wikidot", Kind: "user", Avatar: true, Hover: true, User: wd},
			{Name: "wikidot_display_name", Kind: "user", Avatar: true, Hover: true, User: wdNoClaim},
			{Name: "banned", Kind: "user", Avatar: true, Hover: true, User: banned},
			{Name: "banned_wikidot", Kind: "user", Avatar: true, Hover: true, User: bannedWikidot},
			{Name: "bot", Kind: "user", Avatar: true, Hover: true, User: bot},
			{Name: "banned_bot", Kind: "user", Avatar: true, Hover: true, User: bannedBot},
			{Name: "escaped_display_name", Kind: "user", Avatar: true, Hover: true, User: escaped},
			{Name: "badges", Kind: "user", Avatar: true, Hover: true, User: decorated, Roles: []string{"staff", "no-text"}},
			{Name: "same_category_keeps_lowest_index", Kind: "user", Avatar: true, Hover: true, User: decorated, Roles: []string{"cat-second", "cat-first"}},
			{Name: "same_category_different_modes", Kind: "user", Avatar: true, Hover: true, User: decorated, Roles: []string{"cat-first", "cat-second", "cat-icon"}},
			{Name: "icons_before_badges", Kind: "user", Avatar: true, Hover: true, User: decorated, Roles: []string{"plain-icon", "staff"}},
			{Name: "icon_without_file", Kind: "user", Avatar: true, Hover: true, User: decorated, Roles: []string{"icon-no-file"}},
			{Name: "hidden_role", Kind: "user", Avatar: true, Hover: true, User: decorated, Roles: []string{"hidden-role"}},
			{Name: "roles_need_avatar", Kind: "user", User: decorated, Roles: []string{"staff", "plain-icon"}},
			{Name: "banned_beats_roles", Kind: "user", Avatar: true, Hover: true, User: banned, Roles: []string{"staff"}},
		},
	}
}
