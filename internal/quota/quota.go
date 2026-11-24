// Package quota answers how large an upload may be and how much a site may hold.
package quota

import (
	"fmt"
	"strconv"
	"strings"
)

const DefaultMaxFile int64 = 35 << 20

// A nil field defers to the layer below it; zero means no ceiling.
type Rule struct {
	MaxFile   *int64
	SiteQuota *int64
}

type Limits struct {
	MaxFile   int64
	SiteQuota int64
}

// The command line wins over pwikit.toml for a site, since it is the setting
// someone made most deliberately.
type Policy struct {
	Default     Rule
	Sites       map[string]Rule
	KeepDeleted bool
	// Ceilings over the whole instance, counting every site together.
	InstanceLive  int64
	InstanceTotal int64
}

func (p *Policy) For(slug string, override Rule) Limits {
	l := Limits{MaxFile: DefaultMaxFile}
	for _, r := range []Rule{p.Default, p.Sites[slug], override} {
		if r.MaxFile != nil {
			l.MaxFile = *r.MaxFile
		}
		if r.SiteQuota != nil {
			l.SiteQuota = *r.SiteQuota
		}
	}
	return l
}

var units = map[string]int64{"B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30, "TB": 1 << 40}

func ParseSize(spec string) (int64, error) {
	spec = strings.TrimSpace(spec)
	digits := strings.TrimLeft(spec, "0123456789")
	number, err := strconv.ParseInt(spec[:len(spec)-len(digits)], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("no leading number in %q", spec)
	}
	unit := strings.ToUpper(strings.TrimSpace(digits))
	if unit == "" {
		return number, nil
	}
	scale, ok := units[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit %q", unit)
	}
	return number * scale, nil
}

// An empty setting is left for the next layer to decide.
func ParseRule(maxFile, siteQuota string) (Rule, error) {
	var r Rule
	for _, f := range []struct {
		raw  string
		into **int64
		name string
	}{{maxFile, &r.MaxFile, "max_file"}, {siteQuota, &r.SiteQuota, "site_quota"}} {
		if strings.TrimSpace(f.raw) == "" {
			continue
		}
		n, err := ParseSize(f.raw)
		if err != nil {
			return Rule{}, fmt.Errorf("%s: %w", f.name, err)
		}
		*f.into = &n
	}
	return r, nil
}

func Format(n int64) string {
	switch {
	case n >= 1<<40 && n%(1<<40) == 0:
		return strconv.FormatInt(n>>40, 10) + " TB"
	case n >= 1<<30 && n%(1<<30) == 0:
		return strconv.FormatInt(n>>30, 10) + " GB"
	case n >= 1<<20 && n%(1<<20) == 0:
		return strconv.FormatInt(n>>20, 10) + " MB"
	case n >= 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + " MB"
	case n >= 1<<10:
		return strconv.FormatFloat(float64(n)/(1<<10), 'f', 0, 64) + " KB"
	}
	return strconv.FormatInt(n, 10) + " B"
}
