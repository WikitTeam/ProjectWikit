package admin

import "github.com/WikitTeam/ProjectWikit/internal/i18n"

// The stored value of a category that takes the site's answer.
const followSite = "default"

type settingChoice struct {
	Value string
	Label string
}

// A site row written by an older release can still hold the value that followed
// something, and a site has nothing to follow.
func siteSetting(value, fallback string) string {
	if value == "" || value == followSite {
		return fallback
	}
	return value
}

// Spelling out what following the site resolves to saves the reader a trip to
// the site settings.
func settingChoices(loc *i18n.Localizer, prefix string, values []string, siteValue string) []settingChoice {
	out := make([]settingChoice, 0, len(values))
	for _, value := range values {
		label := loc.T("admin.setting-follow", "mode", loc.T(prefix+siteValue))
		if value != followSite {
			label = loc.T(prefix + value)
		}
		out = append(out, settingChoice{Value: value, Label: label})
	}
	return out
}
