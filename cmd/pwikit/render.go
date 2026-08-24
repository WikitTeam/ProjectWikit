package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

const envSidecar = "PWIKIT_FTML_SIDECAR"

func render(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	file := fs.String("file", "", "read wikitext from this file instead of stdin")
	mode := fs.String("mode", string(renderer.ModeArticle), "wikitext mode: article, message, inline, system, system-with-modules")
	output := fs.String("output", "html", "what to produce: html, text, backlinks, code")
	dsn := fs.String("dsn", "", "PostgreSQL connection string; without it links are never resolved and includes always miss")
	dataDir := fs.String("data-dir", "", "state directory holding role icons; defaults to the directory holding the executable")
	sidecar := fs.String("sidecar", os.Getenv(envSidecar), "path to the ftml sidecar binary; without it the linked-in ftml is used")
	trace := fs.String("trace", "", "write the callback sequence to this file, or - for stderr")
	pageName := fs.String("page", "page", "page name reported to ftml")
	category := fs.String("category", "_default", "page category reported to ftml")
	domain := fs.String("domain", "example.org", "site domain reported to ftml")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if !renderer.Mode(*mode).Valid() {
		return fmt.Errorf("unknown mode %q", *mode)
	}

	source, err := readSource(*file)
	if err != nil {
		return err
	}

	ctx := context.Background()
	engine, closeEngine, err := newRenderer(*sidecar)
	if err != nil {
		return err
	}
	defer closeEngine()

	bundle, err := i18n.Load("")
	if err != nil {
		return err
	}

	store := cliRepository{}
	var vars *page.Vars
	if *dsn != "" {
		conn, err := db.Open(ctx, *dsn)
		if err != nil {
			return err
		}
		defer conn.Close()
		p, err := paths.New(*dataDir)
		if err != nil {
			return err
		}
		users := printuser.New(bundle.Localizer(i18n.DefaultLanguage), roles.FileIcons(p.Files()))
		store.data = repo.New(ctx, conn, users)
		vars, err = cliPageVars(ctx, conn, bundle.Localizer(i18n.DefaultLanguage), *category, *pageName, *domain)
		if err != nil {
			return err
		}
	}

	cb := callbacks.New(bundle.Localizer(i18n.DefaultLanguage), store)
	cb.SetPageVars(vars)
	source = page.ThisVars(source, vars)
	var handler renderer.Callbacks = cb
	var recorder *tracer
	if *trace != "" {
		recorder = &tracer{inner: cb}
		handler = recorder
	}

	info := renderer.PageInfo{Page: *pageName, Category: *category, Domain: *domain, Title: *pageName}
	if err := emit(ctx, engine, *output, source, info, handler, renderer.Mode(*mode)); err != nil {
		return err
	}
	return writeTrace(*trace, recorder)
}

// cliPageVars resolves the page being rendered to a real row when there is one,
// so %%this|x%% answers with that page rather than staying put.
func cliPageVars(ctx context.Context, conn *db.DB, loc *i18n.Localizer, category, name, domain string) (*page.Vars, error) {
	ref := name
	if category != db.DefaultCategory {
		ref = category + ":" + name
	}
	article, err := conn.ArticleByName(ctx, ref)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var siteID int64
	if site, err := conn.SiteByHosts(ctx, []string{domain}); err == nil {
		siteID = site.ID
	}
	return page.NewVars(article, nil, repo.NewVarSource(ctx, conn, siteID), loc), nil
}

func readSource(file string) (string, error) {
	if file == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func emit(ctx context.Context, engine renderer.Renderer, output, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) error {
	switch output {
	case "html":
		result, err := engine.RenderHTML(ctx, source, info, cb, mode)
		if err != nil {
			return err
		}
		fmt.Println(result.Body)
	case "text":
		result, err := engine.RenderText(ctx, source, info, cb, mode)
		if err != nil {
			return err
		}
		fmt.Println(result.Body)
	case "backlinks":
		result, err := engine.CollectBacklinks(ctx, source, info, cb, mode)
		if err != nil {
			return err
		}
		printList("included", result.IncludedPages)
		printList("linked", result.LinkedPages)
	case "code":
		parts, err := engine.CollectCodeAndHTML(ctx, source, info, cb, mode)
		if err != nil {
			return err
		}
		for _, block := range parts.Code {
			fmt.Printf("--- code (%s) ---\n%s\n", block.Language, block.Source)
		}
		for _, block := range parts.HTML {
			fmt.Printf("--- html ---\n%s\n", block)
		}
	default:
		return fmt.Errorf("unknown output %q", output)
	}
	return nil
}

func printList(label string, values []string) {
	for _, value := range values {
		fmt.Printf("%s\t%s\n", label, value)
	}
}

func writeTrace(target string, recorder *tracer) error {
	if recorder == nil {
		return nil
	}
	body := strings.Join(recorder.lines, "\n") + "\n"
	if target == "-" {
		_, err := io.WriteString(os.Stderr, body)
		return err
	}
	return os.WriteFile(target, []byte(body), 0o644)
}

type cliRepository struct {
	data *repo.Repository
}

var _ callbacks.Repository = cliRepository{}

func (r cliRepository) RenderModule(name string, params map[string]string, _ string) (string, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+params[key])
	}
	return `<div class="module">[` + name + " " + strings.Join(pairs, " ") + `]</div>`, nil
}

func (r cliRepository) RenderUser(username string, avatar bool) (string, error) {
	if r.data == nil {
		return `<span class="printuser">[` + username + `]</span>`, nil
	}
	return r.data.RenderUser(username, avatar)
}

func (r cliRepository) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	if r.data == nil {
		return nil, nil
	}
	return r.data.PageInfo(refs)
}

func (r cliRepository) IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	if r.data == nil {
		out := make([]renderer.FetchedPage, 0, len(refs))
		for _, ref := range refs {
			out = append(out, renderer.FetchedPage{FullName: ref.FullName})
		}
		return out, nil
	}
	return r.data.IncludeSources(refs)
}
