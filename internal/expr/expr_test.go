package expr

import "testing"

func check(t *testing.T, src string, want Value) {
	t.Helper()
	got := Evaluate(src)
	if got != want {
		t.Errorf("Evaluate(%q) = %+v, want %+v", src, got, want)
	}
}

func TestEvaluateLiterals(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"1", IntOf(1)},
		{"-1", IntOf(-1)},
		{"1.5", FloatOf(1.5)},
		{".5", FloatOf(0.5)},
		{"1e3", FloatOf(1000)},
		{"1e-3", FloatOf(0.001)},
		{"'abc'", StrOf("abc")},
		{`"abc"`, StrOf("abc")},
		{`'a\nb'`, StrOf("a\nb")},
		{"True", BoolOf(true)},
		{"False", BoolOf(false)},
		{"None", None()},
		{"  1  ", IntOf(1)},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateArithmetic(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"1 + 2", IntOf(3)},
		{"5 - 8", IntOf(-3)},
		{"3 * 4", IntOf(12)},
		{"1 + 2 * 3", IntOf(7)},
		{"(1 + 2) * 3", IntOf(9)},
		{"1.5 + 1", FloatOf(2.5)},
		{"True + True", IntOf(2)},
		{"-True", IntOf(-1)},
		{"'a' + 'b'", StrOf("ab")},
		{"'ab' * 3", StrOf("ababab")},
		{"3 * 'ab'", StrOf("ababab")},
		{"'ab' * 0", StrOf("")},
		{"'a' + 1", None()},
		{"-'a'", None()},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateDivisionIsAlwaysFloat(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"6 / 3", FloatOf(2)},
		{"7 / 2", FloatOf(3.5)},
		{"1 / 0", None()},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateCaretIsXorNotPower(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"2 ^ 3", IntOf(1)},
		{"1 ^ 2 + 3", IntOf(4)},
		{"True ^ False", BoolOf(true)},
		{"True ^ 1", IntOf(0)},
		{"1.0 ^ 2", None()},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateComparison(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"1 == 1", BoolOf(true)},
		{"1 != 2", BoolOf(true)},
		{"1 < 2 < 3", BoolOf(true)},
		{"1 < 2 > 5", BoolOf(false)},
		{"'a' < 'b'", BoolOf(true)},
		{"1 == True", BoolOf(true)},
		{"1 == '1'", BoolOf(false)},
		{"None == None", BoolOf(true)},
		{"None == 0", BoolOf(false)},
		{"1 < 'a'", None()},
		{"1 <= 1.0", BoolOf(true)},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateBoolOpsReturnBoolNotOperand(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"1 and 2", BoolOf(true)},
		{"0 and 2", BoolOf(false)},
		{"0 or 2", BoolOf(true)},
		{"0 or ''", BoolOf(false)},
		{"'x' or 0", BoolOf(true)},
		{"1 or 0 and 0", BoolOf(true)},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateUnsupportedUnary(t *testing.T) {
	for _, src := range []string{"not True", "+1", "not 0"} {
		t.Run(src, func(t *testing.T) { check(t, src, None()) })
	}
}

func TestEvaluateNumericBuiltins(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"min(3, 1, 2)", IntOf(1)},
		{"max(3, 1, 2)", IntOf(3)},
		{"min(5)", None()},
		{"min(1, 'a')", None()},
		{"abs(-3)", IntOf(3)},
		{"abs(-3.5)", FloatOf(3.5)},
		{"abs('a')", None()},
		{"ceil(1.2)", IntOf(2)},
		{"floor(1.8)", IntOf(1)},
		{"ceil(2)", IntOf(2)},
		{"div(7, 2)", IntOf(3)},
		{"div(-7, 2)", IntOf(-4)},
		{"div(7, 0)", None()},
		{"sqrt(9)", FloatOf(3)},
		{"sqrt(-1)", None()},
		{"pow(2, 3)", FloatOf(8)},
		{"pow(2, 0.5)", FloatOf(1.4142135623730951)},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateRoundUsesBankersRoundingAndReturnsFloat(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"round(2.5)", FloatOf(2)},
		{"round(3.5)", FloatOf(4)},
		{"round(-2.5)", FloatOf(-2)},
		{"round(2.34, 1)", FloatOf(2.3)},
		{"round(7)", IntOf(7)},
		{"round('a')", None()},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateTrigAlwaysFails(t *testing.T) {
	for _, src := range []string{"sin(0)", "cos(0)", "tan(0)", "asin(0)", "acos(1)", "atan(0)"} {
		t.Run(src, func(t *testing.T) { check(t, src, None()) })
	}
}

func TestEvaluateStringBuiltins(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"len('abc')", IntOf(3)},
		{"len('中文')", IntOf(2)},
		{"len(3)", None()},
		{"lower('ABC')", StrOf("abc")},
		{"upper('abc')", StrOf("ABC")},
		{"upper(3)", None()},
		{"substr('abcdef', 1, 3)", StrOf("bc")},
		{"substr('abcdef', 2)", StrOf("cdef")},
		{"substr('abcdef', -2)", StrOf("ef")},
		{"substr('abcdef', 4, 2)", StrOf("")},
		{"substr('abcdef', 0, 99)", StrOf("abcdef")},
		{"substr('中文字', 1, 2)", StrOf("文")},
		{"substr(3, 1)", None()},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateUnset(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"unset('%%title%%')", BoolOf(true)},
		{"unset('title')", BoolOf(false)},
		{"unset('%%')", BoolOf(true)},
		{"unset(1)", BoolOf(false)},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) { check(t, tt.src, tt.want) })
	}
}

func TestEvaluateFunctionNamesAreCaseInsensitive(t *testing.T) {
	check(t, "MIN(1, 2)", IntOf(1))
	check(t, "Upper('a')", StrOf("A"))
}

func TestEvaluateConstantNamesAreCaseSensitive(t *testing.T) {
	for _, src := range []string{"true", "false", "none"} {
		t.Run(src, func(t *testing.T) { check(t, src, None()) })
	}
}

func TestEvaluateMalformedInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"1 +",
		"1 2",
		"(1",
		"1)",
		"foo",
		"foo(1)",
		"1 % 2",
		"7 // 2",
		"'unterminated",
		"1, 2",
		"min(1,)",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) { check(t, src, None()) })
	}
}

func TestEvaluateRandomStaysInRange(t *testing.T) {
	for range 50 {
		got := Evaluate("random(1, 3)")
		if got.Kind != KindInt || got.Int < 1 || got.Int > 3 {
			t.Fatalf("Evaluate(\"random(1, 3)\") = %+v, want an int in 1..3", got)
		}
	}
	check(t, "random(3, 1)", None())
}

func TestText(t *testing.T) {
	tests := []struct {
		in   Value
		want string
	}{
		{None(), "None"},
		{BoolOf(true), "True"},
		{BoolOf(false), "False"},
		{IntOf(-7), "-7"},
		{FloatOf(1), "1.0"},
		{FloatOf(1.5), "1.5"},
		{StrOf("x"), "x"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.in.Text(); got != tt.want {
				t.Errorf("Text(%+v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruthy(t *testing.T) {
	tests := []struct {
		in   Value
		want bool
	}{
		{None(), false},
		{BoolOf(false), false},
		{IntOf(0), false},
		{IntOf(1), true},
		{FloatOf(0), false},
		{StrOf(""), false},
		{StrOf("0"), true},
	}
	for _, tt := range tests {
		t.Run(tt.in.Text(), func(t *testing.T) {
			if got := tt.in.Truthy(); got != tt.want {
				t.Errorf("Truthy(%+v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
