//go:build bundle

package pgbundle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func testConfig(t *testing.T) Config {
	root := t.TempDir()
	return Config{
		Root:     root,
		Postgres: filepath.Join(root, "postgres"),
		Data:     filepath.Join(root, "pgdata"),
		Secrets:  filepath.Join(root, "secrets"),
		Logs:     filepath.Join(root, "logs"),
	}
}

func mustStart(t *testing.T, cfg Config, owner Owner) *Server {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	s, err := Start(ctx, cfg, owner)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { mustStop(t, s) })
	return s
}

func mustStop(t *testing.T, s *Server) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func queryString(t *testing.T, dsn, sql string) string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("pgx.Connect(%q) error = %v", dsn, err)
	}
	defer conn.Close(ctx)
	var out string
	if err := conn.QueryRow(ctx, sql).Scan(&out); err != nil {
		t.Fatalf("QueryRow(%q) error = %v", sql, err)
	}
	return out
}

func TestStartServesThePwikitDatabase(t *testing.T) {
	cfg := testConfig(t)
	s := mustStart(t, cfg, OwnerServe)

	if got := queryString(t, s.DSN(), "SELECT current_database()"); got != Database {
		t.Errorf("current_database() = %q, want %q", got, Database)
	}
	if got := queryString(t, s.DSN(), "SELECT current_setting('server_version_num')"); got[:2] != Version[:2] {
		t.Errorf("server_version_num = %q, want major %s", got, Version[:2])
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, s.DSN())
	if err != nil {
		t.Fatalf("pgx.Connect() error = %v", err)
	}
	defer conn.Close(ctx)
	for _, ext := range []string{"citext", "pg_trgm"} {
		if _, err := conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS "+ext); err != nil {
			t.Errorf("CREATE EXTENSION %s error = %v", ext, err)
		}
	}
	var similar float32
	if err := conn.QueryRow(ctx, "SELECT similarity('wikidot', 'wikidoc')").Scan(&similar); err != nil || similar <= 0 {
		t.Errorf("similarity('wikidot', 'wikidoc') = %v, %v, want a positive score", similar, err)
	}
}

func TestStartRefusesASecondOwner(t *testing.T) {
	cfg := testConfig(t)
	mustStart(t, cfg, OwnerServe)

	_, err := Start(context.Background(), cfg, OwnerCommand)
	var held *HeldError
	if !errors.As(err, &held) {
		t.Fatalf("second Start() error = %v, want *HeldError", err)
	}
	if held.Owner != OwnerServe {
		t.Errorf("HeldError.Owner = %q, want %q", held.Owner, OwnerServe)
	}
	if held.PID != os.Getpid() {
		t.Errorf("HeldError.PID = %d, want %d", held.PID, os.Getpid())
	}
}

func TestAttachReachesTheRunningServer(t *testing.T) {
	cfg := testConfig(t)
	mustStart(t, cfg, OwnerServe)

	dsn, err := Attach(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if got := queryString(t, dsn, "SELECT current_database()"); got != Database {
		t.Errorf("current_database() = %q, want %q", got, Database)
	}
}

func TestStopClosesExitedAndFreesTheLock(t *testing.T) {
	cfg := testConfig(t)
	ctx := context.Background()
	s, err := Start(ctx, cfg, OwnerServe)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	mustStop(t, s)

	select {
	case <-s.Exited():
	default:
		t.Errorf("Exited() open after Stop, want closed")
	}
	if _, ok := readPostmaster(cfg.Data); ok {
		t.Errorf("readPostmaster() found %s after Stop, want none", postmasterFile)
	}

	again := mustStart(t, cfg, OwnerServe)
	if got := queryString(t, again.DSN(), "SELECT current_database()"); got != Database {
		t.Errorf("current_database() after restart = %q, want %q", got, Database)
	}
}

func TestStartKeepsDataAcrossRestarts(t *testing.T) {
	cfg := testConfig(t)
	ctx := context.Background()
	s, err := Start(ctx, cfg, OwnerServe)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	conn, err := pgx.Connect(ctx, s.DSN())
	if err != nil {
		mustStop(t, s)
		t.Fatalf("pgx.Connect() error = %v", err)
	}
	_, err = conn.Exec(ctx, "CREATE TABLE kept (v text); INSERT INTO kept VALUES ('here')")
	conn.Close(ctx)
	mustStop(t, s)
	if err != nil {
		t.Fatalf("Exec(CREATE TABLE kept) error = %v", err)
	}

	again := mustStart(t, cfg, OwnerServe)
	if got := queryString(t, again.DSN(), "SELECT v FROM kept"); got != "here" {
		t.Errorf("SELECT v FROM kept = %q, want %q", got, "here")
	}
}

func TestStartStopsAServerLeftBehind(t *testing.T) {
	cfg := testConfig(t)
	ctx := context.Background()
	first, err := Start(ctx, cfg, OwnerServe)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	leftPID := first.proc.pid()
	first.lock.Release()

	second := mustStart(t, cfg, OwnerServe)
	select {
	case <-first.Exited():
	case <-time.After(30 * time.Second):
		t.Fatalf("process %d still running after a new Start, want stopped", leftPID)
	}
	if got := second.proc.pid(); got == leftPID {
		t.Errorf("new server pid = %d, want a new process", got)
	}
	if got := queryString(t, second.DSN(), "SELECT current_database()"); got != Database {
		t.Errorf("current_database() = %q, want %q", got, Database)
	}
}

func TestStartRefusesDataFromAnotherMajor(t *testing.T) {
	cfg := testConfig(t)
	if err := os.MkdirAll(cfg.Data, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Data, versionFile), []byte("17\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(context.Background(), cfg, OwnerServe); err == nil {
		t.Fatalf("Start() on PG_VERSION 17 error = nil, want refusal")
	}
	lock, err := TryLock(filepath.Join(cfg.Root, lockFile), OwnerServe)
	if err != nil {
		t.Fatalf("TryLock() after refused Start error = %v, want nil", err)
	}
	lock.Release()
}

func TestStartRefusesFilesThatAreNotData(t *testing.T) {
	cfg := testConfig(t)
	if err := os.MkdirAll(cfg.Data, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Data, "notes.txt"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(context.Background(), cfg, OwnerServe); err == nil {
		t.Fatalf("Start() on a directory of other files error = nil, want refusal")
	}
	if _, err := os.Stat(filepath.Join(cfg.Data, "notes.txt")); err != nil {
		t.Errorf("Stat(notes.txt) after refused Start error = %v, want nil", err)
	}
}
