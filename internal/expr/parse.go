package expr

import (
	"errors"
	"strconv"
	"strings"
)

var errSyntax = errors.New("expr: syntax error")

type node interface{}

type constNode struct{ v Value }

type unaryNode struct {
	op string
	x  node
}

type binNode struct {
	op   string
	x, y node
}

type cmpNode struct {
	ops   []string
	items []node
}

type boolNode struct {
	op    string
	items []node
}

type callNode struct {
	name string
	args []node
}

type tokenKind uint8

const (
	tokEOF tokenKind = iota
	tokNumber
	tokString
	tokIdent
	tokOp
)

type token struct {
	kind tokenKind
	text string
	v    Value
}

var operators = []string{"==", "!=", "<=", ">=", "<", ">", "+", "-", "*", "/", "^", "(", ")", ","}

const (
	quoteSingle = '\''
	quoteDouble = '"'
	backslash   = '\\'
)

func lex(src string) ([]token, error) {
	var out []token
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == quoteSingle || c == quoteDouble:
			s, next, err := lexString(src, i)
			if err != nil {
				return nil, err
			}
			out = append(out, token{kind: tokString, v: StrOf(s)})
			i = next
		case isDigit(c) || (c == '.' && i+1 < len(src) && isDigit(src[i+1])):
			v, next, err := lexNumber(src, i)
			if err != nil {
				return nil, err
			}
			out = append(out, token{kind: tokNumber, v: v})
			i = next
		case isIdentStart(c):
			j := i
			for j < len(src) && isIdentPart(src[j]) {
				j++
			}
			out = append(out, token{kind: tokIdent, text: src[i:j]})
			i = j
		default:
			op := matchOperator(src[i:])
			if op == "" {
				return nil, errSyntax
			}
			out = append(out, token{kind: tokOp, text: op})
			i += len(op)
		}
	}
	return append(out, token{kind: tokEOF}), nil
}

func matchOperator(s string) string {
	for _, op := range operators {
		if strings.HasPrefix(s, op) {
			return op
		}
	}
	return ""
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentPart(c byte) bool { return isIdentStart(c) || isDigit(c) }

func lexString(src string, start int) (string, int, error) {
	quote := src[start]
	var b strings.Builder
	i := start + 1
	for i < len(src) {
		switch src[i] {
		case quote:
			return b.String(), i + 1, nil
		case backslash:
			if i+1 >= len(src) {
				return "", 0, errSyntax
			}
			switch src[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(src[i+1])
			}
			i += 2
		default:
			b.WriteByte(src[i])
			i++
		}
	}
	return "", 0, errSyntax
}

func lexNumber(src string, start int) (Value, int, error) {
	i := start
	isFloat := false
	for i < len(src) {
		c := src[i]
		if isDigit(c) {
			i++
			continue
		}
		if c == '.' && !isFloat {
			isFloat = true
			i++
			continue
		}
		if (c == 'e' || c == 'E') && i > start && exponentFollows(src, i) {
			isFloat = true
			i += 2
			continue
		}
		break
	}

	text := src[start:i]
	if isFloat {
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return None(), 0, errSyntax
		}
		return FloatOf(f), i, nil
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return None(), 0, errSyntax
	}
	return IntOf(n), i, nil
}

func exponentFollows(src string, i int) bool {
	if i+1 >= len(src) {
		return false
	}
	if isDigit(src[i+1]) {
		return true
	}
	return (src[i+1] == '+' || src[i+1] == '-') && i+2 < len(src) && isDigit(src[i+2])
}

type parser struct {
	tokens []token
	pos    int
}

func parse(src string) (node, error) {
	tokens, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	n, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, errSyntax
	}
	return n, nil
}

func (p *parser) peek() token { return p.tokens[p.pos] }

func (p *parser) acceptOp(ops ...string) (string, bool) {
	t := p.peek()
	if t.kind != tokOp {
		return "", false
	}
	for _, op := range ops {
		if t.text == op {
			p.pos++
			return op, true
		}
	}
	return "", false
}

func (p *parser) acceptWord(word string) bool {
	if t := p.peek(); t.kind == tokIdent && t.text == word {
		p.pos++
		return true
	}
	return false
}

func (p *parser) parseOr() (node, error)  { return p.parseBool("or", p.parseAnd) }
func (p *parser) parseAnd() (node, error) { return p.parseBool("and", p.parseCompare) }

func (p *parser) parseBool(word string, next func() (node, error)) (node, error) {
	first, err := next()
	if err != nil {
		return nil, err
	}
	items := []node{first}
	for p.acceptWord(word) {
		operand, err := next()
		if err != nil {
			return nil, err
		}
		items = append(items, operand)
	}
	if len(items) == 1 {
		return first, nil
	}
	return &boolNode{op: word, items: items}, nil
}

func (p *parser) parseCompare() (node, error) {
	first, err := p.parseXor()
	if err != nil {
		return nil, err
	}
	var ops []string
	items := []node{first}
	for {
		op, ok := p.acceptOp("==", "!=", "<=", ">=", "<", ">")
		if !ok {
			break
		}
		operand, err := p.parseXor()
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
		items = append(items, operand)
	}
	if len(ops) == 0 {
		return first, nil
	}
	return &cmpNode{ops: ops, items: items}, nil
}

func (p *parser) parseXor() (node, error) { return p.parseBinary(p.parseAdd, "^") }
func (p *parser) parseAdd() (node, error) { return p.parseBinary(p.parseMul, "+", "-") }
func (p *parser) parseMul() (node, error) { return p.parseBinary(p.parseUnary, "*", "/") }

func (p *parser) parseBinary(next func() (node, error), ops ...string) (node, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for {
		op, ok := p.acceptOp(ops...)
		if !ok {
			return left, nil
		}
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = &binNode{op: op, x: left, y: right}
	}
}

func (p *parser) parseUnary() (node, error) {
	if op, ok := p.acceptOp("-", "+"); ok {
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: op, x: operand}, nil
	}
	if p.acceptWord("not") {
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: "not", x: operand}, nil
	}
	return p.parseAtom()
}

func (p *parser) parseAtom() (node, error) {
	t := p.peek()
	switch t.kind {
	case tokNumber, tokString:
		p.pos++
		return &constNode{v: t.v}, nil
	case tokIdent:
		p.pos++
		switch t.text {
		case "True":
			return &constNode{v: BoolOf(true)}, nil
		case "False":
			return &constNode{v: BoolOf(false)}, nil
		case "None":
			return &constNode{v: None()}, nil
		}
		if _, ok := p.acceptOp("("); !ok {
			return nil, errSyntax
		}
		args, err := p.parseArgs()
		if err != nil {
			return nil, err
		}
		return &callNode{name: strings.ToLower(t.text), args: args}, nil
	case tokOp:
		if _, ok := p.acceptOp("("); ok {
			inner, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if _, ok := p.acceptOp(")"); !ok {
				return nil, errSyntax
			}
			return inner, nil
		}
	}
	return nil, errSyntax
}

func (p *parser) parseArgs() ([]node, error) {
	var args []node
	if _, ok := p.acceptOp(")"); ok {
		return args, nil
	}
	for {
		arg, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if _, ok := p.acceptOp(","); ok {
			continue
		}
		if _, ok := p.acceptOp(")"); ok {
			return args, nil
		}
		return nil, errSyntax
	}
}
