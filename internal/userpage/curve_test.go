package userpage

import (
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
