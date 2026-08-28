package modules

import "github.com/WikitTeam/ProjectWikit/internal/module"

// pathUnder puts the path parameters underneath the module's own, so what the
// page wrote wins over what the reader appended to the URL.
func pathUnder(env module.Env, params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	if env.Page != nil {
		for _, param := range env.Page.PathParams {
			out[param.Key] = param.Value
		}
	}
	for key, value := range params {
		out[key] = value
	}
	return out
}

// pathOver lets the path parameters replace the module's own. Only pagesbytag
// reads them this way round.
func pathOver(env module.Env, params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for key, value := range params {
		out[key] = value
	}
	if env.Page != nil {
		for _, param := range env.Page.PathParams {
			out[param.Key] = param.Value
		}
	}
	return out
}
