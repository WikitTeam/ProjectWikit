package update

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
)

const boardCache = 30 * time.Second

type Board struct {
	DB              *db.DB
	Bundle          *i18n.Bundle
	Settings        Settings
	Current         string
	BundledPostgres string
	Container       bool

	mu     sync.Mutex
	state  db.UpdateState
	loaded time.Time
}

func (b *Board) State(ctx context.Context) db.UpdateState {
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Since(b.loaded) < boardCache {
		return b.state
	}
	if st, err := b.DB.UpdateState(ctx); err == nil {
		b.state = st
	}
	b.loaded = time.Now()
	return b.state
}

func (b *Board) Facts(now time.Time) Facts {
	return Facts{Current: b.Current, BundledPostgres: b.BundledPostgres, Container: b.Container, Now: now}
}

func (b *Board) Notices(ctx context.Context) []Notice {
	return Notices(b.State(ctx), b.Settings, b.Facts(time.Now()))
}

func (b *Board) change(ctx context.Context, edit func(*db.UpdateState)) error {
	st, err := b.DB.UpdateState(ctx)
	if err != nil {
		return err
	}
	edit(&st)
	if err := b.DB.SaveUpdateState(ctx, st); err != nil {
		return err
	}
	b.mu.Lock()
	b.state, b.loaded = st, time.Now()
	b.mu.Unlock()
	return nil
}

func (b *Board) Postpone(ctx context.Context) error {
	return b.change(ctx, func(st *db.UpdateState) { Postpone(st, time.Now()) })
}

func (b *Board) StartNow(ctx context.Context) error {
	return b.change(ctx, func(st *db.UpdateState) { StartNow(st, b.Facts(time.Now())) })
}

func (b *Board) Startable(ctx context.Context) bool {
	return Startable(b.State(ctx), b.Facts(time.Now()))
}

func (b *Board) Skip(ctx context.Context) error {
	return b.change(ctx, func(st *db.UpdateState) { Skip(st) })
}

func (b *Board) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		at, show := Banner(b.State(r.Context()), b.Settings, time.Now())
		if !show || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		loc := b.Bundle.Localizer(b.Bundle.Match(r.Header.Get(i18n.AcceptHeader)))
		bw := &bannerWriter{ResponseWriter: w, snippet: bannerHTML(loc, at)}
		next.ServeHTTP(bw, r)
		bw.finish()
	})
}

type bannerWriter struct {
	http.ResponseWriter
	snippet  string
	status   int
	decided  bool
	buffered bool
	body     bytes.Buffer
}

func (bw *bannerWriter) WriteHeader(status int) {
	if bw.decided {
		return
	}
	bw.decided = true
	bw.status = status
	h := bw.Header()
	html := strings.HasPrefix(h.Get("Content-Type"), "text/html") && h.Get("Content-Encoding") == ""
	if html && status == http.StatusOK || html && status >= 400 {
		bw.buffered = true
		return
	}
	bw.ResponseWriter.WriteHeader(status)
}

func (bw *bannerWriter) Write(p []byte) (int, error) {
	if !bw.decided {
		if bw.Header().Get("Content-Type") == "" {
			bw.Header().Set("Content-Type", http.DetectContentType(p))
		}
		bw.WriteHeader(http.StatusOK)
	}
	if bw.buffered {
		return bw.body.Write(p)
	}
	return bw.ResponseWriter.Write(p)
}

func (bw *bannerWriter) Flush() {
	if bw.buffered {
		return
	}
	if f, ok := bw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (bw *bannerWriter) finish() {
	if !bw.buffered {
		return
	}
	page := bw.body.Bytes()
	if i := bytes.LastIndex(bytes.ToLower(page), []byte("</body>")); i >= 0 {
		page = append(page[:i:i], append([]byte(bw.snippet), page[i:]...)...)
	}
	bw.Header().Del("Content-Length")
	bw.Header().Set("Content-Length", strconv.Itoa(len(page)))
	bw.ResponseWriter.WriteHeader(bw.status)
	_, _ = bw.ResponseWriter.Write(page)
}

const timeToken = "\x00time\x00"

// A floating layer outside the page's own markup, so a theme's layout is left
// exactly as it was. The script turns the UTC time into the reader's own zone.
func bannerHTML(loc *i18n.Localizer, at time.Time) string {
	utc := at.UTC()
	stamp := `<time datetime="` + utc.Format(time.RFC3339) + `">` + utc.Format("15:04") + ` UTC</time>`
	text := strings.Replace(escape.HTML(loc.T("update.banner", "time", timeToken)), escape.HTML(timeToken), stamp, 1)
	key := strconv.FormatInt(utc.Unix(), 10)
	return `<div id="pwikit-update-banner" role="status" data-key="` + key + `" style="position:fixed;top:0;left:0;right:0;z-index:2147483000;` +
		`margin:0;padding:8px 44px 8px 16px;background:#fff3cd;color:#664d03;border-bottom:1px solid #ffe69c;` +
		`box-shadow:0 1px 4px rgba(0,0,0,.12);font:14px/1.5 -apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;text-align:center">` +
		`<span>` + text + `</span>` +
		`<button type="button" aria-label="` + escape.HTML(loc.T("update.banner-close")) + `" style="position:absolute;right:10px;top:4px;` +
		`margin:0;padding:0 4px;background:none;border:0;color:inherit;font-size:22px;line-height:1.2;cursor:pointer">&times;</button></div>` +
		`<script>(function(){var b=document.getElementById('pwikit-update-banner');if(!b)return;` +
		`var k='pwikit-update-banner:'+b.getAttribute('data-key');` +
		`try{if(localStorage.getItem(k)){b.parentNode.removeChild(b);return;}}catch(e){}` +
		`var t=b.querySelector('time'),d=new Date(t.getAttribute('datetime'));` +
		`if(!isNaN(d.getTime())){t.textContent=d.toLocaleString([],{month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit'});}` +
		`b.querySelector('button').addEventListener('click',function(){try{localStorage.setItem(k,'1');}catch(e){}b.parentNode.removeChild(b);});` +
		`})();</script>`
}
