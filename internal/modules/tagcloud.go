package modules

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("tagcloud", renderTagCloud) }

const (
	defaultMinSize   = "100%"
	defaultMaxSize   = "300%"
	defaultMinColor  = "128,128,192"
	defaultMaxColor  = "64,64,128"
	defaultTagTarget = "system:page-tags"
)

func renderTagCloud(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}

	// Both bounds have to be named together, since interpolating between a
	// default and a chosen one is not what the caller asked for.
	minSize, maxSize := defaultMinSize, defaultMaxSize
	_, hasMin := params["minfontsize"]
	_, hasMax := params["maxfontsize"]
	if hasMin && hasMax {
		minSize, maxSize = params["minfontsize"], params["maxfontsize"]
	}
	minColor, maxColor := defaultMinColor, defaultMaxColor
	_, hasMinColor := params["mincolor"]
	_, hasMaxColor := params["maxcolor"]
	if hasMinColor && hasMaxColor {
		minColor, maxColor = params["mincolor"], params["maxcolor"]
	}

	// A limit that does not parse, and a negative one, both blow up before the
	// module can say anything about it, so the reader gets the generic block.
	var limit *int
	if raw, ok := params["limit"]; ok {
		value, err := wikinum.Int(raw)
		if err != nil || value < 0 {
			return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
		}
		limit = &value
	}

	target, ok := params["target"]
	if !ok {
		target = defaultTagTarget
	}

	tags, err := env.Data.TagCloud(limit)
	if err != nil {
		return "", err
	}
	scale, err := newTagScale(env, tags, minSize, maxSize, minColor, maxColor)
	if err != nil {
		return "", err
	}

	// The query orders by count so the limit cuts the quiet tags, and the
	// display order is the bare name even where the link carries a category.
	shown := make([]db.CloudTag, len(tags))
	copy(shown, tags)
	sort.SliceStable(shown, func(i, j int) bool { return shown[i].Name < shown[j].Name })

	if params["categories"] == "yes" {
		return tagCloudByCategory(shown, scale, target), nil
	}
	return tagCloudFlat(shown, scale, target), nil
}

func tagCloudFlat(tags []db.CloudTag, scale tagScale, target string) string {
	var b strings.Builder
	b.WriteString(`<div class="pages-tag-cloud-box">` + "\n                ")
	for i := range tags {
		b.WriteString("\n                    " +
			tagAnchor(tags[i], tags[i].FullName, scale, target) + "\n                ")
	}
	b.WriteString("\n            </div>")
	return b.String()
}

func tagCloudByCategory(tags []db.CloudTag, scale tagScale, target string) string {
	order := make([]int64, 0, len(tags))
	byCategory := map[int64][]db.CloudTag{}
	for _, tag := range tags {
		if _, seen := byCategory[tag.CategoryID]; !seen {
			order = append(order, tag.CategoryID)
		}
		byCategory[tag.CategoryID] = append(byCategory[tag.CategoryID], tag)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return lessPriority(byCategory[order[i]][0].Priority, byCategory[order[j]][0].Priority)
	})

	var b strings.Builder
	b.WriteString(`<div class="pages-tag-cloud-box">` + "\n                ")
	for _, id := range order {
		group := byCategory[id]
		head := group[0]

		b.WriteString("\n                    " + `<div class="w-collapsible collapsible-block">` +
			"\n                        " + `<div class="collapsible-block-folded" style="display: none;">` +
			"\n                            " + `<h2 class="collapsible-block-link"><a href="javascript:;">+ ` +
			tagCategoryLabel(head) + `</a></h2>` +
			"\n                        </div>" +
			"\n                        " + `<div class="collapsible-block-unfolded" style="display: block;">` +
			"\n                            " + `<div class="collapsible-block-unfolded-link">` +
			"\n                                " + `<h2 class="collapsible-block-link"><a href="javascript:;">- ` +
			tagCategoryLabel(head) + `</a></h2>` +
			"\n                            </div>" +
			"\n                            " + `<div class="collapsible-block-content">` +
			"\n                                ")
		if head.CategoryText != "" {
			b.WriteString("<h4>" + escape.HTML(head.CategoryText) + "</h4>")
		}
		b.WriteString("\n                                " + `<hr style="margin: auto">` +
			"\n                                ")
		for i := range group {
			b.WriteString("\n                                    " +
				tagAnchor(group[i], group[i].Name, scale, target) +
				"\n                                ")
		}
		b.WriteString("\n                            </div>" +
			"\n                        </div>" +
			"\n                    </div>" +
			"\n                    <br>" +
			"\n                ")
	}
	b.WriteString("\n            </div>")
	return b.String()
}

func tagCategoryLabel(tag db.CloudTag) string {
	label := escape.HTML(tag.CategoryName)
	if tag.CategorySlug == "_default" || tag.CategoryName == tag.CategorySlug {
		return label
	}
	return label + ` <pre style="display: inline; font-size: 60%">[` +
		escape.HTML(tag.CategorySlug) + `]</pre>`
}

