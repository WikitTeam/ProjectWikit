package module

import "testing"

func TestLookupIsCaseInsensitive(t *testing.T) {
	for _, name := range []string{"ListPages", "listpages", "  LISTPAGES  "} {
		if _, ok := Lookup(name); !ok {
			t.Errorf("Lookup(%q) = false, want true", name)
		}
	}
}

func TestLookupOfAnUnknownName(t *testing.T) {
	if _, ok := Lookup("no-such-module"); ok {
		t.Error("Lookup(\"no-such-module\") = true, want false")
	}
}

func TestIsInlineOfARemovedModule(t *testing.T) {
	if IsInline("interwiki") {
		t.Error("IsInline(\"interwiki\") = true, want false")
	}
}

func TestHasContentOfARemovedModule(t *testing.T) {
	info, ok := Lookup("interwiki")
	if !ok || !info.HasContent {
		t.Fatalf("Lookup(\"interwiki\").HasContent = %v, want true", ok && info.HasContent)
	}
	if HasContent("interwiki") {
		t.Error("HasContent(\"interwiki\") = true, want false")
	}
}

func TestAllIsSortedByName(t *testing.T) {
	all := All()
	for i := 1; i < len(all); i++ {
		if all[i-1].Name >= all[i].Name {
			t.Errorf("All()[%d].Name = %q, want it sorted after %q", i, all[i].Name, all[i-1].Name)
		}
	}
}

func TestBoolParam(t *testing.T) {
	params := map[string]string{
		"yes": "YES", "one": "1", "t": "t", "true": "true",
		"no": "no", "zero": "0", "f": "F", "false": "false",
		"junk": "maybe", "empty": "",
	}
	cases := []struct {
		key  string
		def  bool
		want bool
	}{
		{"yes", false, true},
		{"one", false, true},
		{"t", false, true},
		{"true", false, true},
		{"no", true, false},
		{"zero", true, false},
		{"f", true, false},
		{"false", true, false},
		{"junk", true, true},
		{"junk", false, false},
		{"empty", true, true},
		{"missing", true, true},
		{"missing", false, false},
	}
	for _, c := range cases {
		if got := BoolParam(params, c.key, c.def); got != c.want {
			t.Errorf("BoolParam(%q, %v) = %v, want %v", c.key, c.def, got, c.want)
		}
	}
}

func TestRegisterRejectsAnUnknownName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Register(\"no-such-module\") did not panic, want a panic")
		}
	}()
	Register("no-such-module", func(Env, map[string]string, string) (string, error) { return "", nil })
}

func TestParseBool(t *testing.T) {
	cases := []struct {
		in         string
		want, isOk bool
	}{
		{"yes", true, true},
		{"TRUE", true, true},
		{" 1 ", true, true},
		{"no", false, true},
		{"f", false, true},
		{"0", false, true},
		{"入组申请", false, false},
		{"", false, false},
	}
	for _, c := range cases {
		got, ok := ParseBool(c.in)
		if ok != c.isOk {
			t.Errorf("ParseBool(%q) ok = %v, want %v", c.in, ok, c.isOk)
			continue
		}
		if ok && got != c.want {
			t.Errorf("ParseBool(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFormModulesTakeNoBody(t *testing.T) {
	for _, name := range []string{"applicationform", "membershipbypassword"} {
		if HasContent(name) {
			t.Errorf("HasContent(%q) = true, want false", name)
		}
	}
}
