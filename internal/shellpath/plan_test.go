package shellpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecide(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pwikit")
	other := filepath.Join(dir, "other-pwikit")
	for _, name := range []string{exe, other} {
		if err := os.WriteFile(name, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gone := filepath.Join(dir, "moved-away", "pwikit")

	cases := []struct {
		name     string
		existing entry
		force    bool
		want     action
	}{
		{"nothing there", entry{}, false, actCreate},
		{"already ours", entry{present: true, link: true, target: exe, working: true}, false, actKeep},
		{"left by a moved pwikit", entry{present: true, link: true, target: gone}, false, actReplace},
		{"another live pwikit", entry{present: true, link: true, target: other, working: true}, false, actRefuse},
		{"another live pwikit with force", entry{present: true, link: true, target: other, working: true}, true, actReplace},
		{"a plain file", entry{present: true, working: true}, false, actRefuse},
		{"a plain file with force", entry{present: true, working: true}, true, actReplace},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := decide(tt.existing, exe, tt.force); got != tt.want {
				t.Errorf("decide(%+v, force=%t) = %v, want %v", tt.existing, tt.force, got, tt.want)
			}
		})
	}
}
