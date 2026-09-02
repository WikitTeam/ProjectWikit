package moduleapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParamsFlattensScalars(t *testing.T) {
	got := params(json.RawMessage(`{"T": 123, "Name": "a b", "Deep": 1.5, "On": true}`))
	want := map[string]string{"t": "123", "name": "a b", "deep": "1.5", "on": "true"}
	if len(got) != len(want) {
		t.Fatalf("len(params()) = %d, want %d", len(got), len(want))
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("params()[%q] = %q, want %q", key, got[key], value)
		}
	}
}

func TestParamsDropsWhatIsNotAScalar(t *testing.T) {
	got := params(json.RawMessage(`{"a": null, "b": [1], "c": {"d": 1}, "e": "kept"}`))
	if len(got) != 1 || got["e"] != "kept" {
		t.Errorf("params() = %v, want map[e:kept]", got)
	}
}

func TestParamsOfWhatIsNotAnObject(t *testing.T) {
	for _, raw := range []string{"", "[]", "7", "not json"} {
		if got := params(json.RawMessage(raw)); got != nil {
			t.Errorf("params(%q) = %v, want nil", raw, got)
		}
	}
}

func TestFieldEscapesLikeTheStoredJSON(t *testing.T) {
	cases := map[string]string{
		`<p>a</p>`: `{"result": "<p>a</p>"}`,
		`"quoted"`: `{"result": "\"quoted\""}`,
		"line\n":   `{"result": "line\n"}`,
		"é":        `{"result": "\u00e9"}`,
	}
	for value, want := range cases {
		if got := field("result", value); got != want {
			t.Errorf("field(result, %q) = %q, want %q", value, got, want)
		}
	}
}

func TestPeekReadsAShortBody(t *testing.T) {
	raw, rest, err := peek(io.NopCloser(strings.NewReader(`{"a": 1}`)))
	if err != nil {
		t.Fatalf("peek() err = %v, want nil", err)
	}
	if string(raw) != `{"a": 1}` {
		t.Errorf("peek() raw = %q, want %q", raw, `{"a": 1}`)
	}
	if rest != nil {
		t.Error("peek() rest != nil, want nil")
	}
}

func TestPeekLeavesALongBodyToTheCaller(t *testing.T) {
	body := strings.Repeat("x", maxParsed+10)
	raw, rest, err := peek(io.NopCloser(strings.NewReader(body)))
	if err == nil {
		t.Fatal("peek(long) err = nil, want an error")
	}
	if len(raw) != maxParsed {
		t.Errorf("len(peek(long) raw) = %d, want %d", len(raw), maxParsed)
	}
	tail, _ := io.ReadAll(rest)
	if len(raw)+len(tail) != len(body) {
		t.Errorf("peek(long) read back %d bytes, want %d", len(raw)+len(tail), len(body))
	}
}

type recorder struct {
	body   []byte
	method string
	called bool
}

func (u *recorder) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	u.called = true
	u.method = r.Method
	u.body, _ = io.ReadAll(r.Body)
}

func post(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, Path, bytes.NewReader([]byte(body)))
	r.Header.Set("Content-Type", jsonMime)
	return r
}

func TestWriteMethodsGoUpstreamWithTheirBody(t *testing.T) {
	up := &recorder{}
	body := `{"module": "rate", "method": "rate", "params": {"value": 1}}`

	New(Deps{}, up).ServeHTTP(httptest.NewRecorder(), post(body))

	if !up.called {
		t.Fatal("upstream called = false, want true")
	}
	if string(up.body) != body {
		t.Errorf("upstream body = %q, want %q", up.body, body)
	}
}

func TestABodyTooLargeToReadGoesUpstreamWhole(t *testing.T) {
	up := &recorder{}
	body := `{"module": "forumnewpost", "method": "submit", "params": {"source": "` +
		strings.Repeat("x", maxParsed) + `"}}`

	New(Deps{}, up).ServeHTTP(httptest.NewRecorder(), post(body))

	if string(up.body) != body {
		t.Errorf("len(upstream body) = %d, want %d", len(up.body), len(body))
	}
}

func TestWhatIsNotAPostGoesUpstream(t *testing.T) {
	up := &recorder{}
	r := httptest.NewRequest(http.MethodGet, Path, nil)

	New(Deps{}, up).ServeHTTP(httptest.NewRecorder(), r)

	if up.method != http.MethodGet {
		t.Errorf("upstream method = %q, want %q", up.method, http.MethodGet)
	}
}

func TestABodyThatIsNotACallGoesUpstream(t *testing.T) {
	for _, body := range []string{`[1, 2]`, `not json`, `{"method": "render"`} {
		up := &recorder{}
		New(Deps{}, up).ServeHTTP(httptest.NewRecorder(), post(body))
		if !up.called {
			t.Errorf("upstream called for %q = false, want true", body)
		}
	}
}
