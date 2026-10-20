// Package mirror keeps copies of pages on other Wikidot wikis that local pages include.
package mirror

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikidotsrc"
)

const maxDepth = 5

var (
	includeRef = regexp.MustCompile(`(?i)\[\[\s*include\s+:([a-z0-9-]+):([^\s|\]]+)`)
	inert      = regexp.MustCompile(`(?is)\[\[\s*code\b[^\]]*\]\].*?\[\[/\s*code\s*\]\]|\[!--.*?--\]|@@[^\n]*?@@`)
)

type Target struct {
	Wiki string
	Page string
}

func (t Target) String() string { return ":" + t.Wiki + ":" + t.Page }

func Key(wiki, page string) Target {
	return Target{Wiki: strings.ToLower(strings.TrimSpace(wiki)), Page: unixName(page)}
}

func unixName(page string) string {
	b := []byte(wikidot.Normalize(page))
	for i, c := range b {
		if c == '_' && i > 0 && b[i-1] != ':' {
			b[i] = '-'
		}
	}
	return wikidot.Normalize(string(b))
}

func Refs(source string) []Target {
	seen := map[Target]bool{}
	var out []Target
	for _, m := range includeRef.FindAllStringSubmatch(inert.ReplaceAllString(source, ""), -1) {
		t := Key(m[1], m[2])
		if t.Page == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

type Summary struct {
	Pages    int
	Refs     int
	Instance int
	Mirrored int
	Missing  int
}

func Scan(ctx context.Context, d *db.DB, siteID int64) (Summary, error) {
	var sum Summary
	sources, err := d.SiteSources(ctx, siteID)
	if err != nil {
		return sum, err
	}
	local, err := d.InstanceSiteSlugs(ctx)
	if err != nil {
		return sum, err
	}
	var refs []db.IncludeRef
	for id, source := range sources {
		found := Refs(source)
		if len(found) > 0 {
			sum.Pages++
		}
		for _, t := range found {
			refs = append(refs, db.IncludeRef{ArticleID: id, Wiki: t.Wiki, Page: t.Page})
		}
	}
	if err := d.ReplaceIncludeRefs(ctx, siteID, refs); err != nil {
		return sum, err
	}
	sum.Refs = len(refs)
	targets := map[Target]bool{}
	for _, r := range refs {
		targets[Target{r.Wiki, r.Page}] = true
	}
	for t := range targets {
		switch {
		case local[t.Wiki]:
			sum.Instance++
		default:
			if _, ok, err := d.MirrorSource(ctx, t.Wiki, t.Page); err != nil {
				return sum, err
			} else if ok {
				sum.Mirrored++
			} else {
				sum.Missing++
			}
		}
	}
	return sum, nil
}

type Progress struct {
	Done    int
	Total   int
	Failed  int
	Current string
}

type Result struct {
	Fetched int
	Failed  int
}

func Fetch(ctx context.Context, d *db.DB, client *wikidotsrc.Client, targets []Target, report func(Progress)) (Result, error) {
	var res Result
	local, err := d.InstanceSiteSlugs(ctx)
	if err != nil {
		return res, err
	}
	seen := map[Target]bool{}
	var mu sync.Mutex
	progress := Progress{}
	tell := func() {
		if report != nil {
			report(progress)
		}
	}

	level := dedupe(targets, seen)
	for depth := 0; depth < maxDepth && len(level) > 0; depth++ {
		mu.Lock()
		progress.Total += len(level)
		tell()
		mu.Unlock()

		var next []Target
		var wg sync.WaitGroup
		for _, t := range level {
			wg.Add(1)
			go func() {
				defer wg.Done()
				nested, ok := fetchOne(ctx, d, client, t)
				mu.Lock()
				defer mu.Unlock()
				progress.Done++
				progress.Current = t.String()
				if ok {
					res.Fetched++
				} else {
					res.Failed++
					progress.Failed++
				}
				for _, n := range nested {
					if local[n.Wiki] || seen[n] {
						continue
					}
					if _, stored, _ := d.MirrorSource(ctx, n.Wiki, n.Page); stored {
						continue
					}
					seen[n] = true
					next = append(next, n)
				}
				tell()
			}()
		}
		wg.Wait()
		if err := ctx.Err(); err != nil {
			return res, err
		}
		level = next
	}
	return res, nil
}

func dedupe(targets []Target, seen map[Target]bool) []Target {
	var out []Target
	for _, t := range targets {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func fetchOne(ctx context.Context, d *db.DB, client *wikidotsrc.Client, t Target) ([]Target, bool) {
	var known int64
	if m, err := d.MirrorByName(ctx, t.Wiki, t.Page); err == nil && m.RemoteID != nil {
		known = *m.RemoteID
	}
	page, err := client.Fetch(ctx, t.Wiki, t.Page, known)
	now := time.Now()
	if err != nil {
		status := db.MirrorFailed
		if errors.Is(err, wikidotsrc.ErrNoPage) || errors.Is(err, wikidotsrc.ErrBadName) {
			status = db.MirrorMissing
		}
		d.SaveMirror(ctx, db.Mirror{Wiki: t.Wiki, Page: t.Page, Status: status, Problem: err.Error(), FetchedAt: now})
		return nil, false
	}
	id := page.ID
	if err := d.SaveMirror(ctx, db.Mirror{Wiki: t.Wiki, Page: t.Page, RemoteID: &id, Source: page.Source,
		Status: db.MirrorOK, FetchedAt: now}); err != nil {
		return nil, false
	}
	return Refs(page.Source), true
}

func Missing(ctx context.Context, d *db.DB, siteID *int64) ([]Target, error) {
	rows, err := d.MissingIncludes(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out := make([]Target, 0, len(rows))
	for _, r := range rows {
		out = append(out, Target{Wiki: r.Wiki, Page: r.Page})
	}
	return out, nil
}

type Job struct {
	Started  time.Time
	Finished time.Time
	Running  bool
	Progress Progress
	Result   Result
	Err      string
}

type Runner struct {
	DB     *db.DB
	Client *wikidotsrc.Client

	mu  sync.Mutex
	job Job
}

func NewRunner(d *db.DB, client *wikidotsrc.Client) *Runner {
	return &Runner{DB: d, Client: client}
}

func (r *Runner) Status() Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.job
}

func (r *Runner) Start(targets []Target) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job.Running {
		return false
	}
	r.job = Job{Started: time.Now(), Running: true}
	go func() {
		res, err := Fetch(context.Background(), r.DB, r.Client, targets, func(p Progress) {
			r.mu.Lock()
			r.job.Progress = p
			r.mu.Unlock()
		})
		r.mu.Lock()
		defer r.mu.Unlock()
		r.job.Running = false
		r.job.Finished = time.Now()
		r.job.Result = res
		if err != nil {
			r.job.Err = err.Error()
		}
	}()
	return true
}
