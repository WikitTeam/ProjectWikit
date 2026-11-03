// Package plugin runs the wasm plugins an instance has installed, each enabled
// site by site.
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

// A plugin built for another API version is skipped rather than loaded into
// calls it would misread.
const API = 1

const (
	PermKV        = "kv"
	PermPagesRead = "pages.read"
	PermHTTP      = "http"
)

var knownPerms = []string{PermKV, PermPagesRead, PermHTTP}

var Events = []string{"page.created", "page.edited", "page.deleted", "vote.changed", "post.created"}

const (
	defaultPages   = 256
	mostPages      = 1024
	defaultTimeout = time.Second
	mostTimeout    = 5 * time.Second
	enabledFor     = 30 * time.Second
	idleInstances  = 2
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,39}$`)

type ModuleSpec struct {
	Name    string `json:"name"`
	Content bool   `json:"content"`
}

type Manifest struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	API          int          `json:"api"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Modules      []ModuleSpec `json:"modules"`
	Events       []string     `json:"events"`
	Permissions  []string     `json:"permissions"`
	AllowedHosts []string     `json:"allowed_hosts"`
	ProfileTitle string       `json:"profile_title"`
	MemoryPages  uint32       `json:"memory_pages"`
	TimeoutMS    int          `json:"timeout_ms"`
}

func (m *Manifest) Has(perm string) bool { return slices.Contains(m.Permissions, perm) }

type Plugin struct {
	Manifest
	compiled *extism.CompiledPlugin
	timeout  time.Duration
	idle     chan *extism.Plugin
}

// A nil Host has no plugins, so an instance without a plugins directory runs
// exactly as before.
type Host struct {
	db      *db.DB
	log     *slog.Logger
	plugins map[string]*Plugin
	modules map[string]*Plugin

	mu      sync.Mutex
	enabled map[int64]enabledSet
}

type enabledSet struct {
	names []string
	at    time.Time
}

// A broken plugin is logged and left out rather than keeping the wiki from
// starting.
func Load(ctx context.Context, dir string, d *db.DB, log *slog.Logger, builtin func(string) bool) (*Host, error) {
	h := &Host{db: d, log: log, plugins: map[string]*Plugin{}, modules: map[string]*Plugin{}, enabled: map[int64]enabledSet{}}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return h, nil
	} else if err != nil {
		return nil, fmt.Errorf("read plugins in %s: %w", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := h.compile(ctx, filepath.Join(dir, e.Name()))
		if err != nil {
			log.Error("skip plugin", "dir", e.Name(), "err", err)
			continue
		}
		if _, taken := h.plugins[p.Name]; taken {
			log.Error("skip plugin", "dir", e.Name(), "err", "another plugin has the same name")
			continue
		}
		clash := ""
		for _, m := range p.Modules {
			if builtin(m.Name) || h.modules[m.Name] != nil {
				clash = m.Name
			}
		}
		if clash != "" {
			log.Error("skip plugin", "dir", e.Name(), "err", fmt.Sprintf("module %q is already taken", clash))
			continue
		}
		h.plugins[p.Name] = p
		for _, m := range p.Modules {
			h.modules[m.Name] = p
		}
		log.Info("plugin loaded", "name", p.Name, "version", p.Version)
	}
	return h, nil
}

func readManifest(dir string) (Manifest, error) {
	var m Manifest
	raw, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, fmt.Errorf("parse plugin.json: %w", err)
	}
	switch {
	case !namePattern.MatchString(m.Name):
		return m, fmt.Errorf("name %q is not lowercase letters, digits, - or _", m.Name)
	case m.API != API:
		return m, fmt.Errorf("plugin speaks api %d, this pwikit speaks %d", m.API, API)
	}
	for i, mod := range m.Modules {
		m.Modules[i].Name = strings.ToLower(strings.TrimSpace(mod.Name))
		if !namePattern.MatchString(m.Modules[i].Name) {
			return m, fmt.Errorf("module name %q is not lowercase letters, digits, - or _", mod.Name)
		}
	}
	for _, perm := range m.Permissions {
		if !slices.Contains(knownPerms, perm) {
			return m, fmt.Errorf("unknown permission %q", perm)
		}
	}
	for _, ev := range m.Events {
		if !slices.Contains(Events, ev) {
			return m, fmt.Errorf("unknown event %q", ev)
		}
	}
	if len(m.AllowedHosts) > 0 && !m.Has(PermHTTP) {
		return m, errors.New("allowed_hosts needs the http permission")
	}
	return m, nil
}

// Settings is what the instance's operator gives a plugin, kept beside it in
// config.json. The plugin's author cannot know the addresses on this machine,
// so the operator's hosts replace the ones the manifest names.
type Settings struct {
	Config       map[string]string `json:"config"`
	AllowedHosts []string          `json:"allowed_hosts"`
}

func readSettings(dir string) (Settings, error) {
	var s Settings
	raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	} else if err != nil {
		return s, err
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("parse config.json: %w", err)
	}
	return s, nil
}

