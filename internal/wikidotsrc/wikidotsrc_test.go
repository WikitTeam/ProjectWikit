package wikidotsrc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name, body, want string
	}{
		{"lines and spaces",
			"<h1>x</h1>\n\n<div class=\"page-source\">\n\t[[module css]]<br />\n.a {<br />\n&nbsp;&nbsp;b: c;<br />\n}\n</div>",
			"[[module css]]\n.a {\n  b: c;\n}"},
		{"include link",
			"<div class=\"page-source\">\n\t[[include <a href=\"https://x.wikidot.com/c:y\">:x:c:y</a>]]\n</div>",
			"[[include :x:c:y]]"},
		{"escaped markup",
			"<div class=\"page-source\">\n\t&lt;b&gt; &amp;amp; &quot;q&quot;\n</div>",
			"<b> &amp; \"q\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.body)
			if err != nil {
				t.Fatalf("Decode() err = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("Decode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeRefusesUnexpectedMarkup(t *testing.T) {
	if _, err := Decode(`<div class="page-source"><b>x</b></div>`); err == nil {
		t.Errorf("Decode(<b>) err = nil, want an error")
	}
	if _, err := Decode(`<p>nothing</p>`); err == nil {
		t.Errorf("Decode(no block) err = nil, want an error")
	}
}

type fakeWiki struct {
	pages    map[string]int64
	sources  map[int64]string
	inFlight atomic.Int32
	peak     atomic.Int32
	calls    atomic.Int32
	failOnce atomic.Bool
}

func (f *fakeWiki) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	n := f.inFlight.Add(1)
	defer f.inFlight.Add(-1)
	for {
		p := f.peak.Load()
		if n <= p || f.peak.CompareAndSwap(p, n) {
			break
		}
	}
	time.Sleep(20 * time.Millisecond)
	if f.failOnce.CompareAndSwap(true, false) {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if r.Method == http.MethodGet {
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), "/norender/true/noredirect/true")
		id, ok := f.pages[name]
		if !ok {
			fmt.Fprint(w, "<html>no such page</html>")
			return
		}
		fmt.Fprintf(w, "<script>WIKIREQUEST.info.pageId = %d;</script>", id)
		return
	}
	r.ParseForm()
	var id int64
	fmt.Sscan(r.PostForm.Get("page_id"), &id)
	src, ok := f.sources[id]
	if !ok {
		fmt.Fprint(w, `{"status":"no_page","message":"gone"}`)
		return
	}
	fmt.Fprintf(w, `{"status":"ok","body":"<div class=\"page-source\">\n\t%s\n</div>"}`, src)
}

func clientFor(f *fakeWiki, perSite, total int) *Client {
	srv := httptest.NewServer(f)
	c := New(perSite, total)
	c.HTTP = srv.Client()
	c.base = func(string) string { return srv.URL }
	return c
}

func TestFetch(t *testing.T) {
	f := &fakeWiki{pages: map[string]int64{"c:a": 7}, sources: map[int64]string{7: "hello"}}
	c := clientFor(f, 2, 4)
	got, err := c.Fetch(context.Background(), "w", "c:a", 0)
	if err != nil {
		t.Fatalf("Fetch() err = %v, want nil", err)
	}
	if got.ID != 7 || got.Source != "hello" {
		t.Errorf("Fetch() = %+v, want {7 hello}", got)
	}
}

func TestFetchKnownIDSkipsThePage(t *testing.T) {
	f := &fakeWiki{pages: map[string]int64{}, sources: map[int64]string{7: "hello"}}
	c := clientFor(f, 2, 4)
	got, err := c.Fetch(context.Background(), "w", "c:a", 7)
	if err != nil || got.Source != "hello" {
		t.Fatalf("Fetch(known 7) = %+v, %v, want hello", got, err)
	}
	if n := f.calls.Load(); n != 1 {
		t.Errorf("Fetch(known 7) made %d calls, want 1", n)
	}
}

func TestFetchStaleIDLooksThePageUpAgain(t *testing.T) {
	f := &fakeWiki{pages: map[string]int64{"c:a": 8}, sources: map[int64]string{8: "new"}}
	c := clientFor(f, 2, 4)
	got, err := c.Fetch(context.Background(), "w", "c:a", 7)
	if err != nil || got.ID != 8 || got.Source != "new" {
		t.Errorf("Fetch(stale 7) = %+v, %v, want {8 new}", got, err)
	}
}

func TestFetchMissingPage(t *testing.T) {
	c := clientFor(&fakeWiki{pages: map[string]int64{}}, 2, 4)
	if _, err := c.Fetch(context.Background(), "w", "nope", 0); !errors.Is(err, ErrNoPage) {
		t.Errorf("Fetch(nope) err = %v, want ErrNoPage", err)
	}
}

func TestFetchRefusedSourceIsNoPage(t *testing.T) {
	f := &fakeWiki{pages: map[string]int64{"a": 1}, sources: map[int64]string{}}
	c := clientFor(f, 2, 4)
	if _, err := c.Fetch(context.Background(), "w", "a", 0); !errors.Is(err, ErrNoPage) {
		t.Errorf("Fetch(private) err = %v, want ErrNoPage", err)
	}
}

func TestFetchRetriesAServerError(t *testing.T) {
	f := &fakeWiki{pages: map[string]int64{"a": 1}, sources: map[int64]string{1: "x"}}
	f.failOnce.Store(true)
	c := clientFor(f, 2, 4)
	if got, err := c.Fetch(context.Background(), "w", "a", 0); err != nil || got.Source != "x" {
		t.Errorf("Fetch() after one 502 = %+v, %v, want x", got, err)
	}
}

func TestFetchRefusesBadNames(t *testing.T) {
	c := New(1, 1)
	for _, wiki := range []string{"", "evil.com", "a/b", "A_B"} {
		if _, err := c.Fetch(context.Background(), wiki, "p", 0); !errors.Is(err, ErrBadName) {
			t.Errorf("Fetch(%q) err = %v, want ErrBadName", wiki, err)
		}
	}
	if _, err := c.Fetch(context.Background(), "w", "a/../b", 0); !errors.Is(err, ErrBadName) {
		t.Errorf("Fetch(page a/../b) err = %v, want ErrBadName", err)
	}
}

func TestFetchHoldsThePerSiteLimit(t *testing.T) {
	pages := map[string]int64{}
	sources := map[int64]string{}
	for i := range 20 {
		pages[fmt.Sprint("p", i)] = int64(i + 1)
		sources[int64(i+1)] = "x"
	}
	f := &fakeWiki{pages: pages, sources: sources}
	c := clientFor(f, 3, 10)
	done := make(chan struct{})
	for name := range pages {
		go func() { c.Fetch(context.Background(), "w", name, 0); done <- struct{}{} }()
	}
	for range pages {
		<-done
	}
	if p := f.peak.Load(); p > 3 {
		t.Errorf("peak requests to one site = %d, want at most 3", p)
	}
}
