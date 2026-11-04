package plugin

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

const mostOutputBytes = 1 << 20

type userJSON struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type moduleCall struct {
	Module string            `json:"module"`
	Params map[string]string `json:"params"`
	Body   string            `json:"body"`
	Site   string            `json:"site"`
	Page   string            `json:"page,omitempty"`
	PageID int64             `json:"page_id,omitempty"`
	User   *userJSON         `json:"user"`
}

// What a plugin returns is wikitext and goes through the renderer like any
// other source, so a plugin cannot put raw markup on the page.
func (h *Host) RenderModule(env module.Env, name string, params map[string]string, body string) (string, bool, error) {
	if h == nil || env.Site == nil {
		return "", false, nil
	}
	p := h.modules[name]
	ctx := context.Background()
	if p == nil || !h.enabledOn(ctx, env.Site.ID, p.Name) {
		return "", false, nil
	}
	call := moduleCall{Module: name, Params: params, Body: body, Site: env.Site.Slug}
	if env.Page != nil && env.Page.Article != nil {
		call.Page = env.Page.Article.FullName()
		call.PageID = env.Page.Article.ID
	}
	if env.User != nil {
		call.User = &userJSON{ID: env.User.ID, Username: env.User.Username}
	}
	input, err := json.Marshal(call)
	if err != nil {
		return "", true, err
	}
	out, err := h.call(ctx, p, &callInfo{plugin: p.Name, site: env.Site}, "module", input)
	if err != nil {
		h.log.Error("plugin module", "module", name, "err", err)
		return "", true, &module.Error{Message: env.Text("module-plugin-failed", "name", name)}
	}
	if len(out) > mostOutputBytes {
		return "", true, &module.Error{Message: env.Text("module-plugin-failed", "name", name)}
	}
	if env.Render == nil {
		return "", true, nil
	}
	html, err := env.Render(string(out), env.Page)
	return html, true, err
}

// Events about a page an anonymous visitor may not read stay away from plugins,
// whose output anyone may read.
func (h *Host) PagePublic(ctx context.Context, site *db.Site, name string) bool {
	if h == nil || site == nil {
		return false
	}
	_, err := h.publicPage(ctx, site, name)
	return err == nil
}

// Events go out in the background. A plugin only learns of the event and
// cannot hold up or refuse the change that caused it.
func (h *Host) Emit(site *db.Site, event string, payload map[string]any) {
	if h == nil || site == nil || len(h.plugins) == 0 {
		return
	}
	go func() {
		ctx := context.Background()
		if name, ok := payload["page"].(string); ok && event != "page.deleted" && !h.PagePublic(ctx, site, name) {
			return
		}
		names, err := h.Enabled(ctx, site.ID)
		if err != nil {
			h.log.Error("read enabled plugins", "site", site.ID, "err", err)
			return
		}
		body := map[string]any{"event": event, "site": site.Slug, "at": time.Now().UTC().Format(time.RFC3339)}
		for k, v := range payload {
			body[k] = v
		}
		input, err := json.Marshal(body)
		if err != nil {
			return
		}
		for _, name := range names {
			p := h.plugins[name]
			if p == nil || !slices.Contains(p.Events, event) {
				continue
			}
			if _, err := h.call(ctx, p, &callInfo{plugin: p.Name, site: site}, "event", input); err != nil {
				h.log.Error("plugin event", "plugin", p.Name, "event", event, "err", err)
			}
		}
	}()
}

type profileCall struct {
	Site   string    `json:"site"`
	User   userJSON  `json:"user"`
	Viewer *userJSON `json:"viewer"`
}

// A plugin that fails is left out, so one broken plugin costs only its own
// block.
func (h *Host) ProfileSections(ctx context.Context, site *db.Site, owner, viewer *db.User) []module.ProfileSection {
	if h == nil || site == nil || owner == nil || len(h.plugins) == 0 {
		return nil
	}
	names, err := h.Enabled(ctx, site.ID)
	if err != nil {
		h.log.Error("read enabled plugins", "site", site.ID, "err", err)
		return nil
	}
	call := profileCall{Site: site.Slug, User: userJSON{ID: owner.ID, Username: owner.Username}}
	if viewer != nil {
		call.Viewer = &userJSON{ID: viewer.ID, Username: viewer.Username}
	}
	input, err := json.Marshal(call)
	if err != nil {
		return nil
	}
	var out []module.ProfileSection
	for _, name := range names {
		p := h.plugins[name]
		if p == nil {
			continue
		}
		if p.ProfileBadge {
			if body := h.profileCall(ctx, p, site, "badge", input); body != "" {
				out = append(out, module.ProfileSection{Plugin: p.Name, Body: body, Badge: true})
			}
		}
		if p.ProfileTitle != "" {
			if body := h.profileCall(ctx, p, site, "profile", input); body != "" {
				out = append(out, module.ProfileSection{Plugin: p.Name, Title: p.ProfileTitle, Body: body})
			}
		}
	}
	return out
}

func (h *Host) profileCall(ctx context.Context, p *Plugin, site *db.Site, export string, input []byte) string {
	body, err := h.call(ctx, p, &callInfo{plugin: p.Name, site: site}, export, input)
	if err != nil {
		h.log.Error("plugin profile", "plugin", p.Name, "export", export, "err", err)
		return ""
	}
	if len(body) > mostOutputBytes {
		return ""
	}
	return string(body)
}
