package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

//go:embed texts/*.txt
var texts embed.FS

const (
	mainModule = "github.com/WikitTeam/ProjectWikit"
	sourceBase = "https://github.com/WikitTeam/ProjectWikit/tree/"
	rule       = "================================================================================"
)

var rustTargets = map[string]string{
	"linux/amd64":   "x86_64-unknown-linux-gnu",
	"linux/arm64":   "aarch64-unknown-linux-gnu",
	"darwin/amd64":  "x86_64-apple-darwin",
	"darwin/arm64":  "aarch64-apple-darwin",
	"windows/amd64": "x86_64-pc-windows-gnu",
}

var licenseName = regexp.MustCompile(`(?i)^(licen[cs]e|copying|copyright|notice|patents|unlicense)([-._][a-z0-9-]+)*(\.(txt|md|markdown|rst))?$`)

var apacheText = regexp.MustCompile(`^\s*Apache License\s+Version 2\.0, January 2004`)

var skippedDirs = map[string]bool{"tests": true, "test": true, "benches": true, "examples": true, "target": true, "fuzz": true, ".git": true, "node_modules": true}

type text struct {
	name string
	body string
}

type entry struct {
	title    string
	declared string
	texts    []text
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "notices: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("notices", flag.ContinueOnError)
	goos := fs.String("goos", runtime.GOOS, "operating system of the build")
	goarch := fs.String("goarch", runtime.GOARCH, "architecture of the build")
	tags := fs.String("tags", "bundle", "build tags of the build; bundle adds the PostgreSQL notices")
	version := fs.String("version", "", "release the notices are for")
	out := fs.String("out", "NOTICE", "file the notices are written to")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	target := *goos + "/" + *goarch

	var w writer
	notice, err := os.ReadFile("NOTICE")
	if err != nil {
		return err
	}
	w.raw(strings.TrimRight(normalize(string(notice)), "\n") + "\n")
	ref := *version
	if ref == "" {
		ref = "main"
	}
	w.raw("\nSource code for this release: " + sourceBase + ref + "\n")

	rufoundation, err := os.ReadFile(filepath.Join("third_party", "RuFoundation-LICENSE"))
	if err != nil {
		return err
	}
	w.section("RuFoundation")
	w.entry(entry{title: "RuFoundation (https://github.com/scpru/rufoundation)", texts: []text{{name: "LICENSE", body: string(rufoundation)}}})

	goEntries, err := goModules(*goos, *goarch, *tags)
	if err != nil {
		return err
	}
	w.section("Go modules compiled into pwikit")
	for _, e := range goEntries {
		w.entry(e)
	}

	triple, ok := rustTargets[target]
	if !ok {
		return fmt.Errorf("no Rust target for %s", target)
	}
	rustEntries, err := rustCrates(triple)
	if err != nil {
		return err
	}
	w.section("Rust crates linked into pwikit through ftml")
	for _, e := range rustEntries {
		w.entry(e)
	}

	npmEntries, err := npmPackages(filepath.Join("static", "app.js.map"))
	if err != nil {
		return err
	}
	w.section("JavaScript packages bundled into static/app.js")
	for _, e := range npmEntries {
		w.entry(e)
	}

	if err := w.manual("Fonts, icons and images", "static.txt"); err != nil {
		return err
	}
	if hasTag(*tags, "bundle") {
		if err := w.manual("Bundled PostgreSQL", "postgresql-"+*goos+".txt"); err != nil {
			return err
		}
	}

	return os.WriteFile(*out, []byte(w.buf.String()), 0o644)
}

type writer struct {
	buf  bytes.Buffer
	seen map[string]string
}

func (w *writer) raw(s string) { w.buf.WriteString(s) }

func (w *writer) section(title string) {
	fmt.Fprintf(&w.buf, "\n\n%s\n%s\n%s\n", rule, title, rule)
}

