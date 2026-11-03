package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	extism "github.com/extism/go-sdk"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	mostKeyBytes   = 200
	mostValueBytes = 64 << 10
	mostKeys       = 10000
)

type callKey struct{}

// callInfo is what a host function learns about who is calling. It travels in
// the context because one compiled plugin serves every site.
type callInfo struct {
	plugin string
	site   *db.Site
}

func callOf(ctx context.Context) *callInfo {
	c, _ := ctx.Value(callKey{}).(*callInfo)
	return c
}

// Each plugin gets only the functions its manifest asked for, so one that never
// declared a permission cannot even link against the call.
func (h *Host) hostFunctions(m *Manifest) []extism.HostFunction {
	var out []extism.HostFunction
	if m.Has(PermKV) {
		out = append(out,
			stringFunc("kv_get", 1, func(ctx context.Context, c *callInfo, args []string) (string, bool) {
				value, err := h.db.PluginKVGet(ctx, c.site.ID, c.plugin, args[0])
				if err != nil {
					return "", false
				}
				return string(value), true
			}),
			stringFunc("kv_set", 2, func(ctx context.Context, c *callInfo, args []string) (string, bool) {
				if len(args[0]) == 0 || len(args[0]) > mostKeyBytes || len(args[1]) > mostValueBytes {
					return "", false
				}
				if _, err := h.db.PluginKVGet(ctx, c.site.ID, c.plugin, args[0]); errors.Is(err, db.ErrNotFound) {
					if n, err := h.db.PluginKVCount(ctx, c.site.ID, c.plugin); err != nil || n >= mostKeys {
						return "", false
					}
				}
				if err := h.db.PluginKVSet(ctx, c.site.ID, c.plugin, args[0], []byte(args[1])); err != nil {
					h.log.Error("plugin kv_set", "plugin", c.plugin, "err", err)
				}
				return "", false
			}),
			stringFunc("kv_delete", 1, func(ctx context.Context, c *callInfo, args []string) (string, bool) {
				if err := h.db.PluginKVDelete(ctx, c.site.ID, c.plugin, args[0]); err != nil {
					h.log.Error("plugin kv_delete", "plugin", c.plugin, "err", err)
				}
				return "", false
			}),
		)
	}
	if m.Has(PermPagesRead) {
		out = append(out, stringFunc("page_get", 1, func(ctx context.Context, c *callInfo, args []string) (string, bool) {
			raw, err := h.publicPage(ctx, c.site, args[0])
			if err != nil {
				if !errors.Is(err, db.ErrNotFound) {
					h.log.Error("plugin page_get", "plugin", c.plugin, "err", err)
				}
				return "", false
			}
			return raw, true
		}))
	}
	return out
}

// A result of false hands the plugin a zero offset.
func stringFunc(name string, argc int, fn func(ctx context.Context, c *callInfo, args []string) (string, bool)) extism.HostFunction {
	params := make([]extism.ValueType, argc)
	for i := range params {
		params[i] = extism.ValueTypePTR
	}
	returns := []extism.ValueType{}
	if name == "kv_get" || name == "page_get" {
		returns = []extism.ValueType{extism.ValueTypePTR}
	}
	return extism.NewHostFunctionWithStack(name, func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		c := callOf(ctx)
		args := make([]string, argc)
		for i := range args {
			if s, err := p.ReadString(stack[i]); err == nil {
				args[i] = s
			}
		}
		result, ok := "", false
		if c != nil && c.site != nil {
			result, ok = fn(ctx, c, args)
		}
		if len(returns) == 0 {
			return
		}
		stack[0] = 0
		if ok {
			if offset, err := p.WriteString(result); err == nil {
				stack[0] = offset
			}
		}
	}, params, returns)
}

// A plugin reads a page the way an anonymous visitor would, since its output
// reaches readers who may hold fewer rights than whoever triggered it.
func (h *Host) publicPage(ctx context.Context, site *db.Site, name string) (string, error) {
	article, err := h.db.ArticleByName(ctx, site.ID, wikidot.Normalize(name))
	if err != nil {
		return "", err
	}
	perm := repo.NewPermsOn(ctx, h.db, site)
	subject, err := perm.Subject(nil, time.Now())
	if err != nil {
		return "", err
	}
	object, err := perm.Article(article, nil)
	if err != nil {
		return "", err
	}
	if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
		return "", db.ErrNotFound
	}
	tags, err := h.db.ArticleTags(ctx, article.ID)
	if err != nil {
		return "", err
	}
	authors, err := h.db.ArticleAuthors(ctx, article.ID)
	if err != nil {
		return "", err
	}
	stats, err := h.db.VoteStats(ctx, article.ID)
	if err != nil {
		return "", err
	}
	names := make([]string, len(authors))
	for i, a := range authors {
		names[i] = a.Username
	}
	raw, err := json.Marshal(map[string]any{
		"name":       article.FullName(),
		"title":      article.DisplayName(),
		"tags":       tags,
		"authors":    names,
		"rating":     stats.Sum,
		"votes":      stats.Count,
		"created_at": article.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at": article.UpdatedAt.UTC().Format(time.RFC3339),
	})
	return string(raw), err
}
