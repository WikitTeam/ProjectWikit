package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/version"
)

type CreateOptions struct {
	DSN    string
	Files  string
	Output string
	// Site narrows the backup to one site, leaving the rest of the instance
	// behind. Empty takes everything.
	Site          string
	KeepPasswords bool
	Report        func(string)
}

type CreateResult struct {
	Path     string
	Manifest Manifest
	Bytes    int64
}

func Create(ctx context.Context, opts CreateOptions) (CreateResult, error) {
	if opts.Output == "" {
		return CreateResult{}, fmt.Errorf("no output path")
	}
	conn, err := pgx.Connect(ctx, opts.DSN)
	if err != nil {
		return CreateResult{}, fmt.Errorf("connect to back up: %w", err)
	}
	defer conn.Close(ctx)

	staged, m, err := stage(ctx, conn, opts)
	defer staged.remove()
	if err != nil {
		return CreateResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(opts.Output), 0o755); err != nil {
		return CreateResult{}, err
	}
	// Written beside the target so the rename cannot cross a filesystem, and so
	// an interrupted run leaves a name nothing mistakes for a backup.
	temp, err := os.CreateTemp(filepath.Dir(opts.Output), filepath.Base(opts.Output)+".partial-*")
	if err != nil {
		return CreateResult{}, err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	report(opts.Report, "writing "+filepath.Base(opts.Output))
	if err := writeArchive(temp, m, staged, opts.Files); err != nil {
		return CreateResult{}, err
	}
	size, err := temp.Seek(0, io.SeekCurrent)
	if err != nil {
		return CreateResult{}, err
	}
	if err := temp.Close(); err != nil {
		return CreateResult{}, err
	}
	if err := os.Rename(temp.Name(), opts.Output); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Path: opts.Output, Manifest: m, Bytes: size}, nil
}

// The manifest cannot be finished until every checksum is in, yet it has to be
// the first entry, so each table waits in a scratch file.
type staging struct {
	order []string
	paths map[string]string
}

func (s staging) remove() {
	for _, name := range s.paths {
		if name != "" {
			os.Remove(name)
		}
	}
}

func stage(ctx context.Context, conn *pgx.Conn, opts CreateOptions) (staging, Manifest, error) {
	staged := staging{paths: map[string]string{}}
	m := Manifest{
		Format:    Format,
		CreatedAt: time.Now().UTC(),
		Pwikit:    version.String(),
		Site:      opts.Site,
		Tables:    map[string]Table{},
		Files:     Files{Entries: map[string]string{}},
	}
	if opts.Site != "" {
		known, err := db.SiteSlugExists(ctx, conn, opts.Site)
		if err != nil {
			return staged, m, err
		}
		if !known {
			return staged, m, fmt.Errorf("no site has the slug %q", opts.Site)
		}
		m.KeptPasswords = opts.KeepPasswords
	}
	var err error
	if m.PGVersion, err = db.ServerVersion(ctx, conn); err != nil {
		return staged, m, err
	}
	if m.Migrations, err = db.AppliedMigrations(ctx, conn); err != nil {
		return staged, m, err
	}

	// Every table is read inside one snapshot, so the rows of one cannot be
	// newer than the rows of another.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return staged, m, fmt.Errorf("open the snapshot: %w", err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	tables, err := db.BackupTables(ctx, conn)
	if err != nil {
		return staged, m, err
	}
	for i, name := range tables {
		report(opts.Report, fmt.Sprintf("[%d/%d] %s", i+1, len(tables), name))
		query := ""
		if opts.Site != "" {
			scoped, err := db.SiteExportQuery(ctx, conn, name, opts.KeepPasswords)
			if err != nil {
				return staged, m, err
			}
			if scoped == "" {
				m.Tables[name] = Table{SHA256: emptySHA256}
				staged.order = append(staged.order, name)
				staged.paths[name] = ""
				continue
			}
			slug, err := db.QuoteLiteral(ctx, tx, opts.Site)
			if err != nil {
				return staged, m, err
			}
			query = strings.ReplaceAll(scoped, "$1", slug)
		}
		t, at, err := copyTable(ctx, tx, name, query)
		if err != nil {
			return staged, m, err
		}
		m.Tables[name] = t
		staged.order = append(staged.order, name)
		staged.paths[name] = at
	}
	if err := tx.Rollback(ctx); err != nil && !strings.Contains(err.Error(), "closed") {
		return staged, m, err
	}

	// The files are read after the rows, so one deleted while this ran is still
	// in the archive rather than missing from it.
	if opts.Files != "" {
		report(opts.Report, "reading files")
		if err := hashFiles(opts.Files, &m); err != nil {
			return staged, m, err
		}
	}
	return staged, m, nil
}

