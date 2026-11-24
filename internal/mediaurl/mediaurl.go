// Package mediaurl decides where a page sends readers for an uploaded file
// when the file lives in a bucket served straight to them.
package mediaurl

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

const (
	Prefix     = "/local--files/"
	TokenParam = "t"
	// Long enough that a page left open for an afternoon keeps its images.
	tokenLife = 6 * time.Hour
)

type Files interface {
	ArticleFile(ctx context.Context, siteID int64, articleRef, fileName string) (*db.ArticleFile, error)
	ArticleFileByMedia(ctx context.Context, siteID int64, articleMedia, fileMedia string) (*db.ArticleFile, error)
}

// Asked about a reader who is not signed in, since a bucket answers anyone.
type Public func(ctx context.Context, site *db.Site, articleName string) bool

type Signer struct {
	Secret []byte
}

func (s Signer) mac(siteID int64, key string, expires int64) []byte {
	h := hmac.New(sha256.New, s.Secret)
	h.Write([]byte(strconv.FormatInt(siteID, 10) + "\x00" + key + "\x00" + strconv.FormatInt(expires, 10)))
	return h.Sum(nil)[:16]
}

func (s Signer) Sign(siteID int64, key string, now time.Time) string {
	expires := now.Add(tokenLife).Unix()
	raw := make([]byte, 8, 24)
	binary.BigEndian.PutUint64(raw, uint64(expires))
	raw = append(raw, s.mac(siteID, key, expires)...)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (s Signer) Valid(siteID int64, key, token string, now time.Time) bool {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 24 || len(s.Secret) == 0 {
		return false
	}
	expires := int64(binary.BigEndian.Uint64(raw[:8]))
	if now.Unix() > expires {
		return false
	}
	return hmac.Equal(raw[8:], s.mac(siteID, key, expires))
}

func Quote(s string) string {
	return strings.NewReplacer(":", "%3A", "/", "%2F", "?", "%3F").Replace(s)
}

func MediaKey(articleMedia, fileMedia string) string {
	return storage.Join("media", Quote(articleMedia), Quote(fileMedia))
}

type Linker struct {
	Set    *storage.Set
	Signer Signer
	Files  Files
	Public Public
	Now    func() time.Time
}

var linkPattern = regexp.MustCompile(`(?:(?:https?:)?//([A-Za-z0-9.:\-]+))?/local--(files|resized-images)/([^"'\s<>?#]+)`)

// A file on a page some readers may not see keeps its address on pwikit, with a
// token proving the page was rendered for someone allowed to read it.
func (l *Linker) Rewrite(ctx context.Context, site *db.Site, html string) string {
	if l == nil || site == nil || !strings.Contains(html, "/local--") {
		return html
	}
	backend := l.Set.For(site.Slug)
	shared := l.Set.Shared
	if backend.Mode != storage.ModeDirect && shared.Mode != storage.ModeDirect {
		return html
	}
	now := time.Now()
	if l.Now != nil {
		now = l.Now()
	}
	public := map[string]bool{}
	return linkPattern.ReplaceAllStringFunc(html, func(whole string) string {
		match := linkPattern.FindStringSubmatch(whole)
		if host := match[1]; host != "" && host != site.MediaDomain && host != site.Domain {
			return whole
		}
		resized, rest := match[2] == "resized-images", match[3]
		parts := strings.Split(rest, "/")
		for i, p := range parts {
			if unescaped, err := url.PathUnescape(p); err == nil {
				parts[i] = unescaped
			}
		}
		if parts[0] == "-" && !resized {
			if to, ok := shared.URL(storage.Join("-", strings.Join(quoted(parts[1:]), "/"))); ok {
				return to
			}
			return whole
		}
		if resized {
			if len(parts) != 3 {
				return whole
			}
			parts = parts[:2]
		}
		if backend.Mode != storage.ModeDirect || len(parts) != 2 {
			return whole
		}
		f, err := l.Files.ArticleFile(ctx, site.ID, parts[0], parts[1])
		if err != nil {
			if f, err = l.Files.ArticleFileByMedia(ctx, site.ID, parts[0], parts[1]); err != nil {
				return whole
			}
		}
		key := MediaKey(f.ArticleMediaName, f.MediaName)
		open, seen := public[f.ArticleName]
		if !seen {
			open = l.Public(ctx, site, f.ArticleName)
			public[f.ArticleName] = open
		}
		if open && resized {
			return whole
		}
		if open {
			if to, ok := backend.URL(key); ok {
				return to
			}
			return whole
		}
		return whole + "?" + TokenParam + "=" + l.Signer.Sign(site.ID, key, now)
	})
}

func quoted(parts []string) []string {
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = Quote(p)
	}
	return out
}
