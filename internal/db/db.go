// Package db is the only place in pwikit that issues SQL.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("db: not found")

// EnvDSN points database-backed tests at a live Postgres. It lives here rather
// than in a _test.go file because other packages' tests gate on the same name.
const EnvDSN = "PWIKIT_TEST_DSN"

type DB struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

type query struct {
	name string
	sql  string
}

// Every statement goes through register so the schema-drift test can reach all
// of them; a query held in a plain const would silently escape that check.
var queries []query

func register(name, sql string) string {
	queries = append(queries, query{name: name, sql: sql})
	return sql
}

// prefixed qualifies a column list with a table alias so joins can reuse the
// one list the scan order is written against.
func prefixed(alias, columns string) string {
	parts := strings.Split(columns, ", ")
	for i, c := range parts {
		parts[i] = alias + "." + c
	}
	return strings.Join(parts, ", ")
}
