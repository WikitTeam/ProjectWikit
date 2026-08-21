package cgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func repoFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", path, err)
	}
	return string(data)
}

func TestABIVersionAgreesEverywhere(t *testing.T) {
	declared := strings.TrimSpace(repoFile(t, "ftml-capi", "ABI_VERSION"))
	version, err := strconv.Atoi(declared)
	if err != nil {
		t.Fatalf("ABI_VERSION = %q, want an integer", declared)
	}
	if version != ABIVersion {
		t.Errorf("ABI_VERSION = %d, want %d", version, ABIVersion)
	}

	symbol := fmt.Sprintf("ftml_abi_%d", ABIVersion)
	for _, source := range []struct{ name, body string }{
		{"ftml.h", repoFile(t, "internal", "renderer", "cgo", "ftml.h")},
		{"ftml-capi/src/lib.rs", repoFile(t, "ftml-capi", "src", "lib.rs")},
	} {
		if !strings.Contains(source.body, symbol) {
			t.Errorf("%s does not mention %s, want it", source.name, symbol)
		}
	}
}