func (w *writer) entry(e entry) {
	if w.seen == nil {
		w.seen = map[string]string{}
	}
	fmt.Fprintf(&w.buf, "\n-- %s\n", e.title)
	if e.declared != "" {
		fmt.Fprintf(&w.buf, "License: %s\n", e.declared)
	}
	if len(e.texts) == 0 {
		w.buf.WriteString("No license file was found in the package.\n")
		return
	}
	for _, t := range e.texts {
		body := strings.Trim(normalize(t.body), "\n")
		key := strings.Join(strings.Fields(body), " ")
		if apacheText.MatchString(body) {
			key = "Apache-2.0"
		}
		label := e.title
		if t.name != "" {
			label += ", " + t.name
		}
		if first, ok := w.seen[key]; ok {
			fmt.Fprintf(&w.buf, "\n[%s: same text as %s above]\n", orDefault(t.name, "text"), first)
			continue
		}
		w.seen[key] = label
		fmt.Fprintf(&w.buf, "\n%s\n", body)
	}
}

func (w *writer) manual(section, name string) error {
	data, err := texts.ReadFile("texts/" + name)
	if err != nil {
		return err
	}
	w.section(section)
	var current *entry
	flush := func() {
		if current != nil {
			w.entry(*current)
		}
	}
	for _, chunk := range strings.Split(normalize(string(data)), "\n== ") {
		chunk = strings.TrimPrefix(chunk, "== ")
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		title, body, _ := strings.Cut(chunk, "\n")
		e := entry{title: title}
		for _, part := range strings.Split(body, "\n---\n") {
			e.texts = append(e.texts, text{body: part})
		}
		current = &e
		flush()
		current = nil
	}
	return nil
}

func goModules(goos, goarch, tags string) ([]entry, error) {
	cmd := exec.Command("go", "list", "-deps", "-tags", tags, "-f",
		"{{if .Standard}}std{{else if .Module}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}\t{{.Dir}}{{end}}", "./cmd/pwikit")
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=1")
	cmd.Stderr = os.Stderr
	listed, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	type module struct {
		version string
		dir     string
		pkgDirs map[string]bool
	}
	modules := map[string]*module{}
	usesStd := false
	scanner := bufio.NewScanner(bytes.NewReader(listed))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "std" {
			usesStd = true
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 || fields[0] == mainModule {
			continue
		}
		m := modules[fields[0]]
		if m == nil {
			m = &module{version: fields[1], dir: fields[2], pkgDirs: map[string]bool{}}
			modules[fields[0]] = m
		}
		m.pkgDirs[fields[3]] = true
	}

	var out []entry
	if usesStd {
		goroot, err := exec.Command("go", "env", "GOROOT", "GOVERSION").Output()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.TrimSpace(normalize(string(goroot))), "\n")
		e := entry{title: "Go standard library and runtime " + lines[len(lines)-1]}
		e.texts = licenseFiles(lines[0], lines[0], false)
		out = append(out, e)
	}

	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m := modules[name]
		e := entry{title: name + " " + m.version}
		seen := map[string]bool{}
		add := func(files []text) {
			for _, f := range files {
				if !seen[f.name] {
					seen[f.name] = true
					e.texts = append(e.texts, f)
				}
			}
		}
		add(licenseFiles(m.dir, m.dir, false))
		dirs := make([]string, 0, len(m.pkgDirs))
		for dir := range m.pkgDirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		for _, dir := range dirs {
			for d := dir; strings.HasPrefix(d, m.dir) && d != m.dir; d = filepath.Dir(d) {
				add(licenseFiles(m.dir, d, false))
			}
		}
		out = append(out, e)
	}
	return out, nil
}

