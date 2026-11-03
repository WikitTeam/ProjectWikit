package quiz

import (
	"testing"
	"time"
)

const sample = `云分是否允许自我评分？
- 允许
+ 功能上不允许 站规也不允许

以下哪些行为会被封禁？（多选）
+ 恶意破坏
- 撰写礼貌的差评
+ 辱骂

没有正确答案的题
- 甲
- 乙
`

func TestParse(t *testing.T) {
	got := Parse(sample)
	if len(got) != 2 {
		t.Fatalf("len(Parse()) = %d, want 2", len(got))
	}
	if got[0].Text != "云分是否允许自我评分？" || len(got[0].Options) != 2 || got[0].Multiple() {
		t.Errorf("Parse()[0] = %+v, want a two-option single choice", got[0])
	}
	if !got[1].Multiple() || len(got[1].Options) != 3 {
		t.Errorf("Parse()[1] = %+v, want a three-option multiple choice", got[1])
	}
}

func TestPassed(t *testing.T) {
	qs := Parse(sample)
	if !Passed(qs, map[int][]int{0: {1}, 1: {0, 2}}) {
		t.Error("Passed(all right) = false, want true")
	}
	if Passed(qs, map[int][]int{0: {1}, 1: {0}}) {
		t.Error("Passed(one missing in a multiple choice) = true, want false")
	}
	if Passed(qs, map[int][]int{0: {1}, 1: {0, 1, 2}}) {
		t.Error("Passed(one extra in a multiple choice) = true, want false")
	}
	if Passed(qs, map[int][]int{0: {1, 7}, 1: {0, 2}}) {
		t.Error("Passed(an out of range option) = true, want false")
	}
	if Passed(nil, nil) {
		t.Error("Passed(no questions) = true, want false")
	}
}

func TestState(t *testing.T) {
	base := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	fail := func(minutes int) Attempt { return Attempt{At: base.Add(time.Duration(minutes) * time.Minute)} }
	cool := 12 * time.Hour

	if left, until := State(nil, 3, cool, base); left != 3 || !until.IsZero() {
		t.Errorf("State(none) = %d, %v, want 3, zero", left, until)
	}
	if left, _ := State([]Attempt{fail(0), fail(1)}, 3, cool, base.Add(time.Hour)); left != 1 {
		t.Errorf("State(two failures) left = %d, want 1", left)
	}
	three := []Attempt{fail(0), fail(1), fail(2)}
	if left, until := State(three, 3, cool, base.Add(time.Hour)); left != 0 || !until.Equal(base.Add(2*time.Minute+cool)) {
		t.Errorf("State(three failures, an hour on) = %d, %v, want 0, %v", left, until, base.Add(2*time.Minute+cool))
	}
	if left, until := State(three, 3, cool, base.Add(13*time.Hour)); left != 3 || !until.IsZero() {
		t.Errorf("State(three failures, after the cooldown) = %d, %v, want 3, zero", left, until)
	}
	again := append(three, fail(13*60))
	if left, _ := State(again, 3, cool, base.Add(14*time.Hour)); left != 2 {
		t.Errorf("State(a failure after the cooldown) left = %d, want 2", left)
	}
}
