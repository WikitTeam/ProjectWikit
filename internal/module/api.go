package module

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type API func(env Env, params map[string]string) (wikijson.Object, error)

// Only the methods that change nothing are registered here. The ones that write
// are checked for a CSRF token first, and nothing does that yet.
var apis = map[string]API{}

func RegisterAPI(module, method string, fn API) {
	if _, ok := registry[module]; !ok {
		panic("module: " + module + " is not in the registry")
	}
	apis[apiKey(module, method)] = fn
}

func LookupAPI(module, method string) (API, bool) {
	info, ok := Lookup(module)
	if !ok || info.Removed {
		return nil, false
	}
	fn, ok := apis[apiKey(info.Name, method)]
	return fn, ok
}

func apiKey(module, method string) string {
	return strings.ToLower(module) + "." + strings.ToLower(method)
}
