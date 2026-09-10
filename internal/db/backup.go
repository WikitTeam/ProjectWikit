package db

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
)

// The migration ledger is rebuilt by replaying the migrations a restore names,
// so carrying its rows would fight with that.
const LedgerTable = "pwikit_migration"

var qBackupTables = register("BackupTables", `
SELECT c.relname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r'
ORDER BY c.relname`)

// These take a connection rather than the pool, because a backup streams COPY
// and rebuilds the schema inside one transaction.
func BackupTables(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, qBackupTables)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if name != LedgerTable {
			out = append(out, name)
		}
	}
	return out, rows.Err()
}

var qServerVersion = register("ServerVersion", `SELECT current_setting('server_version_num')::int`)

func ServerVersion(ctx context.Context, conn *pgx.Conn) (int, error) {
	var n int
	if err := conn.QueryRow(ctx, qServerVersion).Scan(&n); err != nil {
		return 0, fmt.Errorf("read the postgres version: %w", err)
	}
	return n, nil
}

var qAppliedMigrations = register("AppliedMigrations",
	`SELECT name FROM `+LedgerTable+` ORDER BY name`)

func AppliedMigrations(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, qAppliedMigrations)
	if err != nil {
		return nil, fmt.Errorf("read the applied migrations: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

var qOtherConnections = register("OtherConnections", `
SELECT count(*) FROM pg_stat_activity
WHERE datname = current_database() AND pid <> pg_backend_pid()`)

func OtherConnections(ctx context.Context, conn *pgx.Conn) (int, error) {
	var n int
	if err := conn.QueryRow(ctx, qOtherConnections).Scan(&n); err != nil {
		return 0, fmt.Errorf("look for other connections: %w", err)
	}
	return n, nil
}

func NonEmptyTables(ctx context.Context, conn *pgx.Conn, tables []string) ([]string, error) {
	var out []string
	for _, name := range tables {
		var any bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+QuoteName(name)+`)`).Scan(&any); err != nil {
			return nil, fmt.Errorf("look inside %s: %w", name, err)
		}
		if any {
			out = append(out, name)
		}
	}
	return out, nil
}

var qSiteSlugExists = register("SiteSlugExists", `SELECT EXISTS (SELECT 1 FROM web_site WHERE slug = $1)`)

func SiteSlugExists(ctx context.Context, conn *pgx.Conn, slug string) (bool, error) {
	var found bool
	if err := conn.QueryRow(ctx, qSiteSlugExists, slug).Scan(&found); err != nil {
		return false, fmt.Errorf("look for site %q: %w", slug, err)
	}
	return found, nil
}

func CopyOut(ctx context.Context, tx pgx.Tx, w io.Writer, table, query string) (int64, error) {
	source := `COPY ` + QuoteName(table) + ` TO STDOUT`
	if query != "" {
		source = `COPY (` + query + `) TO STDOUT`
	}
	tag, err := tx.Conn().PgConn().CopyTo(ctx, w, source)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", table, err)
	}
	return tag.RowsAffected(), nil
}

func CopyIn(ctx context.Context, tx pgx.Tx, r io.Reader, table string) (int64, error) {
	tag, err := tx.Conn().PgConn().CopyFrom(ctx, r, `COPY `+QuoteName(table)+` FROM STDIN`)
	if err != nil {
		return 0, fmt.Errorf("load %s: %w", table, err)
	}
	return tag.RowsAffected(), nil
}

// CopyTo takes no arguments, so a value the query needs is put into the text
// after Postgres has quoted it.
func QuoteLiteral(ctx context.Context, tx pgx.Tx, value string) (string, error) {
	var quoted string
	if err := tx.QueryRow(ctx, `SELECT quote_literal($1::text)`, value).Scan(&quoted); err != nil {
		return "", err
	}
	return quoted, nil
}

func ResetSchema(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DROP SCHEMA public CASCADE`); err != nil {
		return fmt.Errorf("clear the database: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE SCHEMA public`); err != nil {
		return fmt.Errorf("clear the database: %w", err)
	}
	return nil
}

func TruncateAll(ctx context.Context, tx pgx.Tx, tables []string) error {
	if len(tables) == 0 {
		return nil
	}
	quoted := make([]string, len(tables))
	for i, name := range tables {
		quoted[i] = QuoteName(name)
	}
	if _, err := tx.Exec(ctx, `TRUNCATE `+strings.Join(quoted, ", ")); err != nil {
		return fmt.Errorf("clear the seeded rows: %w", err)
	}
	return nil
}

type ForeignKey struct {
	Table string
	Name  string
	Def   string
}

// LiftForeignKeys takes the references off so rows can arrive in any order. Two
// of them point at each other, so no load order satisfies every one.
func LiftForeignKeys(ctx context.Context, tx pgx.Tx) ([]ForeignKey, error) {
	rows, err := tx.Query(ctx, `
SELECT c.relname, k.conname, pg_get_constraintdef(k.oid)
FROM pg_constraint k
JOIN pg_class c ON c.oid = k.conrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE k.contype = 'f' AND n.nspname = 'public'
ORDER BY c.relname, k.conname`)
	if err != nil {
		return nil, fmt.Errorf("list the references: %w", err)
	}
	var keys []ForeignKey
	for rows.Next() {
		var k ForeignKey
		if err := rows.Scan(&k.Table, &k.Name, &k.Def); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, k := range keys {
		if _, err := tx.Exec(ctx, `ALTER TABLE `+QuoteName(k.Table)+` DROP CONSTRAINT `+QuoteName(k.Name)); err != nil {
			return nil, fmt.Errorf("set aside %s: %w", k.Name, err)
		}
	}
	return keys, nil
}

// Putting them back checks every row in bulk rather than a trigger at a time.
func RestoreForeignKeys(ctx context.Context, tx pgx.Tx, keys []ForeignKey) error {
	for _, k := range keys {
		sql := `ALTER TABLE ` + QuoteName(k.Table) + ` ADD CONSTRAINT ` + QuoteName(k.Name) + ` ` + k.Def
		if _, err := tx.Exec(ctx, sql); err != nil {
			return fmt.Errorf("the restored rows break %s on %s: %w", k.Name, k.Table, err)
		}
	}
	return nil
}

// The rows carry their own ids, so every identity column has to be told where
// to carry on from or the next insert collides with row one.
func ResetIdentities(ctx context.Context, tx pgx.Tx, tables map[string]bool) error {
	rows, err := tx.Query(ctx, `
SELECT c.relname, a.attname
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r' AND a.attidentity <> ''`)
	if err != nil {
		return fmt.Errorf("list the identity columns: %w", err)
	}
	type column struct{ table, name string }
	var found []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.table, &c.name); err != nil {
			rows.Close()
			return err
		}
		found = append(found, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, c := range found {
		if !tables[c.table] {
			continue
		}
		sql := fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', '%s'), coalesce(max(%s), 0) + 1, false) FROM %s`,
			c.table, c.name, QuoteName(c.name), QuoteName(c.table))
		if _, err := tx.Exec(ctx, sql); err != nil {
			return fmt.Errorf("restart the numbering of %s: %w", c.table, err)
		}
	}
	return nil
}

func QuoteName(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
