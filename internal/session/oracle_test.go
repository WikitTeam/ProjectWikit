package session

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateVectors = flag.Bool("update", false, "rewrite the vectors the Django oracle reads")

type vector struct {
	Data    map[string]any `json:"data"`
	Encoded string         `json:"encoded"`
}

func TestWriteVectorsForOracle(t *testing.T) {
	if !*updateVectors {
		t.Skip("run with -update to rewrite the oracle vectors")
	}

	store := fixedStore()
	cases := []map[string]any{
		{},
		{AuthUserID: "1"},
		{
			AuthUserID:      "42",
			AuthUserBackend: "django.contrib.auth.backends.ModelBackend",
			AuthUserHash:    "deadbeef",
		},
		{"中文键": "中文值"},
		{"big": repeat("y", 500)},
		{"nested": map[string]any{"a": float64(1), "b": []any{"x", "y"}}},
	}

	vectors := make([]vector, 0, len(cases))
	for _, data := range cases {
		encoded, err := store.Encode(data)
		if err != nil {
			t.Fatalf("Encode(%v) err = %v, want nil", data, err)
		}
		vectors = append(vectors, vector{Data: data, Encoded: encoded})
	}

	encoded, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(vectors) err = %v, want nil", err)
	}
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatalf("MkdirAll(testdata) err = %v, want nil", err)
	}
	path := filepath.Join("testdata", "go_sessions.json")
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", path, err)
	}
	t.Logf("wrote %s", path)
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
