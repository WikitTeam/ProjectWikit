package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	d, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	t.Cleanup(d.Close)
	return d
}

// TestQueriesMatchSchema asks Postgres to parse and plan every registered
// statement. A column that Django renamed or dropped fails here instead of on
// a page load.
func TestQueriesMatchSchema(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() err = %v, want nil", err)
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() err = %v, want nil", err)
	}
	defer conn.Release()

	if len(queries) == 0 {
		t.Fatal("len(queries) = 0, want every statement registered")
	}
	for _, q := range queries {
		if _, err := conn.Conn().Prepare(ctx, q.name, q.sql); err != nil {
			t.Errorf("Prepare(%s) err = %v, want nil", q.name, err)
		}
	}
}
