// Package mount hands a path prefix to another program on this machine, and
// tells that program who the reader is.
package mount

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

// UserHeader carries the signed reader. Whatever the client sent under this
// name, or under any other X-Pwikit- header, is dropped before forwarding.
const UserHeader = "X-Pwikit-User"

// A signed reader is good for this long, which is enough for one hop on the
// same machine and too short to be worth replaying.
const userFor = time.Minute

type Config struct {
	Path        string
	Upstream    string
	StripPrefix bool
	Secret      string
}

type identity struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Site      string `json:"site"`
	Superuser bool   `json:"superuser,omitempty"`
	Expires   int64  `json:"exp"`
}

func New(c Config, log *slog.Logger) (http.Handler, error) {
	if !strings.HasPrefix(c.Path, "/") || !strings.HasSuffix(c.Path, "/") || c.Path == "/" {
		return nil, fmt.Errorf("mount path %q has to start and end with / and name something", c.Path)
	}
	if strings.HasPrefix(c.Path, "/-/") || strings.HasPrefix(c.Path, "/pw-api/") {
		return nil, fmt.Errorf("mount path %q is inside a path pwikit answers itself", c.Path)
	}
	target, err := url.Parse(c.Upstream)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return nil, fmt.Errorf("mount upstream %q is not an http or https address", c.Upstream)
	}
	if c.Secret == "" {
		return nil, errors.New("mount needs a secret to sign the reader with")
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
			in := pr.In
			if c.StripPrefix {
				rest := "/" + strings.TrimPrefix(in.URL.Path, c.Path)
				pr.Out.URL.Path = singleJoin(target.Path, rest)
				pr.Out.URL.RawPath = ""
			}
			pr.Out.Header.Set("X-Forwarded-Prefix", strings.TrimSuffix(c.Path, "/"))
			for name := range pr.Out.Header {
				if strings.HasPrefix(http.CanonicalHeaderKey(name), "X-Pwikit-") {
					pr.Out.Header.Del(name)
				}
			}
			dropCookies(pr.Out)
			if value := sign(c.Secret, in); value != "" {
				pr.Out.Header.Set(UserHeader, value)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("mount upstream", "path", c.Path, "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		},
	}
	return proxy, nil
}

// The session and token cookies stay with pwikit. Handing them on would let the
// other program act as the reader everywhere else on the site.
func dropCookies(r *http.Request) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")
	for _, c := range cookies {
		if c.Name == session.CookieName || c.Name == csrf.CookieName {
			continue
		}
		r.AddCookie(c)
	}
}

func sign(secret string, r *http.Request) string {
	user := auth.FromContext(r.Context())
	if user == nil {
		return ""
	}
	id := identity{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.DisplayLabel(),
		Superuser: user.IsSuperuser,
		Expires:   time.Now().Add(userFor).Unix(),
	}
	if current := site.FromContext(r.Context()); current != nil {
		id.Site = current.Slug
	}
	raw, err := json.Marshal(id)
	if err != nil {
		return ""
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	return body + "." + Signature(secret, body)
}

func Signature(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func singleJoin(a, b string) string {
	switch {
	case a == "" || a == "/":
		return b
	case strings.HasSuffix(a, "/"):
		return a + strings.TrimPrefix(b, "/")
	default:
		return a + b
	}
}
