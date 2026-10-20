// Package wikidotclient signs in to Wikidot and sends private messages as that account.
package wikidotclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBase = "https://www.wikidot.com"

	originSiteID = "648902"
	tokenCookie  = "wikidot_token7"
	timeout      = 15 * time.Second
	loginPause   = 10 * time.Second
)

var (
	ErrLogin       = errors.New("wikidotclient: wikidot refused the sign-in")
	ErrPaused      = errors.New("wikidotclient: signing in again after a failure has to wait")
	ErrUnavailable = errors.New("wikidotclient: wikidot did not answer")
	ErrNoUser      = errors.New("wikidotclient: no such wikidot user")
	ErrRefused     = errors.New("wikidotclient: wikidot refused the message")
)

type Client struct {
	base     string
	username string
	password string
	http     *http.Client
	now      func() time.Time

	mu         sync.Mutex
	signedIn   bool
	failedAt   time.Time
	failedWith error
}

func New(base, username, password string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		base:     strings.TrimRight(base, "/"),
		username: username,
		password: password,
		http:     &http.Client{Jar: jar, Timeout: timeout},
		now:      time.Now,
	}
}

func (c *Client) Username() string { return c.username }

func (c *Client) LookUp(ctx context.Context, name string) (int64, error) {
	body, err := c.get(ctx, "/quickmodule.php?module=UserLookupQModule&q="+url.QueryEscape(name))
	if err != nil {
		return 0, err
	}
	var answer struct {
		Users json.RawMessage `json:"users"`
	}
	if json.Unmarshal(body, &answer) != nil {
		return 0, fmt.Errorf("%w: user lookup answered %q", ErrUnavailable, clip(body))
	}
	var users []struct {
		Name string          `json:"name"`
		ID   json.RawMessage `json:"user_id"`
	}
	_ = json.Unmarshal(answer.Users, &users)

	want := strings.TrimSpace(name)
	var folded []int
	for i, u := range users {
		if u.Name == want {
			return parseID(u.ID)
		}
		if strings.EqualFold(strings.TrimSpace(u.Name), want) {
			folded = append(folded, i)
		}
	}
	if len(folded) == 1 {
		return parseID(users[folded[0]].ID)
	}
	return 0, ErrNoUser
}

func parseID(raw json.RawMessage) (int64, error) {
	text := strings.Trim(string(raw), `"`)
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrNoUser
	}
	return id, nil
}

func (c *Client) Send(ctx context.Context, to int64, subject, source string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.signedIn {
		if err := c.login(ctx); err != nil {
			return err
		}
	}
	status, raw, err := c.post(ctx, to, subject, source)
	if err != nil {
		return err
	}
	if status == "ok" {
		return nil
	}
	still, err := c.stillSignedIn(ctx)
	if err != nil {
		return err
	}
	if !still {
		c.signedIn = false
		if err := c.login(ctx); err != nil {
			return err
		}
		status, raw, err = c.post(ctx, to, subject, source)
		if err != nil {
			return err
		}
		if status == "ok" {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrRefused, clip(raw))
}

func (c *Client) login(ctx context.Context) error {
	if !c.failedAt.IsZero() && c.now().Sub(c.failedAt) < loginPause {
		return fmt.Errorf("%w: %w", ErrPaused, c.failedWith)
	}
	err := c.signIn(ctx)
	if err != nil {
		c.failedAt, c.failedWith = c.now(), err
		return err
	}
	c.failedAt, c.failedWith = time.Time{}, nil
	c.signedIn = true
	return nil
}

func (c *Client) signIn(ctx context.Context) error {
	form := url.Values{
		"login":        {c.username},
		"password":     {c.password},
		"originSiteId": {originSiteID},
		"action":       {"Login2Action"},
		"event":        {"login"},
	}
	if _, err := c.postForm(ctx, "/default--flow/login__LoginPopupScreen", form); err != nil {
		return err
	}
	still, err := c.stillSignedIn(ctx)
	if err != nil {
		return err
	}
	if !still {
		return ErrLogin
	}
	return nil
}

func (c *Client) stillSignedIn(ctx context.Context) (bool, error) {
	page, err := c.get(ctx, "/account/activity")
	if err != nil {
		return false, err
	}
	return !strings.Contains(string(page), "Sign in") && !strings.Contains(string(page), "Login"), nil
}

func (c *Client) post(ctx context.Context, to int64, subject, source string) (string, []byte, error) {
	form := url.Values{
		"source":         {source},
		"subject":        {subject},
		"to_user_id":     {strconv.FormatInt(to, 10)},
		"action":         {"DashboardMessageAction"},
		"event":          {"send"},
		"moduleName":     {"Empty"},
		"callbackIndex":  {"1"},
		"wikidot_token7": {c.token()},
	}
	body, err := c.postForm(ctx, "/ajax-module-connector.php", form)
	if err != nil {
		return "", nil, err
	}
	var answer struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(body, &answer) != nil {
		return "", nil, fmt.Errorf("%w: message call answered %q", ErrUnavailable, clip(body))
	}
	return answer.Status, body, nil
}

func (c *Client) token() string {
	base, _ := url.Parse(c.base)
	for _, cookie := range c.http.Jar.Cookies(base) {
		if cookie.Name == tokenCookie && cookie.Value != "" {
			return cookie.Value
		}
	}
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	value := hex.EncodeToString(buf)
	c.http.Jar.SetCookies(base, []*http.Cookie{{Name: tokenCookie, Value: value, Path: "/"}})
	return value
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) postForm(ctx context.Context, path string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: %s answered %s", ErrUnavailable, req.URL.Path, resp.Status)
	}
	return body, nil
}

func clip(body []byte) string {
	const limit = 300
	if len(body) > limit {
		return string(body[:limit]) + "..."
	}
	return string(body)
}
