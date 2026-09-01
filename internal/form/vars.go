package form

import "strings"

const (
	VarData  = "form_data"
	VarRaw   = "form_raw"
	VarLabel = "form_label"
	VarHint  = "form_hint"
)

var varNames = []string{VarData, VarRaw, VarLabel, VarHint}

func ParseVar(name string) (kind, field string, ok bool) {
	for _, prefix := range varNames {
		rest, cut := strings.CutPrefix(name, prefix+"{")
		if !cut {
			continue
		}
		field, cut = strings.CutSuffix(rest, "}")
		if !cut || field == "" {
			return "", "", false
		}
		return prefix, field, true
	}
	return "", "", false
}
