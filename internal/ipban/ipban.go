// Package ipban turns away the addresses a site has banned.
package ipban

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/proxyheader"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/static"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func Parse(raw string) (netip.Prefix, bool) {
	raw = strings.TrimSpace(raw)
	var prefix netip.Prefix
	if strings.Contains(raw, "/") {
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			return netip.Prefix{}, false
		}
		prefix = p
	} else {
		a, err := netip.ParseAddr(raw)
		if err != nil || a.Zone() != "" {
			return netip.Prefix{}, false
		}
		prefix = netip.PrefixFrom(a, a.BitLen())
	}
	if addr := prefix.Addr(); addr.Is4In6() && prefix.Bits() >= 96 {
		prefix = netip.PrefixFrom(addr.Unmap(), prefix.Bits()-96)
	}
	if prefix.Bits() == 0 {
		return netip.Prefix{}, false
	}
	return prefix.Masked(), true
}

type Deps struct {
	DB     *db.DB
	Trust  *proxyheader.Trust
	Bundle *i18n.Bundle
	Assets *static.Assets
	Log    *slog.Logger
}

// Reading stays open and only changes are refused. Signing in is not one of
// them, so that a superuser on a banned address can still get in.
func Middleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !guarded(r) {
				next.ServeHTTP(w, r)
				return
			}
			banned, err := d.banned(r)
			if err != nil {
				d.logger().Error("check address ban", "path", r.URL.Path, "err", err)
				next.ServeHTTP(w, r)
				return
			}
			if !banned {
				next.ServeHTTP(w, r)
				return
			}
			d.refuse(w, r)
		})
	}
}

const loginPath = "/-/login"

func guarded(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return r.URL.Path != loginPath
}

func (d Deps) logger() *slog.Logger {
	if d.Log == nil {
		return slog.Default()
	}
	return d.Log
}

func (d Deps) banned(r *http.Request) (bool, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return false, nil
	}
	addr, ok := d.Trust.ClientIP(r)
	if !ok {
		return false, nil
	}
	now := time.Now()
	banned, err := d.DB.IPBanned(ctx, current.ID, addr, now)
	if err != nil || !banned {
		return false, err
	}
	exempt, err := d.exempt(ctx, now)
	return !exempt, err
}

func (d Deps) exempt(ctx context.Context, now time.Time) (bool, error) {
	user := auth.FromContext(ctx)
	if user == nil {
		return false, nil
	}
	if user.IsSuperuser {
		return true, nil
	}
	subject, err := repo.NewPerms(ctx, d.DB).Subject(user, now)
	if err != nil {
		return false, err
	}
	return perms.Resolve(subject, nil).Has(perms.BanMembers), nil
}

func (d Deps) refuse(w http.ResponseWriter, r *http.Request) {
	loc := d.Bundle.For(r.Context())
	message := loc.T("ip-banned")
	if strings.HasPrefix(r.URL.Path, "/pw-api/") {
		body := "{" + wikijson.String("error") + ": " + wikijson.String(message) + "}"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, body)
		return
	}
	renderer := shell.New(loc, d.Assets)
	heading := loc.T("ip-banned-title")
	content, err := renderer.Notice(shell.Notice{Heading: heading, Body: message, LinkURL: "/", LinkText: loc.T("system.back-home")})
	var out strings.Builder
	if err == nil {
		err = renderer.SystemPage(&out, shell.System{Title: heading, BodyClass: "wikit-page", Content: content})
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, out.String())
	}
}
