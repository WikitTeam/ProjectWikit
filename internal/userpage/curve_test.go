package userpage

import (
	"strconv"
	"strings"
	"testing"
)

func TestCurveSinglePoint(t *testing.T) {
	if got := curve([][2]float64{{1, 2}}); got != "M1.0,2.0" {
		t.Errorf("curve(one point) = %q, want %q", got, "M1.0,2.0")
	}
}

func TestCurveEndsOnEveryPoint(t *testing.T) {
	got := curve([][2]float64{{0, 10}, {10, 20}, {20, 5}})
	for _, end := range []string{" 10.0,20.0 C", " 20.0,5.0"} {
		if !strings.Contains(got, end) {
			t.Errorf("curve() = %q, want it to contain %q", got, end)
		}
	}
	if n := strings.Count(got, "C"); n != 2 {
		t.Errorf("curve() segments = %d, want 2", n)
	}
}

func TestCurveFlatAtTurningPoint(t *testing.T) {
	got := curve([][2]float64{{0, 10}, {30, 40}, {60, 10}})
	if !strings.Contains(got, "20.0,40.0 30.0,40.0 C40.0,40.0") {
		t.Errorf("curve() = %q, want flat handles at the peak", got)
	}
}

func TestCurveDropsRepeatedX(t *testing.T) {
	got := curve([][2]float64{{0, 10}, {10, 20}, {10, 30}, {20, 30}})
	if strings.Contains(got, "10.0,20.0") {
		t.Errorf("curve() = %q, want the earlier point at x=10 dropped", got)
	}
	if strings.Contains(got, "NaN") || strings.Contains(got, "Inf") {
		t.Errorf("curve() = %q, want finite numbers", got)
	}
}

func TestCurveStaysBetweenItsPoints(t *testing.T) {
	pts := [][2]float64{{0, 96}, {100, 85}, {100.05, 20}, {100.1, 80}, {400, 50}, {400.02, 10}, {600, 5}}
	got := curve(pts)
	for _, field := range strings.FieldsFunc(got, func(r rune) bool { return r == ' ' || r == 'M' || r == 'C' }) {
		_, y, ok := strings.Cut(field, ",")
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(y, 64)
		if err != nil {
			t.Fatalf("curve() = %q, has unreadable y %q", got, y)
		}
		if v < 5 || v > 96 {
			t.Errorf("curve() y = %v, want within 5..96", v)
		}
	}
}
