package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/migrate"
)

type RestoreOptions struct {
	DSN    string
	Files  string
	Force  bool
	Report func(string)
}

type RestoreResult struct {
	Manifest    Manifest
	Rows        int64
	FilesPut    int
	MigratedUp  []string
	FilesMoved  bool
	ReplacedDir string
}

var ErrNotEmpty = errors.New("the database already holds data")

// The migrations fill these before anyone uses the database, so rows in them
// alone leave nothing to replace or keep.
var seeded = map[string]bool{
	"auth_permission":      true,
	"django_content_type":  true,
	"web_role":             true,
	"web_role_permissions": true,
	"web_rolecategory":     true,
	"web_theme":            true,
}

func Ready(ctx context.Context, dsn string, force bool) (holdsData bool, err error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return false, fmt.Errorf("connect to restore: %w", err)
	}
	defer conn.Close(ctx)
	return readyToRestore(ctx, conn, force)
}

func Restore(ctx context.Context, name string, opts RestoreOptions) (RestoreResult, error) {
	var out RestoreResult

	report(opts.Report, "checking the backup")
	check, err := Verify(name)
	if err != nil {
		return out, err
	}
	if !check.OK() {
		return out, fmt.Errorf("the backup did not pass its check, so nothing was changed:\n  %s",
			strings.Join(check.Problems, "\n  "))
	}
	m := check.Manifest
	out.Manifest = m

	conn, err := pgx.Connect(ctx, opts.DSN)
	if err != nil {
		return out, fmt.Errorf("connect to restore: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := readyToRestore(ctx, conn, opts.Force); err != nil {
		return out, err
	}

	report(opts.Report, "rebuilding the schema")
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if err := db.ResetSchema(ctx, tx); err != nil {
		return out, err
	}
	if err := migrate.ApplyInto(ctx, tx, m.Migrations); err != nil {
		return out, err
	}
	// The baseline seeds permissions, content types and the built-in roles, and
	// the backup carries its own copy of all of them.
	if err := emptyEverything(ctx, tx); err != nil {
		return out, err
	}
	keys, err := db.LiftForeignKeys(ctx, tx)
	if err != nil {
		return out, err
	}

	rows, err := loadTables(ctx, tx, name, m, opts.Report)
	if err != nil {
		return out, err
	}
	out.Rows = rows

	report(opts.Report, "checking the references")
	if err := db.RestoreForeignKeys(ctx, tx, keys); err != nil {
		return out, err
	}
	if err := db.ResetIdentities(ctx, tx, named(m)); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, fmt.Errorf("commit the restore: %w", err)
	}

	// The rest of the migrations run only once the data is in, so a backup
	// taken by an older build comes forward instead of being refused.
	out.MigratedUp, err = catchUp(ctx, opts.DSN, m)
	if err != nil {
		return out, err
	}

	if m.Files.Included && opts.Files != "" {
		report(opts.Report, "putting the files back")
		if err := restoreFiles(name, opts.Files, m, &out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func emptyEverything(ctx context.Context, tx pgx.Tx) error {
	tables, err := db.BackupTables(ctx, tx.Conn())
	if err != nil {
		return err
	}
	return db.TruncateAll(ctx, tx, tables)
}

func named(m Manifest) map[string]bool {
	out := make(map[string]bool, len(m.Tables))
	for name := range m.Tables {
		out[name] = true
	}
	return out
}

func readyToRestore(ctx context.Context, conn *pgx.Conn, force bool) (bool, error) {
	others, err := db.OtherConnections(ctx, conn)
	if err != nil {
		return false, err
	}
	if others > 0 {
		return false, fmt.Errorf("%d other connections are using this database; stop pwikit before restoring", others)
	}
	tables, err := db.BackupTables(ctx, conn)
	if err != nil {
		return false, err
	}
	var used []string
	for _, name := range tables {
		if !seeded[name] {
			used = append(used, name)
		}
	}
	full, err := db.NonEmptyTables(ctx, conn, used)
	if err != nil {
		return false, err
	}
	if len(full) > 0 && !force {
		return true, fmt.Errorf("%w (%s and %d more tables); run again with -force to replace it",
			ErrNotEmpty, full[0], len(full)-1)
	}
	return len(full) > 0, nil
}

func loadTables(ctx context.Context, tx pgx.Tx, name string, m Manifest, out func(string)) (int64, error) {
	// The references are off while this runs, so the tables can arrive in any
	// order and this stays one pass.
	file, err := os.Open(name)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	zip, err := gzip.NewReader(file)
	if err != nil {
		return 0, err
	}
	defer zip.Close()

	var total int64
	done := 0
	tr := tar.NewReader(zip)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
		if !strings.HasPrefix(head.Name, dataDir+"/") {
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return total, err
			}
			continue
		}
		table := strings.TrimSuffix(path.Base(head.Name), dataSuffix)
		done++
		report(out, fmt.Sprintf("[%d/%d] %s", done, len(m.Tables), table))
		rows, err := db.CopyIn(ctx, tx, tr, table)
		if err != nil {
			return total, err
		}
		total += rows
	}
	return total, nil
}

func catchUp(ctx context.Context, dsn string, m Manifest) ([]string, error) {
	if len(m.Migrations) == len(migrate.Names()) {
		return nil, nil
	}
	result, err := migrate.Run(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("bring the schema up to this build: %w", err)
	}
	return result.Applied, nil
}

// The files land beside the real directory and only swap in once every one of
// them is written, so an interrupted restore leaves the old tree alone.
func restoreFiles(archive, root string, m Manifest, out *RestoreResult) error {
	staging := root + ".restoring"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()

	zip, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer zip.Close()

	tr := tar.NewReader(zip)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !strings.HasPrefix(head.Name, filesDir+"/") {
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return err
			}
			continue
		}
		rel := strings.TrimPrefix(head.Name, filesDir+"/")
		to, err := safeJoin(staging, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		written, err := os.Create(to)
		if err != nil {
			return err
		}
		if _, err := io.Copy(written, tr); err != nil {
			written.Close()
			return err
		}
		if err := written.Close(); err != nil {
			return err
		}
		out.FilesPut++
	}

	retired := root + ".replaced"
	if err := os.RemoveAll(retired); err != nil {
		return err
	}
	if _, err := os.Stat(root); err == nil {
		if err := os.Rename(root, retired); err != nil {
			return err
		}
		out.ReplacedDir = retired
	}
	if err := os.Rename(staging, root); err != nil {
		return err
	}
	out.FilesMoved = true
	return nil
}

func safeJoin(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("the archive names %q, which points outside the files directory", rel)
	}
	return filepath.Join(root, clean), nil
}
