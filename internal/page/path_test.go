package page

import "testing"

func TestParsePathParamsKeepsOrder(t *testing.T) {
	got := ParsePathParams(`{"offset": "20", "tag": "euclid"}`)
	want := PathParams{{Key: "offset", Value: "20"}, {Key: "tag", Value: "euclid"}}
	if len(got) != len(want) {
		t.Fatalf("len(ParsePathParams()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ParsePathParams()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParsePathParamsReadsNullAsBare(t *testing.T) {
	got := ParsePathParams(`{"norender": null}`)
	want := PathParam{Key: "norender", Bare: true}
	if len(got) != 1 || got[0] != want {
		t.Errorf("ParsePathParams(null) = %v, want [%v]", got, want)
	}
}

func TestParsePathParamsRejectsWhatIsNotAnObjectOfStrings(t *testing.T) {
	for _, raw := range []string{"", "[]", "{", `{"a": 1}`, `{"a": {"b": "c"}}`, "not json"} {
		if got := ParsePathParams(raw); got != nil {
			t.Errorf("ParsePathParams(%q) = %v, want nil", raw, got)
		}
	}
}
