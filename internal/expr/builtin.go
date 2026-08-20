package expr

import (
	"math"
	"math/rand/v2"
	"strings"
)

var brokenTrig = map[string]bool{
	"sin": true, "cos": true, "tan": true,
	"asin": true, "acos": true, "atan": true,
}

func evalCall(n *callNode) (Value, error) {
	args := make([]Value, 0, len(n.args))
	for _, a := range n.args {
		v, err := eval(a)
		if err != nil {
			return None(), err
		}
		args = append(args, v)
	}

	if brokenTrig[n.name] {
		return None(), errType
	}

	switch n.name {
	case "min", "max":
		return extremum(n.name, args)
	case "abs":
		return absOf(args)
	case "round":
		return roundOf(args)
	case "ceil", "floor":
		return ceilFloor(n.name, args)
	case "div":
		return floorDiv(args)
	case "random":
		return randomOf(args)
	case "sqrt":
		return sqrtOf(args)
	case "pow":
		return powOf(args)
	case "unset":
		return unsetOf(args)
	case "len":
		return lenOf(args)
	case "lower", "upper":
		return changeCase(n.name, args)
	case "substr":
		return substrOf(args)
	}
	return None(), errType
}

func arity(args []Value, allowed ...int) error {
	for _, n := range allowed {
		if len(args) == n {
			return nil
		}
	}
	return errType
}

func extremum(name string, args []Value) (Value, error) {
	if len(args) < 2 {
		return None(), errType
	}
	best := args[0]
	for _, v := range args[1:] {
		cmp, err := order(v, best)
		if err != nil {
			return None(), err
		}
		if (name == "min" && cmp < 0) || (name == "max" && cmp > 0) {
			best = v
		}
	}
	return best, nil
}

func absOf(args []Value) (Value, error) {
	if err := arity(args, 1); err != nil {
		return None(), err
	}
	x := args[0]
	if !x.numeric() {
		return None(), errType
	}
	if x.Kind == KindFloat {
		return FloatOf(math.Abs(x.Float)), nil
	}
	if n := x.toInt(); n < 0 {
		return IntOf(-n), nil
	}
	return IntOf(x.toInt()), nil
}

func roundOf(args []Value) (Value, error) {
	if err := arity(args, 1, 2); err != nil {
		return None(), err
	}
	x := args[0]
	if !x.numeric() {
		return None(), errType
	}
	var digits int64
	if len(args) == 2 {
		if !args[1].integral() {
			return None(), errType
		}
		digits = args[1].toInt()
	}

	if x.integral() {
		if digits >= 0 {
			return IntOf(x.toInt()), nil
		}
		shift := math.Pow(10, float64(-digits))
		return IntOf(int64(math.RoundToEven(float64(x.toInt())/shift) * shift)), nil
	}
	shift := math.Pow(10, float64(digits))
	return FloatOf(math.RoundToEven(x.Float*shift) / shift), nil
}

func ceilFloor(name string, args []Value) (Value, error) {
	if err := arity(args, 1); err != nil {
		return None(), err
	}
	x := args[0]
	if !x.numeric() {
		return None(), errType
	}
	if x.integral() {
		return IntOf(x.toInt()), nil
	}
	if name == "ceil" {
		return IntOf(int64(math.Ceil(x.Float))), nil
	}
	return IntOf(int64(math.Floor(x.Float))), nil
}

func floorDiv(args []Value) (Value, error) {
	if err := arity(args, 2); err != nil {
		return None(), err
	}
	x, y := args[0], args[1]
	if !x.numeric() || !y.numeric() {
		return None(), errType
	}
	if y.toFloat() == 0 {
		return None(), errType
	}
	if x.integral() && y.integral() {
		a, b := x.toInt(), y.toInt()
		q := a / b
		if (a%b != 0) && ((a < 0) != (b < 0)) {
			q--
		}
		return IntOf(q), nil
	}
	return FloatOf(math.Floor(x.toFloat() / y.toFloat())), nil
}

func randomOf(args []Value) (Value, error) {
	if err := arity(args, 2); err != nil {
		return None(), err
	}
	if !args[0].integral() || !args[1].integral() {
		return None(), errType
	}
	low, high := args[0].toInt(), args[1].toInt()
	if low > high {
		return None(), errType
	}
	return IntOf(low + rand.Int64N(high-low+1)), nil
}

func sqrtOf(args []Value) (Value, error) {
	if err := arity(args, 1); err != nil {
		return None(), err
	}
	if !args[0].numeric() {
		return None(), errType
	}
	f := args[0].toFloat()
	if f < 0 {
		return None(), errType
	}
	return FloatOf(math.Sqrt(f)), nil
}

func powOf(args []Value) (Value, error) {
	if err := arity(args, 2); err != nil {
		return None(), err
	}
	if !args[0].numeric() || !args[1].numeric() {
		return None(), errType
	}
	return FloatOf(math.Pow(args[0].toFloat(), args[1].toFloat())), nil
}

func unsetOf(args []Value) (Value, error) {
	if err := arity(args, 1); err != nil {
		return None(), err
	}
	s := args[0].PyStr()
	return BoolOf(strings.HasPrefix(s, "%%") && strings.HasSuffix(s, "%%")), nil
}

func lenOf(args []Value) (Value, error) {
	if err := arity(args, 1); err != nil {
		return None(), err
	}
	if args[0].Kind != KindStr {
		return None(), errType
	}
	return IntOf(int64(len([]rune(args[0].Str)))), nil
}

func changeCase(name string, args []Value) (Value, error) {
	if err := arity(args, 1); err != nil {
		return None(), err
	}
	if args[0].Kind != KindStr {
		return None(), errType
	}
	if name == "lower" {
		return StrOf(strings.ToLower(args[0].Str)), nil
	}
	return StrOf(strings.ToUpper(args[0].Str)), nil
}

func substrOf(args []Value) (Value, error) {
	if err := arity(args, 2, 3); err != nil {
		return None(), err
	}
	if args[0].Kind != KindStr || !args[1].integral() {
		return None(), errType
	}
	runes := []rune(args[0].Str)
	start := clampIndex(args[1].toInt(), len(runes))
	stop := len(runes)
	if len(args) == 3 {
		if !args[2].integral() {
			return None(), errType
		}
		stop = clampIndex(args[2].toInt(), len(runes))
	}
	if stop <= start {
		return StrOf(""), nil
	}
	return StrOf(string(runes[start:stop])), nil
}

func clampIndex(i int64, n int) int {
	if i < 0 {
		i += int64(n)
		if i < 0 {
			return 0
		}
	}
	if i > int64(n) {
		return n
	}
	return int(i)
}
