// Package roles holds the role model and the visual decorations it produces.
package roles

import (
	"errors"
	"slices"
	"strings"
)

type InlineVisualMode string

const (
	InlineHidden InlineVisualMode = "hidden"
	InlineBadge  InlineVisualMode = "badge"
	InlineIcon   InlineVisualMode = "icon"
)

type ProfileVisualMode string

const (
	ProfileHidden ProfileVisualMode = "hidden"
	ProfileBadge  ProfileVisualMode = "badge"
	ProfileStatus ProfileVisualMode = "status"
)

var ErrNoSVGTag = errors.New("roles: icon has no <svg tag")

type Role struct {
	ID                int64
	Slug              string
	Name              string
	ShortName         string
	CategoryID        *int64
	Index             int
	IsStaff           bool
	GroupVotes        bool
	InlineVisualMode  InlineVisualMode
	ProfileVisualMode ProfileVisualMode
	Color             string
	Icon              string
	BadgeText         string
	BadgeBg           string
	BadgeTextColor    string
	BadgeShowBorder   bool
}

type Badge struct {
	Text       string
	Bg         string
	TextColor  string
	ShowBorder bool
	Tooltip    string
}

type Icon struct {
	Icon    string
	Color   string
	Tooltip string
}

type Tails struct {
	Badges []Badge
	Icons  []Icon
}

// IconLoader reads a role icon, given the path stored in the icon column
// relative to the media root.
type IconLoader func(path string) (string, error)

// NameTails builds the decorations shown next to a user name. Roles must
// arrive ordered by index; the first role of a given (mode, category) pair
// wins, which is what Django's DISTINCT ON over that pair resolves to.
func NameTails(rs []Role, load IconLoader) (Tails, error) {
	type key struct {
		mode     InlineVisualMode
		category int64
	}
	seen := make(map[key]bool)
	tails := Tails{}
	for _, role := range rs {
		if role.InlineVisualMode == InlineHidden {
			continue
		}
		if role.CategoryID != nil {
			k := key{role.InlineVisualMode, *role.CategoryID}
			if seen[k] {
				continue
			}
			seen[k] = true
		}
		badge, icon, err := role.NameTail(load)
		if err != nil {
			return Tails{}, err
		}
		switch {
		case badge != nil:
			tails.Badges = append(tails.Badges, *badge)
		case icon != nil:
			tails.Icons = append(tails.Icons, *icon)
		}
	}
	return tails, nil
}

// NameTail returns at most one of the two; a role in icon mode with no icon
// file produces neither.
func (r Role) NameTail(load IconLoader) (*Badge, *Icon, error) {
	switch r.InlineVisualMode {
	case InlineBadge:
		text := r.BadgeText
		if text == "" {
			text = r.Slug
		}
		return &Badge{
			Text:       text,
			Bg:         r.BadgeBg,
			TextColor:  r.BadgeTextColor,
			ShowBorder: r.BadgeShowBorder,
			Tooltip:    r.Name,
		}, nil, nil
	case InlineIcon:
		if r.Icon == "" {
			return nil, nil, nil
		}
		svg, err := load(r.Icon)
		if err != nil {
			return nil, nil, err
		}
		colored, err := ColorizeIcon(svg, r.Color)
		if err != nil {
			return nil, nil, err
		}
		return nil, &Icon{Icon: colored, Color: r.Color, Tooltip: r.Name}, nil
	}
	return nil, nil, nil
}

// ColorizeIcon injects a colour rule after the opening <svg> and percent
// encodes the result for a data: URI. The style element is spliced in without
// its final > because the join below supplies one.
func ColorizeIcon(svg, color string) (string, error) {
	start := strings.Index(svg, "<svg")
	if start < 0 {
		return "", ErrNoSVGTag
	}
	parts := strings.Split(svg[start:], ">")
	parts = slices.Insert(parts, 1, "<style>svg{color:"+color+"}</style")
	return quote(strings.Join(parts, ">")), nil
}

const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.-~/"

// quote is urllib.parse.quote with its default safe="/". Neither url.PathEscape
// nor url.QueryEscape matches: they leave $&+,;=:@ alone and encode ~.
func quote(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		const hex = "0123456789ABCDEF"
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0F])
	}
	return b.String()
}
