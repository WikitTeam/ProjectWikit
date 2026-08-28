package page

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
