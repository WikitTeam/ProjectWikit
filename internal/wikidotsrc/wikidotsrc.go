// Package wikidotsrc reads the source of public pages on wikidot.com.
package wikidotsrc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrBadName = errors.New("wikidotsrc: not a wikidot site or page name")
	ErrNoPage  = errors.New("wikidotsrc: no such public page")
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	token     = "pwikit"
	retries   = 3
)

var (
	wikiName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	pageID   = regexp.MustCompile(`WIKIREQUEST\.info\.pageId\s*=\s*(\d+);`)
	source   = regexp.MustCompile(`(?s)<div class="page-source">(.*)</div>\s*$`)
	anchor   = regexp.MustCompile(`<a [^>]*>|</a>`)
)

type Page struct {
	ID     int64
	Source string
}

type Client struct {
	HTTP    *http.Client
	perSite int
	total   chan struct{}
	mu      sync.Mutex
	sites   map[string]chan struct{}
	base    func(wiki string) string
}

func New(perSite, total int) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		perSite: max(perSite, 1),
		total:   make(chan struct{}, max(total, 1)),
		sites:   map[string]chan struct{}{},
		base:    func(wiki string) string { return "https://" + wiki + ".wikidot.com" },
	}
}

func (c *Client) slot(ctx context.Context, wiki string) (func(), error) {
	c.mu.Lock()
	site, ok := c.sites[wiki]
	if !ok {
		site = make(chan struct{}, c.perSite)
		c.sites[wiki] = site
	}
	c.mu.Unlock()

	select {
	case site <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case c.total <- struct{}{}:
	case <-ctx.Done():
		<-site
		return nil, ctx.Err()
	}
	return func() { <-c.total; <-site }, nil
}

func (c *Client) Fetch(ctx context.Context, wiki, page string, known int64) (Page, error) {
	if !wikiName.MatchString(wiki) || page == "" || strings.ContainsAny(page, "/?#") {
		return Page{}, ErrBadName
	}
	if known > 0 {
		if src, err := c.viewSource(ctx, wiki, known); err == nil {
			return Page{ID: known, Source: src}, nil
		} else if ctx.Err() != nil {
			return Page{}, err
		}
	}
	id, err := c.pageID(ctx, wiki, page)
	if err != nil {
		return Page{}, err
	}
	src, err := c.viewSource(ctx, wiki, id)
	if err != nil {
		return Page{}, err
	}
	return Page{ID: id, Source: src}, nil
}

func (c *Client) pageID(ctx context.Context, wiki, page string) (int64, error) {
	target := c.base(wiki) + "/" + url.PathEscape(page) + "/norender/true/noredirect/true"
	var id int64
	err := c.retry(ctx, wiki, func() (bool, error) {
		req, err := c.request(ctx, http.MethodGet, wiki, target, nil)
		if err != nil {
			return false, err
		}
		body, status, err := c.do(req)
		if err != nil {
			return true, err
		}
		if status == http.StatusNotFound {
			return false, ErrNoPage
		}
		if status != http.StatusOK {
			return status >= 500, fmt.Errorf("wikidotsrc: %s answered %d", target, status)
		}
		m := pageID.FindSubmatch(body)
		if m == nil {
			return false, ErrNoPage
		}
		id, err = strconv.ParseInt(string(m[1]), 10, 64)
		return false, err
	})
	return id, err
}

func (c *Client) viewSource(ctx context.Context, wiki string, id int64) (string, error) {
	form := url.Values{
		"moduleName":     {"viewsource/ViewSourceModule"},
		"page_id":        {strconv.FormatInt(id, 10)},
		"wikidot_token7": {token},
	}
	var out string
	err := c.retry(ctx, wiki, func() (bool, error) {
		req, err := c.request(ctx, http.MethodPost, wiki, c.base(wiki)+"/ajax-module-connector.php", strings.NewReader(form.Encode()))
		if err != nil {
			return false, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		body, status, err := c.do(req)
		if err != nil {
			return true, err
		}
		if status != http.StatusOK {
			return status >= 500, fmt.Errorf("wikidotsrc: view source of %s/%d answered %d", wiki, id, status)
		}
		var answer struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Body    string `json:"body"`
		}
		if err := json.Unmarshal(body, &answer); err != nil {
			return true, fmt.Errorf("wikidotsrc: view source of %s/%d: %w", wiki, id, err)
		}
		if answer.Status != "ok" {
			return false, fmt.Errorf("%w: %s %s", ErrNoPage, answer.Status, answer.Message)
		}
		out, err = Decode(answer.Body)
		return false, err
	})
	return out, err
}

func Decode(body string) (string, error) {
	m := source.FindStringSubmatch(body)
	if m == nil {
		return "", errors.New("wikidotsrc: view source carries no page-source block")
	}
	s := strings.TrimPrefix(m[1], "\n\t")
	s = strings.ReplaceAll(s, "<br />\n", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = anchor.ReplaceAllString(s, "")
	if strings.ContainsRune(s, '<') {
		return "", errors.New("wikidotsrc: view source carries markup it should not")
	}
	return strings.TrimRight(html.UnescapeString(s), "\n\t "), nil
}

func (c *Client) request(ctx context.Context, method, wiki, target string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", c.base(wiki)+"/")
	req.AddCookie(&http.Cookie{Name: "wikidot_token7", Value: token})
	return req, nil
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	return body, resp.StatusCode, err
}

func (c *Client) retry(ctx context.Context, wiki string, call func() (bool, error)) error {
	var err error
	for attempt := 0; attempt < retries; attempt++ {
		release, slotErr := c.slot(ctx, wiki)
		if slotErr != nil {
			return slotErr
		}
		var again bool
		again, err = call()
		release()
		if err == nil || !again {
			return err
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
