package page

import (
	"encoding/json"
	"strings"
)

// A trailing key with nothing after it arrives Bare, which the substitutions
// answer differently from a key whose value is the empty string.
type PathParam struct {
	Key   string
	Value string
	Bare  bool
}

type PathParams []PathParam

func (p PathParams) Lookup(key string) (PathParam, bool) {
	for _, param := range p {
		if param.Key == key {
			return param, true
		}
	}
	return PathParam{}, false
}

func (p PathParams) Get(key string) string {
	param, _ := p.Lookup(key)
	return param.Value
}

func (p PathParams) Has(key string) bool {
	_, ok := p.Lookup(key)
	return ok
}

func (p PathParams) Put(param PathParam) PathParams {
	for i := range p {
		if p[i].Key == param.Key {
			p[i] = param
			return p
		}
	}
	return append(p, param)
}

// The order the caller wrote is kept, since the variables that read a path
// print them back in it.
func ParsePathParams(raw string) PathParams {
	if raw == "" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	var out PathParams
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil
		}
		name, ok := key.(string)
		if !ok {
			return nil
		}
		var value any
		if err := dec.Decode(&value); err != nil {
			return nil
		}
		switch v := value.(type) {
		case nil:
			out = out.Put(PathParam{Key: name, Bare: true})
		case string:
			out = out.Put(PathParam{Key: name, Value: v})
		default:
			return nil
		}
	}
	return out
}