func rustCrates(triple string) ([]entry, error) {
	cmd := exec.Command("cargo", "metadata", "--format-version", "1", "--filter-platform", triple,
		"--manifest-path", filepath.Join("ftml-capi", "Cargo.toml"))
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo metadata: %w", err)
	}
	var meta struct {
		Packages []struct {
			ID           string  `json:"id"`
			Name         string  `json:"name"`
			Version      string  `json:"version"`
			License      string  `json:"license"`
			Source       *string `json:"source"`
			ManifestPath string  `json:"manifest_path"`
			Targets      []struct {
				Kind []string `json:"kind"`
			} `json:"targets"`
		} `json:"packages"`
		Resolve struct {
			Root  string `json:"root"`
			Nodes []struct {
				ID   string `json:"id"`
				Deps []struct {
					Pkg      string `json:"pkg"`
					DepKinds []struct {
						Kind *string `json:"kind"`
					} `json:"dep_kinds"`
				} `json:"deps"`
			} `json:"nodes"`
		} `json:"resolve"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	deps := map[string][]string{}
	for _, n := range meta.Resolve.Nodes {
		for _, d := range n.Deps {
			for _, k := range d.DepKinds {
				if k.Kind == nil {
					deps[n.ID] = append(deps[n.ID], d.Pkg)
					break
				}
			}
		}
	}
	procMacro := map[string]bool{}
	for _, p := range meta.Packages {
		for _, t := range p.Targets {
			for _, k := range t.Kind {
				if k == "proc-macro" {
					procMacro[p.ID] = true
				}
			}
		}
	}

	linked := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		for _, dep := range deps[id] {
			if linked[dep] || procMacro[dep] {
				continue
			}
			linked[dep] = true
			walk(dep)
		}
	}
	walk(meta.Resolve.Root)

	var out []entry
	for _, p := range meta.Packages {
		if !linked[p.ID] || p.Source == nil {
			continue
		}
		dir := filepath.Dir(p.ManifestPath)
		out = append(out, entry{title: p.Name + " " + p.Version, declared: p.License, texts: licenseFiles(dir, dir, true)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].title < out[j].title })
	return out, nil
}

func npmPackages(sourceMap string) ([]entry, error) {
	data, err := os.ReadFile(sourceMap)
	if err != nil {
		return nil, err
	}
	var m struct {
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("read %s: %w", sourceMap, err)
	}

	base := filepath.Dir(sourceMap)
	dirs := map[string]bool{}
	for _, src := range m.Sources {
		p := path.Clean(path.Join(filepath.ToSlash(base), src))
		i := strings.LastIndex(p, "node_modules/")
		if i < 0 {
			continue
		}
		rest := strings.Split(p[i+len("node_modules/"):], "/")
		name := rest[0]
		if strings.HasPrefix(name, "@") && len(rest) > 1 {
			name += "/" + rest[1]
		}
		dirs[p[:i+len("node_modules/")]+name] = true
	}

	var out []entry
	for dir := range dirs {
		var pkg struct {
			Name    string          `json:"name"`
			Version string          `json:"version"`
			License json.RawMessage `json:"license"`
		}
		manifest, err := os.ReadFile(filepath.Join(filepath.FromSlash(dir), "package.json"))
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(manifest, &pkg); err != nil {
			return nil, fmt.Errorf("read %s/package.json: %w", dir, err)
		}
		declared := ""
		if err := json.Unmarshal(pkg.License, &declared); err != nil {
			var typed struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(pkg.License, &typed) == nil {
				declared = typed.Type
			}
		}
		local := filepath.FromSlash(dir)
		out = append(out, entry{title: pkg.Name + " " + pkg.Version, declared: declared, texts: licenseFiles(local, local, false)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].title < out[j].title })
	return out, nil
}

func licenseFiles(root, dir string, recursive bool) []text {
	var out []text
	visit := func(p string, d fs.DirEntry) {
		if d.IsDir() || !licenseName.MatchString(d.Name()) {
			return
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			rel = d.Name()
		}
		out = append(out, text{name: filepath.ToSlash(rel), body: string(body)})
	}
	if !recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		for _, d := range entries {
			visit(filepath.Join(dir, d.Name()), d)
		}
		return out
	}
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && p != dir {
			depth := strings.Count(filepath.ToSlash(strings.TrimPrefix(p, dir)), "/")
			if skippedDirs[d.Name()] || depth > 3 {
				return filepath.SkipDir
			}
		}
		visit(p, d)
		return nil
	})
	return out
}

func normalize(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func hasTag(tags, want string) bool {
	for _, t := range strings.Split(tags, ",") {
		if strings.TrimSpace(t) == want {
			return true
		}
	}
	return false
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
