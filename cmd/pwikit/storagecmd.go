package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/backup"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

func storageCommand(args []string) error {
	if len(args) == 0 || args[0] != "migrate" {
		fmt.Fprint(os.Stderr, `Usage: pwikit storage migrate <-to-bucket|-to-local> [options]

  migrate  copy the uploaded files between the files folder and the buckets
           pwikit.toml names, so a site can change where it keeps them

Options:
  -to-bucket      copy from the files folder into each site's bucket
  -to-local       copy from each site's bucket into the files folder
  -site           only this site; shared files such as avatars go with every run
  -delete-source  remove each file from where it came from once it is copied
  -dry-run        only count what would be copied
`)
		return errors.New("unknown storage subcommand")
	}
	fs := flag.NewFlagSet("storage migrate", flag.ContinueOnError)
	toBucket := fs.Bool("to-bucket", false, "copy from the files folder into each site's bucket")
	toLocal := fs.Bool("to-local", false, "copy from each site's bucket into the files folder")
	only := fs.String("site", "", "only this site")
	deleteSource := fs.Bool("delete-source", false, "remove each file from where it came from once it is copied")
	dryRun := fs.Bool("dry-run", false, "only count what would be copied")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *toBucket == *toLocal {
		return errors.New("give one of -to-bucket or -to-local")
	}

	ctx := context.Background()
	p, err := paths.New(*dataDir)
	if err != nil {
		return err
	}
	set, err := instanceStorage(p)
	if err != nil {
		return err
	}
	dsn, release, err := resolveDatabase(ctx, *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	route, err := backup.NewRouter(ctx, conn, set)
	if err != nil {
		return err
	}

	local := storage.NewLocal(p.Files())
	m := migration{route: route, only: *only, deleteSource: *deleteSource, dryRun: *dryRun}
	if *toBucket {
		err = m.run(ctx, local, func(key string) storage.Store { return bucketOf(route, key) })
	} else {
		for _, b := range set.All() {
			if _, isLocal := b.Store.(*storage.Local); isLocal {
				continue
			}
			from := b
			if err = m.run(ctx, b.Store, func(key string) storage.Store {
				if route.Backend(key) != from {
					return nil
				}
				return local
			}); err != nil {
				break
			}
		}
	}
	fmt.Printf("%d copied, %d already there, %d skipped\n", m.copied, m.present, m.skipped)
	if *dryRun {
		fmt.Println("nothing was changed")
	}
	return err
}

func bucketOf(route *backup.Router, key string) storage.Store {
	b := route.Backend(key)
	if _, isLocal := b.Store.(*storage.Local); isLocal {
		return nil
	}
	return b.Store
}

type migration struct {
	route        *backup.Router
	only         string
	deleteSource bool
	dryRun       bool

	copied, present, skipped int
}

// Resized copies are left behind, since each store makes its own on demand.
func (m *migration) run(ctx context.Context, from storage.Store, target func(key string) storage.Store) error {
	return from.Walk(ctx, "", func(key string, info storage.Info) error {
		if len(key) > 8 && key[:8] == "resized/" {
			return nil
		}
		if site := m.route.Site(key); m.only != "" && site != "" && site != m.only {
			return nil
		}
		to := target(key)
		if to == nil {
			m.skipped++
			return nil
		}
		if have, err := to.Stat(ctx, key); err == nil && have.Size == info.Size {
			m.present++
			return m.drop(ctx, from, key)
		}
		if m.dryRun {
			m.copied++
			return nil
		}
		f, err := from.Open(ctx, key)
		if err != nil {
			return err
		}
		err = to.Put(ctx, key, f, info.Size)
		f.Close()
		if err != nil {
			return fmt.Errorf("copy %s: %w", key, err)
		}
		if have, err := to.Stat(ctx, key); err != nil || have.Size != info.Size {
			return fmt.Errorf("copy %s: the copy does not match the original", key)
		}
		m.copied++
		if m.copied%100 == 0 {
			fmt.Printf("%d copied\n", m.copied)
		}
		return m.drop(ctx, from, key)
	})
}

func (m *migration) drop(ctx context.Context, from storage.Store, key string) error {
	if !m.deleteSource || m.dryRun {
		return nil
	}
	return from.Remove(ctx, key)
}
