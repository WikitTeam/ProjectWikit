package expr

import (
	"fmt"
	"strings"
)

func Evaluate(src string) Value {
	n, err := parse(src)
	if err != nil {
		return None()
	}
	v, err := eval(n)
	if err != nil {
		return None()
	}
	return v
}

func eval(n node) (Value, error) {
	switch n := n.(type) {
	case *constNode:
		return n.v, nil
	case *unaryNode:
		return evalUnary(n)
	case *binNode:
		return evalBinary(n)
	case *cmpNode:
		return evalCompare(n)
	case *boolNode:
		return evalBool(n)
	case *callNode:
		return evalCall(n)
	}
	return None(), fmt.Errorf("%w: 未知节点", errType)
}

func evalUnary(n *unaryNode) (Value, error) {
	if n.op != "-" {
		return None(), fmt.Errorf("%w: 一元运算 %q", errType, n.op)
	}
	x, err := eval(n.x)
	if err != nil {
		return None(), err
	}
	if !x.numeric() {
		return None(), errType
	}
	if x.Kind == KindFloat {
		return FloatOf(-x.Float), nil
	}
	return IntOf(-x.toInt()), nil
}

func evalBinary(n *binNode) (Value, error) {
	x, err := eval(n.x)
	if err != nil {
		return None(), err
	}
	y, err := eval(n.y)
	if err != nil {
		return None(), err
	}

	switch n.op {
	case "+":
		if x.Kind == KindStr && y.Kind == KindStr {
			return StrOf(x.Str + y.Str), nil
		}
		return arith(x, y, func(a, b int64) int64 { return a + b }, func(a, b float64) float64 { return a + b })
	case "-":
		return arith(x, y, func(a, b int64) int64 { return a - b }, func(a, b float64) float64 { return a - b })
	case "*":
		if repeated, ok := repeat(x, y); ok {
			return repeated, nil
		}
		return arith(x, y, func(a, b int64) int64 { return a * b }, func(a, b float64) float64 { return a * b })
	case "/":
		if !x.numeric() || !y.numeric() {
			return None(), errType
		}
		if y.toFloat() == 0 {
			return None(), fmt.Errorf("%w: 除以零", errType)
		}
		return FloatOf(x.toFloat() / y.toFloat()), nil
	case "^":
		if !x.integral() || !y.integral() {
			return None(), errType
		}
		if x.Kind == KindBool && y.Kind == KindBool {
			return BoolOf(x.Bool != y.Bool), nil
		}
		return IntOf(x.toInt() ^ y.toInt()), nil
	}
	return None(), fmt.Errorf("%w: 二元运算 %q", errType, n.op)
}

func arith(x, y Value, ints func(a, b int64) int64, floats func(a, b float64) float64) (Value, error) {
	if !x.numeric() || !y.numeric() {
		return None(), errType
	}
	if x.Kind == KindFloat || y.Kind == KindFloat {
		return FloatOf(floats(x.toFloat(), y.toFloat())), nil
	}
	return IntOf(ints(x.toInt(), y.toInt())), nil
}

func repeat(x, y Value) (Value, bool) {
	str, count := x, y
	if str.Kind != KindStr {
		str, count = y, x
	}
	if str.Kind != KindStr || !count.integral() {
		return None(), false
	}
	n := count.toInt()
	if n <= 0 {
		return StrOf(""), true
	}
	return StrOf(strings.Repeat(str.Str, int(n))), true
}

func evalCompare(n *cmpNode) (Value, error) {
	for i, op := range n.ops {
		left, err := eval(n.items[i])
		if err != nil {
			return None(), err
		}
		right, err := eval(n.items[i+1])
		if err != nil {
			return None(), err
		}
		ok, err := compare(left, op, right)
		if err != nil {
			return None(), err
		}
		if !ok {
			return BoolOf(false), nil
		}
	}
	return BoolOf(true), nil
}

func compare(x Value, op string, y Value) (bool, error) {
	switch op {
	case "==":
		return equal(x, y), nil
	case "!=":
		return !equal(x, y), nil
	}
	cmp, err := order(x, y)
	if err != nil {
		return false, err
	}
	switch op {
	case "<":
		return cmp < 0, nil
	case "<=":
		return cmp <= 0, nil
	case ">":
		return cmp > 0, nil
	case ">=":
		return cmp >= 0, nil
	}
	return false, fmt.Errorf("%w: 比较运算 %q", errType, op)
}

func equal(x, y Value) bool {
	switch {
	case x.integral() && y.integral():
		return x.toInt() == y.toInt()
	case x.numeric() && y.numeric():
		return x.toFloat() == y.toFloat()
	case x.Kind == KindStr && y.Kind == KindStr:
		return x.Str == y.Str
	case x.Kind == KindNone && y.Kind == KindNone:
		return true
	}
	return false
}

func order(x, y Value) (int, error) {
	switch {
	case x.integral() && y.integral():
		return cmpInt(x.toInt(), y.toInt()), nil
	case x.numeric() && y.numeric():
		return cmpFloat(x.toFloat(), y.toFloat()), nil
	case x.Kind == KindStr && y.Kind == KindStr:
		return strings.Compare(x.Str, y.Str), nil
	}
	return 0, errType
}

func cmpInt(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func evalBool(n *boolNode) (Value, error) {
	truthy := 0
	for _, item := range n.items {
		v, err := eval(item)
		if err != nil {
			return None(), err
		}
		if v.Truthy() {
			truthy++
		}
	}
	if n.op == "and" {
		return BoolOf(truthy == len(n.items)), nil
	}
	return BoolOf(truthy > 0), nil
}
