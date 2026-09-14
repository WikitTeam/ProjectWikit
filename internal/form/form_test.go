package form

import "testing"

const template = `**[[[notice|back]]]**
type: %%form_raw{NoticeType}%%

====

[[form]]
fields:
 NoticeBody:
    type: wiki
    label: "body"
    hint: "wiki syntax allowed"
    minLength: 1
 NoticeType:
    type: select
    label: kind
    values:
      normal: plain
      important: "*"
    default : normal
 PinnedNotice:
    type: checkbox
    label: pinned
    default: 0
 Note:
    label: note
[[/form]]
`

func parsed(t *testing.T) *Definition {
	t.Helper()
	def, found, err := Parse(template)
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	if !found {
		t.Fatal("Parse() found = false, want true")
	}
	return def
}

func TestParseReadsFieldsInOrder(t *testing.T) {
	def := parsed(t)

	want := []string{"NoticeBody", "NoticeType", "PinnedNotice", "Note"}
	if len(def.Fields) != len(want) {
		t.Fatalf("len(Fields) = %d, want %d", len(def.Fields), len(want))
	}
	for i, name := range want {
		if def.Fields[i].Name != name {
			t.Errorf("Fields[%d].Name = %q, want %q", i, def.Fields[i].Name, name)
		}
	}
}

func TestParseReadsFieldAttributes(t *testing.T) {
	def := parsed(t)

	field, ok := def.Field("NoticeBody")
	if !ok {
		t.Fatal("Field(\"NoticeBody\") = _, false, want true")
	}
	if field.Type != TypeWiki {
		t.Errorf("Field(\"NoticeBody\").Type = %q, want %q", field.Type, TypeWiki)
	}
	if field.Label != "body" {
		t.Errorf("Field(\"NoticeBody\").Label = %q, want %q", field.Label, "body")
	}
	if field.Hint != "wiki syntax allowed" {
		t.Errorf("Field(\"NoticeBody\").Hint = %q, want %q", field.Hint, "wiki syntax allowed")
	}
}

func TestParseDefaultsTypeToText(t *testing.T) {
	def := parsed(t)

	field, _ := def.Field("Note")
	if field.Type != TypeText {
		t.Errorf("Field(\"Note\").Type = %q, want %q", field.Type, TypeText)
	}
}

func TestParseReadsSelectOptionsInOrder(t *testing.T) {
	def := parsed(t)

	field, _ := def.Field("NoticeType")
	want := []Option{{Key: "normal", Label: "plain"}, {Key: "important", Label: "*"}}
	if len(field.Options) != len(want) {
		t.Fatalf("len(Options) = %d, want %d", len(field.Options), len(want))
	}
	for i := range want {
		if field.Options[i] != want[i] {
			t.Errorf("Options[%d] = %+v, want %+v", i, field.Options[i], want[i])
		}
	}
	if field.Default != "normal" {
		t.Errorf("Field(\"NoticeType\").Default = %q, want %q", field.Default, "normal")
	}
}

func TestParseKeepsReservedOptionKeysAsText(t *testing.T) {
	def, _, err := Parse("[[form]]\nfields:\n done:\n  type: select\n  values:\n   \"08\": eight\n   yes: on\n[[/form]]")
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	field, _ := def.Field("done")
	want := []string{"08", "yes"}
	for i, key := range want {
		if field.Options[i].Key != key {
			t.Errorf("Options[%d].Key = %q, want %q", i, field.Options[i].Key, key)
		}
	}
}

func TestParseFindsNoBlock(t *testing.T) {
	def, found, err := Parse("plain page")
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	if found {
		t.Error("Parse(\"plain page\") found = true, want false")
	}
	if def != nil {
		t.Errorf("Parse(\"plain page\") = %+v, want nil", def)
	}
}

func TestStripRemovesTheBlock(t *testing.T) {
	got := Strip("head\n[[form]]\nfields:\n a: {}\n[[/form]]\ntail")
	if want := "head\n\ntail"; got != want {
		t.Errorf("Strip() = %q, want %q", got, want)
	}
}

func TestStripLeavesAPageWithoutOne(t *testing.T) {
	if got := Strip("head\ntail"); got != "head\ntail" {
		t.Errorf("Strip() = %q, want %q", got, "head\ntail")
	}
}

