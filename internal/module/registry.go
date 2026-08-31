package module

import (
	"slices"
	"strings"
)

type Info struct {
	Name       string
	HasContent bool
	Removed    bool
}

var registry = map[string]Info{
	"applicationform":      {Name: "applicationform"},
	"button":               {Name: "button"},
	"countpages":           {Name: "countpages", HasContent: true},
	"css":                  {Name: "css", HasContent: true},
	"forumcategory":        {Name: "forumcategory"},
	"forumnewpost":         {Name: "forumnewpost"},
	"forumnewthread":       {Name: "forumnewthread"},
	"forumpost":            {Name: "forumpost"},
	"forumstart":           {Name: "forumstart"},
	"forumthread":          {Name: "forumthread"},
	"interwiki":            {Name: "interwiki", HasContent: true, Removed: true},
	"listpages":            {Name: "listpages", HasContent: true},
	"listusers":            {Name: "listusers", HasContent: true},
	"members":              {Name: "members", HasContent: true},
	"membershipbypassword": {Name: "membershipbypassword"},
	"newpage":              {Name: "newpage"},
	"pagedescription":      {Name: "pagedescription", HasContent: true},
	"pageimage":            {Name: "pageimage"},
	"pagesbytag":           {Name: "pagesbytag"},
	"rat":                  {Name: "rat", Removed: true},
	"rate":                 {Name: "rate"},
	"recentposts":          {Name: "recentposts"},
	"redirect":             {Name: "redirect"},
	"search":               {Name: "search"},
	"sitechanges":          {Name: "sitechanges"},
	"tagcloud":             {Name: "tagcloud"},
	"time":                 {Name: "time", HasContent: true},
	"wantedpages":          {Name: "wantedpages"},
}

func Lookup(name string) (Info, bool) {
	info, ok := registry[strings.ToLower(strings.TrimSpace(name))]
	return info, ok
}

func HasContent(name string) bool {
	info, ok := Lookup(name)
	return ok && !info.Removed && info.HasContent
}

// Ported is derived from the renderers that registered themselves, so the list
// cannot drift from the code that answers.
func Ported(name string) bool {
	info, ok := Lookup(name)
	if !ok {
		return false
	}
	_, has := renderers[info.Name]
	return has
}

func All() []Info {
	out := make([]Info, 0, len(registry))
	for _, info := range registry {
		out = append(out, info)
	}
	slices.SortFunc(out, func(a, b Info) int { return strings.Compare(a.Name, b.Name) })
	return out
}
