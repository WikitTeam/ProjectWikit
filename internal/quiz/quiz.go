// Package quiz reads a membership quiz, grades an attempt at it and works out
// how many tries a member has left.
package quiz

import (
	"slices"
	"strings"
	"time"
)

type Option struct {
	Text    string
	Correct bool
}

type Question struct {
	Text    string
	Options []Option
}

func (q Question) Multiple() bool {
	n := 0
	for _, o := range q.Options {
		if o.Correct {
			n++
		}
	}
	return n > 1
}

// Lines starting with + are right answers and lines starting with - wrong ones.
// A block without both is dropped.
func Parse(raw string) []Question {
	var out []Question
	var current *Question
	flush := func() {
		if current != nil && current.Text != "" && len(current.Options) > 1 &&
			slices.ContainsFunc(current.Options, func(o Option) bool { return o.Correct }) {
			out = append(out, *current)
		}
		current = nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		if current == nil {
			current = &Question{}
		}
		switch {
		case strings.HasPrefix(line, "+"):
			current.Options = append(current.Options, Option{Text: strings.TrimSpace(line[1:]), Correct: true})
		case strings.HasPrefix(line, "-"):
			current.Options = append(current.Options, Option{Text: strings.TrimSpace(line[1:])})
		case len(current.Options) == 0:
			current.Text = strings.TrimSpace(current.Text + " " + line)
		}
	}
	flush()
	return out
}

// Passed needs every question answered exactly. Answers are option indexes as
// the question lists them, keyed by question index.
func Passed(questions []Question, answers map[int][]int) bool {
	if len(questions) == 0 {
		return false
	}
	for i, q := range questions {
		picked := answers[i]
		for j, o := range q.Options {
			if o.Correct != slices.Contains(picked, j) {
				return false
			}
		}
		for _, j := range picked {
			if j < 0 || j >= len(q.Options) {
				return false
			}
		}
	}
	return true
}

type Attempt struct {
	At     time.Time
	Passed bool
}

// Running out of tries locks the quiz until the cooldown has passed, after
// which the count starts over.
func State(attempts []Attempt, tries int, cooldown time.Duration, now time.Time) (left int, lockedUntil time.Time) {
	tries = max(tries, 1)
	failed := 0
	var until time.Time
	for _, a := range attempts {
		if a.Passed {
			continue
		}
		if !until.IsZero() && !a.At.Before(until) {
			failed, until = 0, time.Time{}
		}
		failed++
		if failed >= tries {
			until = a.At.Add(cooldown)
		}
	}
	if !until.IsZero() && now.Before(until) {
		return 0, until
	}
	if !until.IsZero() {
		failed = 0
	}
	return max(tries-failed, 0), time.Time{}
}

func Hours(n int) time.Duration { return time.Duration(max(n, 0)) * time.Hour }
