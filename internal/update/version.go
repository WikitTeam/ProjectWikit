// Package update answers whether a newer release exists and how to put it in place.
package update

import (
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
	Pre                 string
}

var versionPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)

func ParseVersion(s string) (Version, bool) {
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return Version{}, false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Version{Major: major, Minor: minor, Patch: patch, Pre: m[4]}, true
}

func (v Version) Compare(o Version) int {
	for _, pair := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Patch, o.Patch}} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case v.Pre == o.Pre:
		return 0
	case v.Pre == "":
		return 1
	case o.Pre == "":
		return -1
	}
	return comparePre(v.Pre, o.Pre)
}

func comparePre(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		an, aErr := strconv.Atoi(as[i])
		bn, bErr := strconv.Atoi(bs[i])
		switch {
		case aErr == nil && bErr == nil:
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		default:
			if c := strings.Compare(as[i], bs[i]); c != 0 {
				return c
			}
		}
	}
	switch {
	case len(as) < len(bs):
		return -1
	case len(as) > len(bs):
		return 1
	}
	return 0
}

func Newer(candidate, current string) bool {
	c, ok := ParseVersion(candidate)
	if !ok {
		return false
	}
	have, ok := ParseVersion(current)
	if !ok {
		return false
	}
	return c.Compare(have) > 0
}

func PostgresMajor(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return major
}
