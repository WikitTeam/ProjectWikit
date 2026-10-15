package form

import (
	"strings"
	"testing"
)

func TestParseReadsNoBreakIndentation(t *testing.T) {
	source := strings.ReplaceAll(template, "    ", "    ")
	source = strings.ReplaceAll(source, "\n ", "\n ")
	def, found, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	if !found {
		t.Fatal("Parse() found = false, want true")
	}
	if len(def.Fields) != 4 {
		t.Fatalf("len(Fields) = %d, want 4", len(def.Fields))
	}
	field, _ := def.Field("NoticeType")
	if len(field.Options) != 2 {
		t.Errorf("len(Field(\"NoticeType\").Options) = %d, want 2", len(field.Options))
	}
}

func TestParseDataReadsNoBreakIndentation(t *testing.T) {
	values, err := ParseData("Body: |\n  one\n  two\nKind: x y\n")
	if err != nil {
		t.Fatalf("ParseData() err = %v, want nil", err)
	}
	if got := values["Body"]; got != "one\ntwo\n" {
		t.Errorf("ParseData()[\"Body\"] = %q, want %q", got, "one\ntwo\n")
	}
	if got := values["Kind"]; got != "x y" {
		t.Errorf("ParseData()[\"Kind\"] = %q, want %q", got, "x y")
	}
}

func TestDataGivesACheckboxBare(t *testing.T) {
	def := parsed(t)
	for stored, want := range map[string]string{"1": "1", "0": "0", "true": "1", "": "0"} {
		got, _ := def.Data(map[string]string{"PinnedNotice": stored}, "PinnedNotice")
		if got != want {
			t.Errorf("Data(PinnedNotice=%q) = %q, want %q", stored, got, want)
		}
	}
}