func TestParseData(t *testing.T) {
	values, err := ParseData("NoticeBody: \"one\\ntwo\"\nNoticeType: important\nPinnedNotice: '0'\n")
	if err != nil {
		t.Fatalf("ParseData() err = %v, want nil", err)
	}
	want := map[string]string{"NoticeBody": "one\ntwo", "NoticeType": "important", "PinnedNotice": "0"}
	for key, value := range want {
		if values[key] != value {
			t.Errorf("ParseData()[%q] = %q, want %q", key, values[key], value)
		}
	}
}

func TestRawIsTheStoredValue(t *testing.T) {
	def := parsed(t)
	values := map[string]string{"NoticeType": "important"}

	got, ok := def.Raw(values, "NoticeType")
	if !ok {
		t.Fatal("Raw(\"NoticeType\") = _, false, want true")
	}
	if got != "important" {
		t.Errorf("Raw(\"NoticeType\") = %q, want %q", got, "important")
	}
}

func TestRawFallsBackToTheDefault(t *testing.T) {
	def := parsed(t)

	got, _ := def.Raw(map[string]string{}, "NoticeType")
	if got != "normal" {
		t.Errorf("Raw(\"NoticeType\") = %q, want %q", got, "normal")
	}
}

func TestRawRejectsAnUnknownField(t *testing.T) {
	def := parsed(t)

	if _, ok := def.Raw(map[string]string{"Other": "x"}, "Other"); ok {
		t.Error("Raw(\"Other\") = _, true, want false")
	}
}

func TestDataMapsASelectToItsLabel(t *testing.T) {
	def := parsed(t)

	got, _ := def.Data(map[string]string{"NoticeType": "important"}, "NoticeType")
	if got != "*" {
		t.Errorf("Data(\"NoticeType\") = %q, want %q", got, "*")
	}
}

func TestDataLeavesWikiUnescaped(t *testing.T) {
	def := parsed(t)

	got, _ := def.Data(map[string]string{"NoticeBody": "**bold**"}, "NoticeBody")
	if got != "**bold**" {
		t.Errorf("Data(\"NoticeBody\") = %q, want %q", got, "**bold**")
	}
}

func TestDataEscapesText(t *testing.T) {
	def := parsed(t)

	got, _ := def.Data(map[string]string{"Note": "**bold**"}, "Note")
	if want := "@@**bold**@@"; got != want {
		t.Errorf("Data(\"Note\") = %q, want %q", got, want)
	}
}

func TestFieldMatchesWithoutCase(t *testing.T) {
	def := parsed(t)

	if _, ok := def.Field("pinnednotice"); !ok {
		t.Error("Field(\"pinnednotice\") = _, false, want true")
	}
}

func TestLabelAndHint(t *testing.T) {
	def := parsed(t)

	if got, _ := def.Label("NoticeType"); got != "kind" {
		t.Errorf("Label(\"NoticeType\") = %q, want %q", got, "kind")
	}
	if got, _ := def.Hint("NoticeBody"); got != "wiki syntax allowed" {
		t.Errorf("Hint(\"NoticeBody\") = %q, want %q", got, "wiki syntax allowed")
	}
}

func TestEscapeMarkup(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"plain", "@@plain@@"},
		{"a@@b", "@@a@@@@@@@@@@b@@"},
	}
	for _, c := range cases {
		if got := EscapeMarkup(c.in); got != c.want {
			t.Errorf("EscapeMarkup(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseVar(t *testing.T) {
	cases := []struct {
		in    string
		kind  string
		field string
		ok    bool
	}{
		{"form_data{Body}", VarData, "Body", true},
		{"form_raw{Body}", VarRaw, "Body", true},
		{"form_label{Body}", VarLabel, "Body", true},
		{"form_hint{Body}", VarHint, "Body", true},
		{"form_data{}", "", "", false},
		{"form_data{Body", "", "", false},
		{"content{1}", "", "", false},
		{"form_other{Body}", "", "", false},
	}
	for _, c := range cases {
		kind, field, ok := ParseVar(c.in)
		if ok != c.ok {
			t.Errorf("ParseVar(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if kind != c.kind || field != c.field {
			t.Errorf("ParseVar(%q) = %q, %q, want %q, %q", c.in, kind, field, c.kind, c.field)
		}
	}
}
