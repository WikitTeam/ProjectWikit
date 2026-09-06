package migrate

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const envDSN = "PWIKIT_TEST_DSN"

func TestNamesAreOrderedAndStartAtTheBaseline(t *testing.T) {
	got := Names()
	if len(got) == 0 {
		t.Fatal("Names() = [], want at least the baseline")
	}
	if got[0] != BaselineName {
		t.Errorf("Names()[0] = %q, want %q", got[0], BaselineName)
	}
	if !slices.IsSorted(got) {
		t.Errorf("Names() = %v, want it sorted", got)
	}
}

func TestBaselineReproducesTheSchemaItWasTakenFrom(t *testing.T) {
	reference := requireDSN(t)
	fresh := scratch(t)

	result, err := Run(context.Background(), fresh)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if result.Adopted {
		t.Error("Run().Adopted = true, want false on an empty database")
	}
	if !slices.Equal(result.Applied, Names()) {
		t.Errorf("Run().Applied = %v, want %v", result.Applied, Names())
	}

	for _, part := range []struct {
		name string
		sql  string
	}{
		{"columns", qColumns},
		{"indexes", qIndexes},
		{"constraints", qConstraints},
	} {
		want := describe(t, reference, part.sql)
		got := describe(t, fresh, part.sql)
		diffLines(t, part.name, got, want)
	}
}

func TestRunAppliesNothingTwice(t *testing.T) {
	fresh := scratch(t)
	ctx := context.Background()

	if _, err := Run(ctx, fresh); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	again, err := Run(ctx, fresh)
	if err != nil {
		t.Fatalf("Run() second time err = %v, want nil", err)
	}
	if len(again.Applied) != 0 {
		t.Errorf("Run() second time Applied = %v, want []", again.Applied)
	}
	if again.Adopted {
		t.Error("Run() second time Adopted = true, want false")
	}
}

func TestRunAdoptsASchemaBuiltBeforeGoOwnedIt(t *testing.T) {
	fresh := scratch(t)
	ctx := context.Background()

	conn := connect(t, fresh)
	body, err := files.ReadFile(dir + "/" + BaselineName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, string(body)); err != nil {
		t.Fatalf("apply the baseline by hand err = %v, want nil", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO django_migrations (app, name, applied) VALUES ('web', '0001_initial', now())`); err != nil {
		t.Fatalf("record a previous migration err = %v, want nil", err)
	}

	state, err := Status(ctx, fresh)
	if err != nil {
		t.Fatalf("Status() err = %v, want nil", err)
	}
	if !state.Adoptable {
		t.Error("Status().Adoptable = false, want true")
	}
	if len(state.Applied) != 0 {
		t.Errorf("Status().Applied = %v, want []", state.Applied)
	}

	result, err := Run(ctx, fresh)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if !result.Adopted {
		t.Error("Run().Adopted = false, want true")
	}
	if len(result.Applied) != 0 {
		t.Errorf("Run().Applied = %v, want [] because the tables are already there", result.Applied)
	}
}

func TestRunRefusesASchemaNewerThanTheBinary(t *testing.T) {
	fresh := scratch(t)
	ctx := context.Background()

	if _, err := Run(ctx, fresh); err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	conn := connect(t, fresh)
	if _, err := conn.Exec(ctx,
		`INSERT INTO `+versionTable+` (name) VALUES ('9999_from_the_future.sql')`); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(ctx, fresh); err == nil {
		t.Error("Run() over a newer schema = nil, want an error")
	} else if !strings.Contains(err.Error(), "9999_from_the_future.sql") {
		t.Errorf("Run() err = %q, want it to name the unknown migration", err)
	}

	state, err := Status(ctx, fresh)
	if err != nil {
		t.Fatalf("Status() err = %v, want nil", err)
	}
	if !slices.Equal(state.Unknown, []string{"9999_from_the_future.sql"}) {
		t.Errorf("Status().Unknown = %v, want [9999_from_the_future.sql]", state.Unknown)
	}
}

func TestStatusWritesNothing(t *testing.T) {
	fresh := scratch(t)
	ctx := context.Background()

	state, err := Status(ctx, fresh)
	if err != nil {
		t.Fatalf("Status() err = %v, want nil", err)
	}
	if !slices.Equal(state.Pending, Names()) {
		t.Errorf("Status().Pending = %v, want %v", state.Pending, Names())
	}

	conn := connect(t, fresh)
	var tables int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Errorf("tables after Status() = %d, want 0", tables)
	}
}

const qColumns = `
SELECT table_name || ' ' || column_name || ' ' || data_type ||
       ' null=' || is_nullable ||
       ' len=' || coalesce(character_maximum_length::text, '-') ||
       ' default=' || coalesce(column_default, '-')
FROM information_schema.columns
WHERE table_schema = 'public'
ORDER BY 1`

const qIndexes = `
SELECT indexdef FROM pg_indexes WHERE schemaname = 'public' ORDER BY 1`

const qConstraints = `
SELECT rel.relname || ' ' || con.conname || ' ' || pg_get_constraintdef(con.oid)
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
WHERE ns.nspname = 'public'
ORDER BY 1`

func describe(t *testing.T, dsn, query string) []string {
	t.Helper()
	conn := connect(t, dsn)
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query() err = %v, want nil", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(line, versionTable) {
			continue
		}
		out = append(out, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func diffLines(t *testing.T, part string, got, want []string) {
	t.Helper()
	if len(want) == 0 {
		t.Fatalf("%s in the reference database = 0 rows, want the schema", part)
	}
	for _, line := range want {
		if !slices.Contains(got, line) {
			t.Errorf("%s missing after the baseline, want %q", part, line)
		}
	}
	for _, line := range got {
		if !slices.Contains(want, line) {
			t.Errorf("%s added by the baseline, want it absent %q", part, line)
		}
	}
}

func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(envDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", envDSN)
	}
	return dsn
}

func connect(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Connect() err = %v, want nil", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

func scratch(t *testing.T) string {
	t.Helper()
	dsn := requireDSN(t)
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("ParseConfig() err = %v, want nil", err)
	}
	name := fmt.Sprintf("pwikit_migrate_%d", rand.Uint32())

	admin := swapDatabase(t, dsn, "postgres")
	ctx := context.Background()
	control, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Skipf("cannot reach the maintenance database to make a scratch one: %v", err)
	}
	defer control.Close(ctx)

	if _, err := control.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("CREATE DATABASE err = %v, want nil", err)
	}
	t.Cleanup(func() {
		clean, err := pgx.Connect(context.Background(), admin)
		if err != nil {
			return
		}
		defer clean.Close(context.Background())
		clean.Exec(context.Background(),
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, name)
		clean.Exec(context.Background(), `DROP DATABASE IF EXISTS `+pgx.Identifier{name}.Sanitize())
	})

	cfg.Database = name
	return swapDatabase(t, dsn, name)
}

func swapDatabase(t *testing.T, dsn, name string) string {
	t.Helper()
	cut := strings.LastIndex(dsn, "/")
	if cut < 0 {
		t.Fatalf("no database in %q, want a URL style connection string", dsn)
	}
	rest := ""
	if q := strings.Index(dsn[cut:], "?"); q >= 0 {
		rest = dsn[cut+q:]
	}
	return dsn[:cut+1] + name + rest
}
