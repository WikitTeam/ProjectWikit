package pynum

import "testing"

func TestInt(t *testing.T) {
	good := map[string]int{"5": 5, " 5 ": 5, "+5": 5, "-5": -5, "1_000": 1000, "0": 0}
	for in, want := range good {
		got, err := Int(in)
		if err != nil || got != want {
			t.Errorf("Int(%q) = %d, %v, want %d, nil", in, got, err, want)
		}
	}
	for _, in := range []string{"", " ", "_5", "5_", "1__0", "5.0", "0x10", "５", "+"} {
		if got, err := Int(in); err == nil {
			t.Errorf("Int(%q) = %d, nil, want an error", in, got)
		}
	}
}

func TestFloatRejectsWhatPythonRejects(t *testing.T) {
	for _, in := range []string{"0x1p-2", "", "abc"} {
		if got, err := Float(in); err == nil {
			t.Errorf("Float(%q) = %v, nil, want an error", in, got)
		}
	}
	if got, err := Float("3.5"); err != nil || got != 3.5 {
		t.Errorf("Float(3.5) = %v, %v, want 3.5, nil", got, err)
	}
}
