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

	"github.com/WikitTeam/ProjectWikit/internal/version"
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

	declarationPrefix = "-- compat: "
	compatible        = "compatible"
	breakingValue     = "breaking"
)

type State struct {
	Applied         []string
	Pending         []string
	Adoptable       bool
	Unknown         []string
	UnknownBreaking []string
	AppliedBy       string
}

func (s State) PendingBreaking() bool {
	for _, name := range s.Pending {
		if breaking[name] {
			return true
		}
	}
	return false
}

type Result struct {
	Adopted bool
	Applied []string
	Newer   []string
}

type NewerSchemaError struct {
	Migrations []string
	AppliedBy  string
}

func (e *NewerSchemaError) Error() string {
	by := "a newer pwikit"
	if e.AppliedBy != "" {
		by = "pwikit " + e.AppliedBy
	}
	return fmt.Sprintf("the database was upgraded by %s, which applied %s; this pwikit (%s) cannot run on that schema. "+
		"Run %s or a newer release, or restore a backup taken before the upgrade",
		by, strings.Join(e.Migrations, ", "), version.String(), by)
}

var names, breaking, declarationErr = load()

func Names() []string { return slices.Clone(names) }

func Breaking(name string) bool { return breaking[name] }

func Declarations() error { return declarationErr }

func load() ([]string, map[string]bool, error) {
	entries, err := files.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	out := make([]string, 0, len(entries))
	marks := map[string]bool{}
	var problems []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		out = append(out, e.Name())
		body, err := files.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			panic(err)
		}
		first, _, _ := strings.Cut(string(body), "\n")
		switch strings.TrimSpace(first) {
		case declarationPrefix + compatible:
		case declarationPrefix + breakingValue:
			marks[e.Name()] = true
		default:
			problems = append(problems, e.Name())
		}
	}
	slices.Sort(out)
	if len(problems) > 0 {
		return out, marks, fmt.Errorf("%s must open with %q or %q",
			strings.Join(problems, ", "), declarationPrefix+compatible, declarationPrefix+breakingValue)
	}
	return out, marks, nil
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
	var applied []record
	if present {
		applied, err = appliedRecords(ctx, conn)
		if err != nil {
			return State{}, err
		}
	}

	state := State{}
	for _, r := range applied {
		state.Applied = append(state.Applied, r.name)
	}
	for _, name := range names {
		if !slices.Contains(state.Applied, name) {
			state.Pending = append(state.Pending, name)
		}
	}
	for _, r := range applied {
		if slices.Contains(names, r.name) {
			continue
		}
		state.Unknown = append(state.Unknown, r.name)
		if r.breaking {
			state.UnknownBreaking = append(state.UnknownBreaking, r.name)
		}
		if r.appliedBy != "" {
			state.AppliedBy = r.appliedBy
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
	if declarationErr != nil {
		return Result{}, declarationErr
	}
	if err := ensureLedger(ctx, conn); err != nil {
		return Result{}, err
	}

	state, err := status(ctx, conn)
	if err != nil {
		return Result{}, err
	}
	if len(state.UnknownBreaking) > 0 {
		return Result{}, &NewerSchemaError{Migrations: state.UnknownBreaking, AppliedBy: state.AppliedBy}
	}
	var out Result
	if len(state.Unknown) > 0 {
		if len(state.Pending) > 0 {
			return Result{}, fmt.Errorf("the database holds %s from a newer pwikit while this build still has %s to apply; run the newer pwikit",
				strings.Join(state.Unknown, ", "), strings.Join(state.Pending, ", "))
		}
		out.Newer = state.Unknown
	}

	// Rows written before the ledger kept declarations carry the default, and
	// an older build reading them later needs the truth.
	for _, name := range state.Applied {
		if breaking[name] {
			if _, err := conn.Exec(ctx, `UPDATE `+versionTable+` SET breaking = true WHERE name = $1 AND NOT breaking`, name); err != nil {
				return Result{}, fmt.Errorf("record the declaration of %s: %w", name, err)
			}
		}
	}

	applied := state.Applied
	if len(applied) == 0 {
		adopt, err := builtElsewhere(ctx, conn)
		if err != nil {
			return Result{}, err
		}
		if adopt {
			if _, err := conn.Exec(ctx, insertLedger, BaselineName, breaking[BaselineName], version.String()); err != nil {
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

func ensureLedger(ctx context.Context, conn *pgx.Conn) error {
	if _, err := conn.Exec(ctx, ledgerDDL); err != nil {
		return fmt.Errorf("create %s: %w", versionTable, err)
	}
	declared, err := ledgerDeclares(ctx, conn)
	if err != nil || declared {
		return err
	}
	if _, err := conn.Exec(ctx, `ALTER TABLE `+versionTable+`
	ADD COLUMN IF NOT EXISTS breaking boolean NOT NULL DEFAULT false,
	ADD COLUMN IF NOT EXISTS applied_by text NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add the declaration columns to %s: %w", versionTable, err)
	}
	return nil
}

const ledgerDDL = `CREATE TABLE IF NOT EXISTS ` + versionTable + ` (
	name text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now(),
	breaking boolean NOT NULL DEFAULT false,
	applied_by text NOT NULL DEFAULT '')`

// ApplyInto replays migrations inside a transaction the caller owns, which is
// what lets a restore rebuild the schema and load the data all or nothing.
func ApplyInto(ctx context.Context, tx pgx.Tx, wanted []string) error {
	if declarationErr != nil {
		return declarationErr
	}
	for _, name := range wanted {
		if !slices.Contains(names, name) {
			return fmt.Errorf("this build does not carry %q", name)
		}
	}
	if _, err := tx.Exec(ctx, ledgerDDL); err != nil {
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
		if _, err := tx.Exec(ctx, insertLedger, name, breaking[name], version.String()); err != nil {
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

const insertLedger = `INSERT INTO ` + versionTable + ` (name, breaking, applied_by) VALUES ($1, $2, $3)`

func ledgerDeclares(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var columns int
	if err := conn.QueryRow(ctx, `
SELECT count(*) FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = $1 AND column_name IN ('breaking', 'applied_by')`, versionTable).Scan(&columns); err != nil {
		return false, fmt.Errorf("read the columns of %s: %w", versionTable, err)
	}
	return columns == 2, nil
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
	if _, err := tx.Exec(ctx, insertLedger, name, breaking[name], version.String()); err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}
	return tx.Commit(ctx)
}

type record struct {
	name      string
	breaking  bool
	appliedBy string
}

func appliedRecords(ctx context.Context, conn *pgx.Conn) ([]record, error) {
	declared, err := ledgerDeclares(ctx, conn)
	if err != nil {
		return nil, err
	}
	query := `SELECT name, false, '' FROM ` + versionTable + ` ORDER BY name`
	if declared {
		query = `SELECT name, breaking, applied_by FROM ` + versionTable + ` ORDER BY name`
	}
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", versionTable, err)
	}
	defer rows.Close()

	var out []record
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.name, &r.breaking, &r.appliedBy); err != nil {
			return nil, err
		}
		out = append(out, r)
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