// A tar entry needs its size up front, so each table lands in a scratch file.
// The digest is taken on the way there, which keeps it to one pass.
func copyTable(ctx context.Context, tx pgx.Tx, name, query string) (Table, string, error) {
	scratch, err := os.CreateTemp("", "pwbak-*")
	if err != nil {
		return Table{}, "", err
	}
	defer scratch.Close()

	sum := sha256.New()
	rows, err := db.CopyOut(ctx, tx, io.MultiWriter(scratch, sum), name, query)
	if err != nil {
		os.Remove(scratch.Name())
		return Table{}, "", err
	}
	size, err := scratch.Seek(0, io.SeekCurrent)
	if err != nil {
		os.Remove(scratch.Name())
		return Table{}, "", err
	}
	return Table{
		Rows:   rows,
		Bytes:  size,
		SHA256: hex.EncodeToString(sum.Sum(nil)),
	}, scratch.Name(), nil
}

func hashFiles(root string, m *Manifest) error {
	m.Files.Included = true
	return filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		defer file.Close()

		sum := sha256.New()
		if _, err := io.Copy(sum, file); err != nil {
			return err
		}
		m.Files.Entries[filepath.ToSlash(rel)] = hex.EncodeToString(sum.Sum(nil))
		m.Files.Count++
		m.Files.Bytes += info.Size()
		return nil
	})
}

func writeArchive(w io.Writer, m Manifest, staged staging, filesRoot string) error {
	zip := gzip.NewWriter(w)
	tw := tar.NewWriter(zip)

	if err := writeManifest(tw, m); err != nil {
		return err
	}
	for _, name := range staged.order {
		if staged.paths[name] == "" {
			if err := writeEmptyEntry(tw, path.Join(dataDir, name+dataSuffix)); err != nil {
				return err
			}
			continue
		}
		if err := writeFileEntry(tw, path.Join(dataDir, name+dataSuffix), staged.paths[name]); err != nil {
			return err
		}
	}
	if m.Files.Included {
		rels := make([]string, 0, len(m.Files.Entries))
		for rel := range m.Files.Entries {
			rels = append(rels, rel)
		}
		sort.Strings(rels)
		for _, rel := range rels {
			from := filepath.Join(filesRoot, filepath.FromSlash(rel))
			if err := writeFileEntry(tw, path.Join(filesDir, rel), from); err != nil {
				return err
			}
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return zip.Close()
}

func writeFileEntry(tw *tar.Writer, as, from string) error {
	file, err := os.Open(from)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: as, Mode: 0o600, Size: info.Size()}); err != nil {
		return err
	}
	_, err = io.Copy(tw, file)
	return err
}

func writeEmptyEntry(tw *tar.Writer, as string) error {
	return tw.WriteHeader(&tar.Header{Name: as, Mode: 0o600, Size: 0})
}

func writeManifest(tw *tar.Writer, m Manifest) error {
	var body strings.Builder
	if err := m.write(&body); err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: ManifestName, Mode: 0o600, Size: int64(body.Len())}); err != nil {
		return err
	}
	_, err := io.WriteString(tw, body.String())
	return err
}

func DefaultName(at time.Time, site string) string {
	name := "pwikit-"
	if site != "" {
		name += site + "-"
	}
	return name + at.UTC().Format("20060102-150405") + Extension
}

func report(f func(string), line string) {
	if f != nil {
		f(line)
	}
}
