package timezone

import (
	"testing"
	"time"
)

func TestLoadResolvesANamedZone(t *testing.T) {
	if got := Load("Asia/Shanghai").String(); got != "Asia/Shanghai" {
		t.Errorf("Load(%q) = %q, want %q", "Asia/Shanghai", got, "Asia/Shanghai")
	}
}

func TestLoadFallsBackToUTC(t *testing.T) {
	for _, name := range []string{"", "Local", "Mars/Olympus", "../etc/passwd"} {
		if got := Load(name); got != time.UTC {
			t.Errorf("Load(%q) = %v, want UTC", name, got)
		}
	}
}

func TestValid(t *testing.T) {
	for _, name := range []string{"UTC", "Asia/Shanghai", "America/New_York", "Europe/Moscow"} {
		if !Valid(name) {
			t.Errorf("Valid(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "Local", "Beijing", "Mars/Olympus"} {
		if Valid(name) {
			t.Errorf("Valid(%q) = true, want false", name)
		}
	}
}

func TestLabel(t *testing.T) {
	at := time.Date(2026, 9, 11, 6, 30, 0, 0, time.UTC)
	cases := map[string]string{
		"UTC":              "UTC",
		"Asia/Shanghai":    "UTC+08:00",
		"Asia/Kolkata":     "UTC+05:30",
		"America/New_York": "UTC-04:00",
	}
	for name, want := range cases {
		if got := Label(at.In(Load(name))); got != want {
			t.Errorf("Label(at in %s) = %q, want %q", name, got, want)
		}
	}
}
