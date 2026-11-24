package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/mediaurl"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/quota"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

func openStorage(p *paths.Paths, cfg config.File) (*storage.Set, error) {
	shared, err := openBackend(p, "", cfg.Storage.StorageBackend)
	if err != nil {
		return nil, err
	}
	set := &storage.Set{Shared: shared, Sites: map[string]*storage.Backend{}}
	for slug := range cfg.Storage.Sites {
		b, err := openBackend(p, slug, cfg.Storage.Site(slug))
		if err != nil {
			return nil, err
		}
		set.Sites[slug] = b
	}
	return set, nil
}

func openBackend(p *paths.Paths, slug string, b config.StorageBackend) (*storage.Backend, error) {
	mode, _ := storage.ParseMode(b.Mode)
	name := "[storage]"
	if slug != "" {
		name = "[storage.sites." + slug + "]"
	}
	if mode == storage.ModeLocal {
		return &storage.Backend{Name: name, Store: storage.NewLocal(p.Files()), Mode: mode}, nil
	}
	id, secret, err := storageKeys(p, b.Credentials)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	s3, err := storage.NewS3(storage.S3Config{
		Endpoint: b.Endpoint, Region: b.Region, Bucket: b.Bucket, Prefix: b.Prefix,
		PathStyle: b.PathStyle, AccessKey: id, SecretKey: secret,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &storage.Backend{Name: name, Store: s3, Mode: mode, PublicURL: b.PublicURL}, nil
}

func storageKeys(p *paths.Paths, credentials string) (string, string, error) {
	base := "storage"
	if credentials != "" {
		base += "-" + credentials
	}
	read := func(file string) (string, error) {
		data, err := os.ReadFile(filepath.Join(p.Secrets(), file))
		if err != nil {
			return "", fmt.Errorf("read the key from %s in the secrets folder: %w", file, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	id, err := read(base + "-access-key-id")
	if err != nil {
		return "", "", err
	}
	secret, err := read(base + "-secret-access-key")
	if err != nil {
		return "", "", err
	}
	return id, secret, nil
}

func uploadPolicy(cfg config.File, live, total, maxFile string) (*quota.Policy, error) {
	var p quota.Policy
	var err error
	if p.InstanceLive, err = quota.ParseSize(live); err != nil {
		return nil, fmt.Errorf("parse -upload-limit: %w", err)
	}
	if p.InstanceTotal, err = quota.ParseSize(total); err != nil {
		return nil, fmt.Errorf("parse -storage-limit: %w", err)
	}
	if p.Default, err = quota.ParseRule(maxFile, cfg.Uploads.SiteQuota); err != nil {
		return nil, fmt.Errorf("[uploads] %w", err)
	}
	p.Sites = map[string]quota.Rule{}
	for slug, limits := range cfg.Uploads.Sites {
		if p.Sites[slug], err = quota.ParseRule(limits.MaxFile, limits.SiteQuota); err != nil {
			return nil, fmt.Errorf("[uploads.sites.%s] %w", slug, err)
		}
	}
	p.KeepDeleted = cfg.Uploads.KeepDeleted
	return &p, nil
}

func instanceStorage(p *paths.Paths) (*storage.Set, error) {
	cfg, err := config.Load(p.Config())
	if err != nil {
		return nil, err
	}
	return openStorage(p, cfg)
}

// Icons only decorate the output of these commands, so a storage setting that
// cannot be read falls back to the files folder instead of stopping them.
func sharedIcons(p *paths.Paths) roles.IconLoader {
	files, err := instanceStorage(p)
	if err != nil {
		return roles.FileIcons(storage.NewLocal(p.Files()))
	}
	return roles.FileIcons(files.Shared.Store)
}

// A page counts as public when a reader who is not signed in may view it, the
// same check the page itself makes before showing anything.
func anonymousCanView(conn *db.DB) mediaurl.Public {
	return func(ctx context.Context, current *db.Site, name string) bool {
		article, err := conn.ArticleByName(ctx, current.ID, name)
		if err != nil {
			return false
		}
		perm := repo.NewPerms(ctx, conn)
		subject, err := perm.Subject(nil, time.Now())
		if err != nil {
			return false
		}
		object, err := perm.Article(article, nil)
		if err != nil {
			return false
		}
		return perms.Resolve(subject, object).Has(perms.ViewArticles)
	}
}
