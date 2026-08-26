package pyjson

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden file and the corpus the oracle reads")

const (
	goldenPath = "testdata/pyjson.golden"
	corpusPath = "testdata/pyjson_corpus.json"
)

func escaped(r rune) string { return fmt.Sprintf(`\u%04x`, r) }

func corpus() []string {
	return []string{
		`null`,
		`true`,
		`false`,
		`0`,
		`-17`,
		`4.0`,
		`3.5`,
		`-0.25`,
		`0.1`,
		`100.0`,
		`1e-05`,
		`""`,
		`"plain"`,
		`"with \"quotes\" and \\ backslash"`,
		`"tab\there"`,
		`"line\nbreak"`,
		`"` + escaped(0x07) + `bell"`,
		`"` + escaped(0x7f) + `del"`,
		`"<b>&amp;</b>"`,
		`"it's"`,
		`"中文"`,
		`"emoji 😀"`,
		`"` + escaped(0xa0) + `nbsp"`,
		`[]`,
		`[1, 2, 3]`,
		`["a", null, true, 2.0]`,
		`[[1], [], [null]]`,
		`{}`,
		`{"a": 1}`,
		`{"b": 1, "a": 2}`,
		`{"user": {"type": "anonymous", "name": "匿名用户", "username": null}, "notificationCount": 0}`,
		`{"nested": {"list": [{"k": "v"}, {}]}}`,
		`{"键": "值"}`,
		`{"rating": 4.0, "votes": 12, "popularity": 92, "mode": "updown"}`,
	}
}

func decode(t *testing.T, doc string) any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(doc))
	dec.UseNumber()
	value, err := decodeValue(t, dec)
	if err != nil {
		t.Fatalf("decode(%s) err = %v, want nil", doc, err)
	}
	return value
}

func decodeValue(t *testing.T, dec *json.Decoder) (any, error) {
	t.Helper()
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeToken(t, dec, token)
}

func decodeToken(t *testing.T, dec *json.Decoder, token json.Token) (any, error) {
	t.Helper()
	switch value := token.(type) {
	case nil:
		return nil, nil
	case bool:
		return value, nil
	case string:
		return value, nil
	case json.Number:
		if strings.ContainsAny(value.String(), ".eE") {
			return value.Float64()
		}
		return value.Int64()
	case json.Delim:
		if value == '{' {
			return decodeObject(t, dec)
		}
		return decodeArray(t, dec)
	}
	return nil, fmt.Errorf("unexpected token %v", token)
}

func decodeObject(t *testing.T, dec *json.Decoder) (Object, error) {
	t.Helper()
	out := Object{}
	for {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return out, nil
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("object key %v is not a string", token)
		}
		value, err := decodeValue(t, dec)
		if err != nil {
			return nil, err
		}
		out = append(out, Field{Key: key, Value: value})
	}
}

func decodeArray(t *testing.T, dec *json.Decoder) (Array, error) {
	t.Helper()
	out := Array{}
	for {
		token, err := dec.Token()
		if err == io.EOF {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok && delim == ']' {
			return out, nil
		}
		value, err := decodeToken(t, dec, token)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
}

func TestMarshalMatchesGolden(t *testing.T) {
	docs := corpus()

	var b strings.Builder
	for _, doc := range docs {
		got, err := Marshal(decode(t, doc))
		if err != nil {
			t.Fatalf("Marshal(%s) err = %v, want nil", doc, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", doc, got)
	}
	got := b.String()

	if *update {
		writeCorpus(t, docs)
		if err := os.WriteFile(filepath.FromSlash(goldenPath), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(filepath.FromSlash(goldenPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("Marshal(corpus) = %q, want %q", got, string(want))
	}
}

func writeCorpus(t *testing.T, docs []string) {
	t.Helper()
	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(corpus) err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.FromSlash(corpusPath), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", corpusPath, err)
	}
}

func TestMarshalRejectsUnknownType(t *testing.T) {
	if _, err := Marshal(struct{}{}); err == nil {
		t.Error("Marshal(struct{}{}) err = nil, want an error")
	}
}

func TestMarshalKeepsObjectOrder(t *testing.T) {
	got, err := Marshal(Object{{Key: "b", Value: 1}, {Key: "a", Value: 2}})
	if err != nil {
		t.Fatalf("Marshal() err = %v, want nil", err)
	}
	want := `{"b": 1, "a": 2}`
	if got != want {
		t.Errorf("Marshal() = %q, want %q", got, want)
	}
}
