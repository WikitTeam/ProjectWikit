// Package pyjson spells values the way Python's json.dumps does.
package pyjson

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Field is one key of an object. Objects are slices rather than maps because
// Python writes a dict in insertion order and these are compared byte for byte.
type Field struct {
	Key   string
	Value any
}

type Object []Field

type Array []any

// Marshal writes v with Python's separators, which carry a space that
// encoding/json never writes.
func Marshal(v any) (string, error) {
	var b strings.Builder
	if err := write(&b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

func write(b *strings.Builder, v any) error {
	switch value := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if value {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		b.WriteString(String(value))
	case int:
		b.WriteString(strconv.Itoa(value))
	case int64:
		b.WriteString(strconv.FormatInt(value, 10))
	case float64:
		s, err := Float(value)
		if err != nil {
			return err
		}
		b.WriteString(s)
	case Object:
		return writeObject(b, value)
	case Array:
		return writeArray(b, value)
	case []string:
		items := make(Array, len(value))
		for i, item := range value {
			items[i] = item
		}
		return writeArray(b, items)
	default:
		return fmt.Errorf("pyjson: cannot write %T", v)
	}
	return nil
}

func writeObject(b *strings.Builder, o Object) error {
	b.WriteByte('{')
	for i, field := range o {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(String(field.Key))
		b.WriteString(": ")
		if err := write(b, field.Value); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	return nil
}

func writeArray(b *strings.Builder, a Array) error {
	b.WriteByte('[')
	for i, item := range a {
		if i > 0 {
			b.WriteString(", ")
		}
		if err := write(b, item); err != nil {
			return err
		}
	}
	b.WriteByte(']')
	return nil
}

// Float writes what Python's repr gives. Go drops the fraction of a whole
// number and Python keeps it, so 4.0 must not come out as 4.
func Float(f float64) (string, error) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "", fmt.Errorf("pyjson: cannot write %v", f)
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s, nil
}

// String escapes every character outside printable ASCII, which is what
// ensure_ascii does, and leaves the HTML ones alone.
func String(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r >= 0x20 && r <= 0x7e:
				b.WriteRune(r)
			case r > 0xffff:
				r -= 0x10000
				fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
			default:
				fmt.Fprintf(&b, `\u%04x`, r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
