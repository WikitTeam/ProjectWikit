package form

import "testing"

func TestParseReadsEditorAttributes(t *testing.T) {
	def, _, err := Parse("[[form]]\nfields:\n  Body:\n    type: wiki\n    height: 5\n    minLength: 1\n    maxLength: 300\n[[/form]]")
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	field, _ := def.Field("Body")
	if field.Height != 5 {
		t.Errorf("Height = %d, want 5", field.Height)
	}
	if field.MinLength != 1 {
		t.Errorf("MinLength = %d, want 1", field.MinLength)
	}
	if field.MaxLength != 300 {
		t.Errorf("MaxLength = %d, want 300", field.MaxLength)
	}
}

func TestSerializeReadsBack(t *testing.T) {
	def := parsed(t)
	values := map[string]string{
		"NoticeBody":   "first line\n\nsecond: \"quoted\" #tag",
		"NoticeType":   "important",
		"PinnedNotice": "1",
		"Note":         "yes",
	}
	got, err := ParseData(def.Serialize(values))
	if err != nil {
		t.Fatalf("ParseData(Serialize()) err = %v, want nil", err)
	}
	for key, want := range values {
		if got[key] != want {
			t.Errorf("ParseData(Serialize())[%q] = %q, want %q", key, got[key], want)
		}
	}
}

func TestSerializeWritesFieldsInOrder(t *testing.T) {
	def := parsed(t)
	got := def.Serialize(map[string]string{"NoticeBody": "body", "NoticeType": "normal", "PinnedNotice": "0", "Note": "plain"})
	want := "NoticeBody: body\nNoticeType: 'normal'\nPinnedNotice: '0'\nNote: plain\n"
	if got != want {
		t.Errorf("Serialize() = %q, want %q", got, want)
	}
}

func TestSerializeFillsDefaults(t *testing.T) {
	def := parsed(t)
	got, _ := ParseData(def.Serialize(map[string]string{}))
	if got["NoticeType"] != "normal" {
		t.Errorf("NoticeType = %q, want %q", got["NoticeType"], "normal")
	}
	if got["PinnedNotice"] != "0" {
		t.Errorf("PinnedNotice = %q, want %q", got["PinnedNotice"], "0")
	}
}

func TestSerializeKeepsUnknownKeys(t *testing.T) {
	def := parsed(t)
	got, _ := ParseData(def.Serialize(map[string]string{"OldTitle": "kept"}))
	if got["OldTitle"] != "kept" {
		t.Errorf("OldTitle = %q, want %q", got["OldTitle"], "kept")
	}
}
