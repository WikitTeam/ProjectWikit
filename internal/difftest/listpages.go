package difftest

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/pyjson"
)

var attrUnescape = strings.NewReplacer(
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", `"`,
	"&#x27;", "'",
	"&amp;", "&",
)

func sortListPagesParams(match []byte) []byte {
	const prefix = `data-list-pages-params="`
	body := string(match)
	if !strings.HasPrefix(body, prefix) || !strings.HasSuffix(body, `"`) {
		return match
	}
	raw := attrUnescape.Replace(body[len(prefix) : len(body)-1])

	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return match
	}
	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make(pyjson.Object, 0, len(keys))
	for _, key := range keys {
		out = append(out, pyjson.Field{Key: key, Value: decoded[key]})
	}
	sorted, err := pyjson.Marshal(out)
	if err != nil {
		return match
	}
	return []byte(prefix + escape.HTML(sorted) + `"`)
}