func (h *Host) compile(ctx context.Context, dir string) (*Plugin, error) {
	m, err := readManifest(dir)
	if err != nil {
		return nil, err
	}
	settings, err := readSettings(dir)
	if err != nil {
		return nil, err
	}
	if len(settings.AllowedHosts) > 0 {
		if !m.Has(PermHTTP) {
			return nil, errors.New("config.json sets allowed_hosts but the plugin has no http permission")
		}
		m.AllowedHosts = settings.AllowedHosts
	}
	pages := m.MemoryPages
	if pages == 0 {
		pages = defaultPages
	}
	pages = min(pages, mostPages)
	timeout := time.Duration(m.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	timeout = min(timeout, mostTimeout)
	manifest := extism.Manifest{
		Wasm:   []extism.Wasm{extism.WasmFile{Path: filepath.Join(dir, "plugin.wasm")}},
		Memory: &extism.ManifestMemory{MaxPages: pages, MaxHttpResponseBytes: 1 << 20, MaxVarBytes: 1 << 20},
		Config: settings.Config,
	}
	if m.Has(PermHTTP) {
		manifest.AllowedHosts = m.AllowedHosts
	}
	cfg := extism.PluginConfig{
		EnableWasi:    true,
		RuntimeConfig: wazero.NewRuntimeConfig().WithCloseOnContextDone(true).WithMemoryLimitPages(pages),
	}
	compiled, err := extism.NewCompiledPlugin(ctx, manifest, cfg, h.hostFunctions(&m))
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", m.Name, err)
	}
	return &Plugin{Manifest: m, compiled: compiled, timeout: timeout, idle: make(chan *extism.Plugin, idleInstances)}, nil
}

func (h *Host) Close(ctx context.Context) {
	if h == nil {
		return
	}
	for _, p := range h.plugins {
		close(p.idle)
		for inst := range p.idle {
			_ = inst.Close(ctx)
		}
		_ = p.compiled.Close(ctx)
	}
}

func (h *Host) Installed() []*Plugin {
	if h == nil {
		return nil
	}
	out := make([]*Plugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (h *Host) ModuleHasContent(name string) bool {
	if h == nil {
		return false
	}
	p := h.modules[name]
	if p == nil {
		return false
	}
	for _, m := range p.Modules {
		if m.Name == name {
			return m.Content
		}
	}
	return false
}

func (h *Host) Enabled(ctx context.Context, siteID int64) ([]string, error) {
	if h == nil || len(h.plugins) == 0 {
		return nil, nil
	}
	h.mu.Lock()
	cached, ok := h.enabled[siteID]
	h.mu.Unlock()
	if ok && time.Since(cached.at) < enabledFor {
		return cached.names, nil
	}
	names, err := h.db.EnabledPlugins(ctx, siteID)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	h.enabled[siteID] = enabledSet{names: names, at: time.Now()}
	h.mu.Unlock()
	return names, nil
}

func (h *Host) SetEnabled(ctx context.Context, siteID int64, name string, on bool) error {
	if h == nil || h.plugins[name] == nil {
		return fmt.Errorf("plugin %q is not installed", name)
	}
	if err := h.db.SetPluginEnabled(ctx, siteID, name, on); err != nil {
		return err
	}
	h.mu.Lock()
	delete(h.enabled, siteID)
	h.mu.Unlock()
	return nil
}

func (h *Host) enabledOn(ctx context.Context, siteID int64, name string) bool {
	names, err := h.Enabled(ctx, siteID)
	if err != nil {
		h.log.Error("read enabled plugins", "site", siteID, "err", err)
		return false
	}
	return slices.Contains(names, name)
}

// An instance that failed is dropped, since a trapped guest may have left its
// memory half-written.
func (h *Host) call(ctx context.Context, p *Plugin, c *callInfo, export string, input []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.WithValue(ctx, callKey{}, c), p.timeout)
	defer cancel()
	var inst *extism.Plugin
	select {
	case inst = <-p.idle:
	default:
		var err error
		if inst, err = p.compiled.Instance(ctx, extism.PluginInstanceConfig{}); err != nil {
			return nil, fmt.Errorf("start plugin %s: %w", p.Name, err)
		}
	}
	if !inst.FunctionExists(export) {
		p.release(ctx, inst)
		return nil, fmt.Errorf("plugin %s has no %s export", p.Name, export)
	}
	code, out, err := inst.CallWithContext(ctx, export, input)
	if err != nil || code != 0 {
		_ = inst.Close(context.Background())
		if err == nil {
			err = fmt.Errorf("exit code %d", code)
		}
		return nil, fmt.Errorf("plugin %s %s: %w", p.Name, export, err)
	}
	p.release(ctx, inst)
	return out, nil
}

func (p *Plugin) release(ctx context.Context, inst *extism.Plugin) {
	select {
	case p.idle <- inst:
	default:
		_ = inst.Close(ctx)
	}
}
