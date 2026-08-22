package modules

import "testing"

func TestRegistryCovers23Modules(t *testing.T) {
	if got := len(All()); got != 23 {
		t.Errorf("len(All()) = %d, want 23", got)
	}
}

func TestHasContent(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"countpages", true},
		{"css", true},
		{"listpages", true},
		{"listusers", true},
		{"pagedescription", true},
		{"rate", false},
		{"forumthread", false},
		{"pageimage", false},
		{"tagcloud", false},
		{"nosuchmodule", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasContent(tt.name); got != tt.want {
				t.Errorf("HasContent(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestHasContentIgnoresCaseAndSpace(t *testing.T) {
	for _, name := range []string{"CSS", " css ", "Css"} {
		t.Run(name, func(t *testing.T) {
			if !HasContent(name) {
				t.Errorf("HasContent(%q) = false, want true", name)
			}
		})
	}
}

func TestRemovedModuleReportsNoContent(t *testing.T) {
	info, ok := Lookup("interwiki")
	if !ok {
		t.Fatal("Lookup(\"interwiki\") ok = false, want true")
	}
	if !info.Removed {
		t.Error("Lookup(\"interwiki\").Removed = false, want true")
	}
	if !info.HasContent {
		t.Error("Lookup(\"interwiki\").HasContent = false, want true")
	}
	if HasContent("interwiki") {
		t.Error("HasContent(\"interwiki\") = true, want false")
	}
}

func TestNothingPortedYet(t *testing.T) {
	for _, info := range All() {
		if info.Ported {
			t.Errorf("Lookup(%q).Ported = true, want false", info.Name)
		}
	}
}

func TestAllIsSortedByName(t *testing.T) {
	all := All()
	for i := 1; i < len(all); i++ {
		if all[i-1].Name >= all[i].Name {
			t.Errorf("All()[%d].Name = %q, want it sorted before %q", i, all[i].Name, all[i-1].Name)
		}
	}
}

func TestRatIsRemoved(t *testing.T) {
	info, ok := Lookup("rat")
	if !ok {
		t.Fatal("Lookup(\"rat\") ok = false, want true")
	}
	if !info.Removed {
		t.Error("Lookup(\"rat\").Removed = false, want true")
	}
}
