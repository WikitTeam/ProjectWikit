package social

import (
	"net/url"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

const (
	MaxLinks      = 12
	maxValueRunes = 200
)

func clip(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > maxValueRunes {
		return string(r[:maxValueRunes])
	}
	return s
}

func FieldsFrom(form url.Values) []db.ProfileField {
	var out []db.ProfileField
	for _, key := range db.ProfileFieldKeys {
		if value := clip(form.Get("field_" + key)); value != "" {
			out = append(out, db.ProfileField{Label: key, Value: value})
		}
	}
	return out
}

func LinksFrom(form url.Values) []db.ProfileLink {
	platforms, values := form["link_platform"], form["link_value"]
	var out []db.ProfileLink
	for i := range platforms {
		if i >= len(values) || len(out) == MaxLinks {
			break
		}
		value := clip(values[i])
		if _, ok := Find(platforms[i]); !ok || value == "" {
			continue
		}
		out = append(out, db.ProfileLink{Platform: platforms[i], Value: value})
	}
	return out
}
