package article

import "testing"

func TestUnquoteDecodesEscapes(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"plain":        "plain",
		"%E4%B8%AD":    "中",
		"%F0%9F%98%80": "😀",
		"%00":          "\x00",
		"+x":           "+x",
		"a%2Fb":        "a/b",
		"%2f":          "/",
	}
	for in, want := range cases {
		if got := unquote(in); got != want {
			t.Errorf("unquote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnquoteKeepsUnreadableEscapes(t *testing.T) {
	cases := map[string]string{
		"%ZZ": "%ZZ",
		"%":   "%",
		"a%2": "a%2",
		"%%":  "%%",
	}
	for in, want := range cases {
		if got := unquote(in); got != want {
			t.Errorf("unquote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnquoteSpendsOneReplacementPerSubpart(t *testing.T) {
	cases := map[string]int{
		"%FF%FF":    2,
		"%E4%B8":    1,
		"%E4%B8%41": 1,
		"%C3":       1,
		"%ED%A0%80": 3,
		"%F0%9F":    1,
	}
	for in, want := range cases {
		got := 0
		for _, r := range unquote(in) {
			if r == '�' {
				got++
			}
		}
		if got != want {
			t.Errorf("replacements in unquote(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParamsPutKeepsFirstPosition(t *testing.T) {
	_, params := ParsePath("main/a/1/b/2/a/3", "")
	if got, want := len(params), 2; got != want {
		t.Fatalf("len(params) = %d, want %d", got, want)
	}
	if got, want := params[0].Key, "a"; got != want {
		t.Errorf("params[0].Key = %q, want %q", got, want)
	}
	if got, want := params[0].Value, "3"; got != want {
		t.Errorf("params[0].Value = %q, want %q", got, want)
	}
}

func TestParamsGetAnswersEmptyForBareKey(t *testing.T) {
	_, params := ParsePath("main/norender", "")
	if got, want := params.Get("norender"), ""; got != want {
		t.Errorf("Get(%q) = %q, want %q", "norender", got, want)
	}
}

func TestParamsGetAnswersEmptyForMissingKey(t *testing.T) {
	_, params := ParsePath("main/a/1", "")
	if got, want := params.Get("b"), ""; got != want {
		t.Errorf("Get(%q) = %q, want %q", "b", got, want)
	}
}

func TestParamsEncodeKeepsOnlyOneBareKey(t *testing.T) {
	params := Params{{Key: "z", Value: "1"}, {Key: "first", Bare: true}, {Key: "a", Value: "2"}, {Key: "second", Bare: true}}
	if got, want := Encode(params), "/a/2/z/1/first"; got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}

func TestParsePathFallsBackToHomePage(t *testing.T) {
	name, _ := ParsePath("", "start")
	if got, want := name, "start"; got != want {
		t.Errorf("ParsePath(%q, %q) name = %q, want %q", "", "start", got, want)
	}
}

func TestParsePathFallsBackToMainWithoutHomePage(t *testing.T) {
	name, _ := ParsePath("", "  ")
	if got, want := name, "main"; got != want {
		t.Errorf("ParsePath(%q, %q) name = %q, want %q", "", "  ", got, want)
	}
}
