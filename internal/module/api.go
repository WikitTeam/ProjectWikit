package module

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type API func(env Env, params map[string]string) (wikijson.Object, error)

var apis = map[string]API{}

var writes = map[string]bool{}

func RegisterAPI(module, method string, fn API) {
	register(module, method, fn, false)
}

// A method registered here changes something, so the caller has to have checked
// the CSRF token before it runs.
func RegisterWriteAPI(module, method string, fn API) {
	register(module, method, fn, true)
}

func register(module, method string, fn API, writes_ bool) {
	if _, ok := registry[module]; !ok {
		panic("module: " + module + " is not in the registry")
	}
	key := apiKey(module, method)
	apis[key] = fn
	writes[key] = writes_
}

func LookupAPI(module, method string) (fn API, safe, ok bool) {
	info, found := Lookup(module)
	if !found || info.Removed {
		return nil, false, false
	}
	key := apiKey(info.Name, method)
	fn, ok = apis[key]
	return fn, !writes[key], ok
}

func apiKey(module, method string) string {
	return strings.ToLower(module) + "." + strings.ToLower(method)
}