func tagAnchor(tag db.CloudTag, shown string, scale tagScale, target string) string {
	value := scale.value(tag.Articles)
	link := "/" + target + "/tag/" + page.QuoteAll(tag.FullName)
	return `<a class="tag" href="` + escape.HTML(link) +
		`" style="font-size: ` + scale.size(value) + `; color: ` + scale.color(value) + `">` +
		escape.HTML(shown) + "</a>"
}

// A category with no priority sorts after every category that has one.
func lessPriority(a, b *int) bool {
	if a == nil || b == nil {
		return b == nil && a != nil
	}
	return *a < *b
}

type tagScale struct {
	min, span        int
	minSize, maxSize float64
	unit             string
	minRGB, maxRGB   [3]int
}

func newTagScale(env module.Env, tags []db.CloudTag, minSize, maxSize, minColor, maxColor string) (tagScale, error) {
	var s tagScale
	if len(tags) > 0 {
		low, high := tags[0].Articles, tags[0].Articles
		for _, tag := range tags {
			low = min(low, tag.Articles)
			high = max(high, tag.Articles)
		}
		s.min, s.span = low, high-low
	}

	var minUnit, maxUnit string
	var err error
	if s.minSize, minUnit, err = parseFontSize(env, minSize); err != nil {
		return s, err
	}
	if s.maxSize, maxUnit, err = parseFontSize(env, maxSize); err != nil {
		return s, err
	}
	if minUnit != maxUnit {
		return s, &module.Error{Message: env.Text("module-tagcloud-units", "min", minSize, "max", maxSize)}
	}
	s.unit = minUnit
	if s.minRGB, err = parseTagColor(env, minColor); err != nil {
		return s, err
	}
	if s.maxRGB, err = parseTagColor(env, maxColor); err != nil {
		return s, err
	}
	return s, nil
}

func (s tagScale) value(articles int) float64 {
	if s.span == 0 {
		return 0
	}
	return float64(articles-s.min) / float64(s.span)
}

func (s tagScale) size(value float64) string {
	return fmt.Sprintf("%.4f%s", s.minSize+(s.maxSize-s.minSize)*value, s.unit)
}

func (s tagScale) color(value float64) string {
	channel := func(i int) int {
		v := int(float64(s.minRGB[i]) + float64(s.maxRGB[i]-s.minRGB[i])*value)
		return max(0, min(v, 255))
	}
	return fmt.Sprintf("#%02x%02x%02x", channel(0), channel(1), channel(2))
}

// The digits are taken greedily and whatever follows is the unit, so 1.5em
// reads as a size of one carrying the unit ".5em".
var fontSizePattern = regexp.MustCompile(`^(\d+)([\s\S]*)$`)

func parseFontSize(env module.Env, size string) (float64, string, error) {
	match := fontSizePattern.FindStringSubmatch(size)
	if match == nil {
		return 0, "", &module.Error{Message: env.Text("module-tagcloud-font-size", "size", size)}
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, "", &module.Error{Message: env.Text("module-tagcloud-font-size", "size", size)}
	}
	return value, strings.ToLower(strings.TrimSpace(match[2])), nil
}

var rgbPattern = regexp.MustCompile(`^rgb\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$`)
var bareRGBPattern = regexp.MustCompile(`^(\d+)\s*,\s*(\d+)\s*,\s*(\d+)$`)

func parseTagColor(env module.Env, color string) ([3]int, error) {
	bad := &module.Error{Message: env.Text("module-tagcloud-color", "color", color)}
	if strings.HasPrefix(color, "#") {
		return parseHexColor(color[1:], bad)
	}
	if strings.HasPrefix(strings.TrimLeft(color, " \t\n\v\f\r"), "rgb") {
		return matchRGB(rgbPattern, strings.TrimSpace(color), bad)
	}
	return matchRGB(bareRGBPattern, strings.TrimSpace(color), bad)
}

func parseHexColor(digits string, bad error) ([3]int, error) {
	if len(digits) == 3 {
		digits = string([]byte{digits[0], digits[0], digits[1], digits[1], digits[2], digits[2]})
	}
	if len(digits) != 6 {
		return [3]int{}, bad
	}
	var out [3]int
	for i := range out {
		value, err := strconv.ParseUint(digits[2*i:2*i+2], 16, 16)
		if err != nil {
			return [3]int{}, bad
		}
		out[i] = int(value)
	}
	return out, nil
}

func matchRGB(pattern *regexp.Regexp, color string, bad error) ([3]int, error) {
	match := pattern.FindStringSubmatch(color)
	if match == nil {
		return [3]int{}, bad
	}
	var out [3]int
	for i := range out {
		value, err := strconv.Atoi(match[i+1])
		if err != nil {
			return [3]int{}, bad
		}
		out[i] = value
	}
	return out, nil
}
