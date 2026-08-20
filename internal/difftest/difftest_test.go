package difftest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resp(status int, body string, header http.Header) Response {
	if header == nil {
		header = http.Header{}
	}
	return Response{Status: status, Header: header, Body: []byte(body)}
}

func headers(pairs ...string) http.Header {
	h := http.Header{}
	for i := 0; i+1 < len(pairs); i += 2 {
		h.Add(pairs[i], pairs[i+1])
	}
	return h
}

func TestCompareReportsIdenticalResponsesAsSame(t *testing.T) {
	c := NewComparer()
	a := resp(200, "<p>hi</p>", headers("Content-Type", "text/html"))
	b := resp(200, "<p>hi</p>", headers("Content-Type", "text/html"))

	if got := c.Compare(a, b); !got.Same() {
		t.Errorf("Compare() = %v, want same", got)
	}
}

func TestCompareReportsStatus(t *testing.T) {
	c := NewComparer()
	got := c.Compare(resp(200, "", nil), resp(404, "", nil))

	if len(got.Diffs) != 1 || got.Diffs[0].Kind != KindStatus {
		t.Fatalf("Diffs = %v, want one status diff", got.Diffs)
	}
	if got.Diffs[0].A != "200" || got.Diffs[0].B != "404" {
		t.Errorf("status diff = a:%s b:%s, want a:200 b:404", got.Diffs[0].A, got.Diffs[0].B)
	}
}

func TestCompareIgnoresNoisyHeaders(t *testing.T) {
	c := NewComparer()
	a := resp(200, "", headers("Date", "Mon", "Server", "nginx", "Content-Length", "0"))
	b := resp(200, "", headers("Date", "Tue", "Server", "gunicorn", "Content-Length", "7"))

	if got := c.Compare(a, b); !got.Same() {
		t.Errorf("Compare() = %v, want same", got)
	}
}

func TestCompareReportsHeaderDifference(t *testing.T) {
	c := NewComparer()
	a := resp(200, "", headers("Content-Type", "text/html"))
	b := resp(200, "", headers("Content-Type", "application/json"))

	got := c.Compare(a, b)
	if len(got.Diffs) != 1 || got.Diffs[0].Name != "Content-Type" {
		t.Fatalf("Diffs = %v, want one Content-Type diff", got.Diffs)
	}
}

func TestCompareReportsHeaderPresentOnOneSideOnly(t *testing.T) {
	c := NewComparer()
	a := resp(200, "", headers("X-Wikit", "1"))
	b := resp(200, "", nil)

	got := c.Compare(a, b)
	if len(got.Diffs) != 1 || got.Diffs[0].B != "" {
		t.Fatalf("Diffs = %v, want one diff with an empty b side", got.Diffs)
	}
}

func TestCompareMatchesSetCookieByNameOnly(t *testing.T) {
	c := NewComparer()
	a := resp(200, "", headers("Set-Cookie", "sessionid=aaa; Path=/", "Set-Cookie", "csrftoken=xxx"))
	b := resp(200, "", headers("Set-Cookie", "csrftoken=yyy", "Set-Cookie", "sessionid=bbb; Path=/"))

	if got := c.Compare(a, b); !got.Same() {
		t.Errorf("Compare() = %v, want same", got)
	}
}

func TestCompareReportsMissingCookie(t *testing.T) {
	c := NewComparer()
	a := resp(200, "", headers("Set-Cookie", "sessionid=aaa", "Set-Cookie", "csrftoken=xxx"))
	b := resp(200, "", headers("Set-Cookie", "sessionid=bbb"))

	got := c.Compare(a, b)
	if len(got.Diffs) != 1 || got.Diffs[0].Name != "Set-Cookie" {
		t.Fatalf("Diffs = %v, want one Set-Cookie diff", got.Diffs)
	}
}

func TestCompareScrubsCSRFToken(t *testing.T) {
	c := NewComparer()
	a := resp(200, `<input name="csrfmiddlewaretoken" value="AAA">`, nil)
	b := resp(200, `<input name="csrfmiddlewaretoken" value="BBB">`, nil)

	got := c.Compare(a, b)
	if !got.Same() {
		t.Errorf("Compare() = %v, want same", got)
	}
	if got.Scrubs["csrfmiddlewaretoken"] != 2 {
		t.Errorf("Scrubs[csrfmiddlewaretoken] = %d, want 2", got.Scrubs["csrfmiddlewaretoken"])
	}
}

