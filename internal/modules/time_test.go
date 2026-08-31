package modules

import (
	"regexp"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func timeEnv(t *testing.T) module.Env {
	t.Helper()
	env := forumEnv(t)
	env.Render = func(source string, _ *page.Context) (string, error) { return source, nil }
	return env
}

var dateSpanPattern = regexp.MustCompile(`<span class="odate w-date" data-timestamp="(\d+)" data-format="([^"]+)" style="display: inline">([^<]*)</span>`)

func TestTimeSubstitutesEveryField(t *testing.T) {
	env := timeEnv(t)
	cases := []struct{ in, want string }{
		{"%%currentyear%%", "%Y"},
		{"%%currentmonth%%", "%m"},
		{"%%currentday%%", "%d"},
		{"%%currenthour%%", "%H"},
		{"%%currentminute%%", "%M"},
		{"%%currentsecond%%", "%S"},
	}
	for _, c := range cases {
		got, err := renderTime(env, nil, c.in)
		if err != nil {
			t.Fatalf("renderTime(%q) err = %v, want nil", c.in, err)
		}
		m := dateSpanPattern.FindStringSubmatch(got)
		if m == nil {
			t.Errorf("renderTime(%q) = %q, want a date span", c.in, got)
			continue
		}
		if m[2] != c.want {
			t.Errorf("renderTime(%q) format = %q, want %q", c.in, m[2], c.want)
		}
		if m[3] == "" {
			t.Errorf("renderTime(%q) text = %q, want the field spelled out", c.in, m[3])
		}
	}
}

func TestTimeYearTextIsFourDigits(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%currentyear%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	m := dateSpanPattern.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("renderTime(%%%%currentyear%%%%) = %q, want a date span", got)
	}
	if len(m[3]) != 4 {
		t.Errorf("renderTime(%%%%currentyear%%%%) text = %q, want four digits", m[3])
	}
}

func TestTimeMonthTextIsPaddedToTwo(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%currentmonth%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	m := dateSpanPattern.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("renderTime(%%%%currentmonth%%%%) = %q, want a date span", got)
	}
	if len(m[3]) != 2 {
		t.Errorf("renderTime(%%%%currentmonth%%%%) text = %q, want two digits", m[3])
	}
}

func TestTimeAcceptsTheCurentSpelling(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%curentyear%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	m := dateSpanPattern.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("renderTime(%%%%curentyear%%%%) = %q, want a date span", got)
	}
	if m[2] != "%Y" {
		t.Errorf("renderTime(%%%%curentyear%%%%) format = %q, want %q", m[2], "%Y")
	}
}

func TestTimeKeepsTheTextAroundTheFields(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%currentyear%%.%%currentmonth%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	if n := len(dateSpanPattern.FindAllString(got, -1)); n != 2 {
		t.Errorf("renderTime(%q) has %d date spans, want 2", "%%currentyear%%.%%currentmonth%%", n)
	}
	if !strings.Contains(got, "</span>.<span") {
		t.Errorf("renderTime(%q) = %q, want the dot kept between the spans", "%%currentyear%%.%%currentmonth%%", got)
	}
}

func TestTimeLeavesAnUnknownVariableAlone(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%currentcentury%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	if want := "%%currentcentury%%"; got != want {
		t.Errorf("renderTime(%q) = %q, want %q", want, got, want)
	}
}

func TestTimeUsesOneStampForEveryField(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%currentminute%%%%currentsecond%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	stamps := dateSpanPattern.FindAllStringSubmatch(got, -1)
	if len(stamps) != 2 {
		t.Fatalf("renderTime() has %d date spans, want 2", len(stamps))
	}
	if stamps[0][1] != stamps[1][1] {
		t.Errorf("renderTime() stamps = %q and %q, want them equal", stamps[0][1], stamps[1][1])
	}
}

func TestTimeCarriesNoHoverClass(t *testing.T) {
	env := timeEnv(t)

	got, err := renderTime(env, nil, "%%currentyear%%")
	if err != nil {
		t.Fatalf("renderTime() err = %v, want nil", err)
	}
	if strings.Contains(got, "w-odate-hover") {
		t.Errorf("renderTime() = %q, want no hover class", got)
	}
}
