// Package migrate owns the database schema.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed sql/*.sql
var files embed.FS

const (
	dir = "sql"

	BaselineName = "0001_baseline.sql"

	// The schema the baseline was taken from, as the database that wrote it
	// records the name.
	BaselineSchema = "0087_align_models_and_schema"

	lockKey = int64(0x7077696B_69746D67)

	versionTable = "pwikit_migration"
)

type State struct {
	Applied   []string
	Pending   []string
	Adoptable bool
	Unknown   []string
}

type Result struct {
	Adopted bool
	Applied []string
}

var names = load()

func Names() []string { return slices.Clone(names) }

func load() []string {
	entries, err := files.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			out = append(out, e.Name())
		}
	}
	slices.Sort(out)
	return out
}

func Status(ctx context.Context, dsn string) (State, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return State{}, fmt.Errorf("connect to read the schema state: %w", err)
	}
	defer conn.Close(ctx)
	return status(ctx, conn)
}

func status(ctx context.Context, conn *pgx.Conn) (State, error) {
	present, err := tableExists(ctx, conn, versionTable)
	if err != nil {
		return State{}, err
	}
	var applied []string
	if present {
		applied, err = appliedNames(ctx, conn)
		if err != nil {
			return State{}, err
		}
	}

	state := State{Applied: applied}
	for _, name := range names {
		if !slices.Contains(applied, name) {
			state.Pending = append(state.Pending, name)
		}
	}
	for _, name := range applied {
		if !slices.Contains(names, name) {
			state.Unknown = append(state.Unknown, name)
		}
	}
	if len(applied) == 0 {
		state.Adoptable, err = builtElsewhere(ctx, conn)
		if err != nil {
			return State{}, err
		}
	}
	return state, nil
}

func Run(ctx context.Context, dsn string) (Result, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return Result{}, fmt.Errorf("connect to migrate: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return Result{}, fmt.Errorf("take the migration lock: %w", err)
	}
	defer conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockKey)

	return run(ctx, conn)
}

func run(ctx context.Context, conn *pgx.Conn) (Result, error) {
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+versionTable+` (
	name text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return Result{}, fmt.Errorf("create %s: %w", versionTable, err)
	}

	applied, err := appliedNames(ctx, conn)
	if err != nil {
		return Result{}, err
	}
	for _, name := range applied {
		if !slices.Contains(names, name) {
			return Result{}, fmt.Errorf("the database has applied %q, which this build does not carry", name)
		}
	}

	var out Result
	if len(applied) == 0 {
		adopt, err := builtElsewhere(ctx, conn)
		if err != nil {
			return Result{}, err
		}
		if adopt {
			if _, err := conn.Exec(ctx, `INSERT INTO `+versionTable+` (name) VALUES ($1)`, BaselineName); err != nil {
				return Result{}, fmt.Errorf("record %s: %w", BaselineName, err)
			}
			applied = append(applied, BaselineName)
			out.Adopted = true
		}
	}

	for _, name := range names {
		if slices.Contains(applied, name) {
			continue
		}
		if err := apply(ctx, conn, name); err != nil {
			return out, err
		}
		out.Applied = append(out.Applied, name)
	}
	return out, nil
}

// ApplyInto replays migrations inside a transaction the caller owns, which is
// what lets a restore rebuild the schema and load the data all or nothing.
func ApplyInto(ctx context.Context, tx pgx.Tx, wanted []string) error {
	for _, name := range wanted {
		if !slices.Contains(names, name) {
			return fmt.Errorf("this build does not carry %q", name)
		}
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+versionTable+` (
	name text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create %s: %w", versionTable, err)
	}
	for _, name := range wanted {
		body, err := files.ReadFile(path.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO `+versionTable+` (name) VALUES ($1)`, name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
		// A deferrable key made here leaves its first check queued, and a queued
		// check blocks the next migration from altering that table.
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
			return fmt.Errorf("settle the constraints after %s: %w", name, err)
		}
	}
	return nil
}

func apply(ctx context.Context, conn *pgx.Conn, name string) error {
	body, err := files.ReadFile(path.Join(dir, name))
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", name, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return fmt.Errorf("apply %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO `+versionTable+` (name) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}
	return tx.Commit(ctx)
}

func appliedNames(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, `SELECT name FROM `+versionTable+` ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", versionTable, err)
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

func builtElsewhere(ctx context.Context, conn *pgx.Conn) (bool, error) {
	present, err := tableExists(ctx, conn, "django_migrations")
	if err != nil || !present {
		return false, err
	}
	var any bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM public.django_migrations)`).Scan(&any); err != nil {
		return false, fmt.Errorf("read the previous version table: %w", err)
	}
	if !any {
		return false, nil
	}
	return true, checkBaselineSchema(ctx, conn)
}

// Adopting a database that stopped before the baseline was taken would apply
// every later migration to a schema they do not fit.
func checkBaselineSchema(ctx context.Context, conn *pgx.Conn) error {
	var reached string
	err := conn.QueryRow(ctx, `
SELECT coalesce(max(name), '') FROM public.django_migrations WHERE app = 'web'`).Scan(&reached)
	if err != nil {
		return fmt.Errorf("read the previous version table: %w", err)
	}
	if reached == "" || reached >= BaselineSchema {
		return nil
	}
	return fmt.Errorf("this database stopped at %q and pwikit needs the schema of %q, so nothing was changed", reached, BaselineSchema)
}

func tableExists(ctx context.Context, conn *pgx.Conn, name string) (bool, error) {
	var present bool
	if err := conn.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, name).Scan(&present); err != nil {
		return false, fmt.Errorf("look for table %q: %w", name, err)
	}
	return present, nil
}
