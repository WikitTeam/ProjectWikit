package expr

import (
	"errors"
	"math"
	"strconv"
)

type Kind uint8

const (
	KindNone Kind = iota
	KindBool
	KindInt
	KindFloat
	KindStr
)

type Value struct {
	Kind  Kind
	Bool  bool
	Int   int64
	Float float64
	Str   string
}

var errType = errors.New("expr: unsupported operand types")

func None() Value             { return Value{Kind: KindNone} }
func BoolOf(b bool) Value     { return Value{Kind: KindBool, Bool: b} }
func IntOf(i int64) Value     { return Value{Kind: KindInt, Int: i} }
func FloatOf(f float64) Value { return Value{Kind: KindFloat, Float: f} }
func StrOf(s string) Value    { return Value{Kind: KindStr, Str: s} }

func (v Value) numeric() bool {
	return v.Kind == KindBool || v.Kind == KindInt || v.Kind == KindFloat
}

func (v Value) integral() bool {
	return v.Kind == KindBool || v.Kind == KindInt
}

func (v Value) toInt() int64 {
	switch v.Kind {
	case KindBool:
		if v.Bool {
			return 1
		}
		return 0
	case KindInt:
		return v.Int
	}
	return 0
}

func (v Value) toFloat() float64 {
	if v.Kind == KindFloat {
		return v.Float
	}
	return float64(v.toInt())
}

func (v Value) Truthy() bool {
	switch v.Kind {
	case KindBool:
		return v.Bool
	case KindInt:
		return v.Int != 0
	case KindFloat:
		return v.Float != 0
	case KindStr:
		return v.Str != ""
	}
	return false
}

func (v Value) PyStr() string {
	switch v.Kind {
	case KindBool:
		if v.Bool {
			return "True"
		}
		return "False"
	case KindInt:
		return strconv.FormatInt(v.Int, 10)
	case KindFloat:
		if math.IsInf(v.Float, 1) {
			return "inf"
		}
		if math.IsInf(v.Float, -1) {
			return "-inf"
		}
		if math.IsNaN(v.Float) {
			return "nan"
		}
		if v.Float == math.Trunc(v.Float) && math.Abs(v.Float) < 1e16 {
			return strconv.FormatFloat(v.Float, 'f', 1, 64)
		}
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case KindStr:
		return v.Str
	}
	return "None"
}

func (v Value) AsInt() int64 { return v.toInt() }