func TestCompareReportsBodyDifferenceWithLineNumber(t *testing.T) {
	c := NewComparer()
	a := resp(200, "same\nsame\nleft\n", nil)
	b := resp(200, "same\nsame\nright\n", nil)

	got := c.Compare(a, b)
	if len(got.Diffs) != 1 || got.Diffs[0].Kind != KindBody {
		t.Fatalf("Diffs = %v, want one body diff", got.Diffs)
	}
	if !strings.Contains(got.Diffs[0].Name, "line 3") {
		t.Errorf("Name = %q, want substring %q", got.Diffs[0].Name, "line 3")
	}
	if got.Diffs[0].A != "left" || got.Diffs[0].B != "right" {
		t.Errorf("excerpt = a:%q b:%q, want a:left b:right", got.Diffs[0].A, got.Diffs[0].B)
	}
}

func TestCompareReportsTruncatedBody(t *testing.T) {
	c := NewComparer()
	a := resp(200, "one\ntwo\n", nil)
	b := resp(200, "one\n", nil)

	got := c.Compare(a, b)
	if len(got.Diffs) != 1 || got.Diffs[0].B != "" {
		t.Fatalf("Diffs = %v, want one body diff with an empty b excerpt", got.Diffs)
	}
}

func TestParseCorpus(t *testing.T) {
	src := `
# comment
/
GET /main
post /api/articles
!GET /-/login
`
	got, err := ParseCorpus(src)
	if err != nil {
		t.Fatalf("ParseCorpus() err = %v, want nil", err)
	}
	want := []Request{
		{Method: "GET", Target: "/"},
		{Method: "GET", Target: "/main"},
		{Method: "POST", Target: "/api/articles"},
		{Method: "GET", Target: "/-/login", KnownDiffers: true},
	}
	if len(got) != len(want) {
		t.Fatalf("len(ParseCorpus()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Method != want[i].Method {
			t.Errorf("ParseCorpus()[%d].Method = %q, want %q", i, got[i].Method, want[i].Method)
		}
		if got[i].Target != want[i].Target {
			t.Errorf("ParseCorpus()[%d].Target = %q, want %q", i, got[i].Target, want[i].Target)
		}
		if got[i].KnownDiffers != want[i].KnownDiffers {
			t.Errorf("ParseCorpus()[%d].KnownDiffers = %v, want %v", i, got[i].KnownDiffers, want[i].KnownDiffers)
		}
	}
}

func TestParseCorpusRejectsBadLines(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"target without a leading slash", "main"},
		{"too many fields", "GET /main extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseCorpus(tt.src); err == nil {
				t.Errorf("ParseCorpus(%q) err = nil, want non-nil", tt.src)
			}
		})
	}
}

func TestNewRunnerRejectsBadBase(t *testing.T) {
	tests := []struct {
		name string
		base string
	}{
		{"empty", ""},
		{"no scheme", "127.0.0.1:8000"},
		{"no host", "http://"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewRunner(tt.base, "http://127.0.0.1:8000"); err == nil {
				t.Errorf("NewRunner(%q, _) err = nil, want non-nil", tt.base)
			}
		})
	}
}

func TestRunnerSendsTheSameHostToBothSides(t *testing.T) {
	var hostA, hostB string
	sideA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostA = r.Host
	}))
	defer sideA.Close()
	sideB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostB = r.Host
	}))
	defer sideB.Close()

	runner, err := NewRunner(sideA.URL, sideB.URL)
	if err != nil {
		t.Fatalf("NewRunner() err = %v, want nil", err)
	}
	runner.Host = "wiki.example"

	if _, _, _, err := runner.Do(context.Background(), Request{Target: "/"}); err != nil {
		t.Fatalf("Do() err = %v, want nil", err)
	}
	if hostA != "wiki.example" || hostB != "wiki.example" {
		t.Errorf("Host = a:%q b:%q, want both %q", hostA, hostB, "wiki.example")
	}
}

func TestRunnerDoesNotFollowRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/elsewhere", http.StatusMovedPermanently)
			return
		}
		w.Write([]byte("followed"))
	}))
	defer target.Close()

	runner, err := NewRunner(target.URL, target.URL)
	if err != nil {
		t.Fatalf("NewRunner() err = %v, want nil", err)
	}
	_, a, _, err := runner.Do(context.Background(), Request{Target: "/"})
	if err != nil {
		t.Fatalf("Do() err = %v, want nil", err)
	}
	if a.Status != http.StatusMovedPermanently {
		t.Errorf("Status = %d, want %d", a.Status, http.StatusMovedPermanently)
	}
}

func TestRunnerReportsUnreachableSide(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	live := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer live.Close()

	runner, err := NewRunner(deadURL, live.URL)
	if err != nil {
		t.Fatalf("NewRunner() err = %v, want nil", err)
	}
	if _, _, _, err := runner.Do(context.Background(), Request{Target: "/"}); err == nil {
		t.Error("Do() err = nil, want non-nil")
	}
}
