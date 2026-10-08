package main

import (
	"errors"
	"strings"
	"testing"
)

func TestTickFailureKeepsTheLastLine(t *testing.T) {
	out := []byte("starting\npwikit: read /srv/pwikit/pwikit.toml: permission denied\n")
	if got, want := tickFailure(errors.New("exit status 1"), out), "pwikit: read /srv/pwikit/pwikit.toml: permission denied"; got != want {
		t.Errorf("tickFailure() = %q, want %q", got, want)
	}
}

func TestTickFailureFallsBackToTheError(t *testing.T) {
	if got := tickFailure(errors.New("exit status 1"), nil); got != "exit status 1" {
		t.Errorf("tickFailure() = %q, want %q", got, "exit status 1")
	}
}

func TestTickFailureIsCut(t *testing.T) {
	if got := tickFailure(errors.New("x"), []byte(strings.Repeat("a", 1000))); len(got) != tickFailureMax {
		t.Errorf("len(tickFailure()) = %d, want %d", len(got), tickFailureMax)
	}
}
