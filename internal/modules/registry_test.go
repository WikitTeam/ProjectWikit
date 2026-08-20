package modules

import "testing"

func TestRegistryCovers23Modules(t *testing.T) {
	if got := len(All()); got != 23 {
		t.Errorf("len(All()) = %d，期望 23", got)
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
				t.Errorf("HasContent(%q) = %v，期望 %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestHasContentIgnoresCaseAndSpace(t *testing.T) {
	for _, name := range []string{"CSS", " css ", "Css"} {
		t.Run(name, func(t *testing.T) {
			if !HasContent(name) {
				t.Errorf("HasContent(%q) = false，期望 true", name)
			}
		})
	}
}

func TestRemovedModuleReportsNoContent(t *testing.T) {
	info, ok := Lookup("interwiki")
	if !ok {
		t.Fatal("Lookup(\"interwiki\") ok = false，期望 true")
	}
	if !info.Removed {
		t.Error("Lookup(\"interwiki\").Removed = false，期望 true")
	}
	if !info.HasContent {
		t.Error("Lookup(\"interwiki\").HasContent = false，期望 true")
	}
	if HasContent("interwiki") {
		t.Error("HasContent(\"interwiki\") = true，期望 false")
	}
}

func TestNothingPortedYet(t *testing.T) {
	for _, info := range All() {
		if info.Ported {
			t.Errorf("Lookup(%q).Ported = true，期望 false", info.Name)
		}
	}
}

func TestAllIsSortedByName(t *testing.T) {
	all := All()
	for i := 1; i < len(all); i++ {
		if all[i-1].Name >= all[i].Name {
			t.Errorf("All()[%d].Name = %q，期望排在 %q 之前", i, all[i].Name, all[i-1].Name)
		}
	}
}
