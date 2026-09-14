package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/config"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/pgbundle"
)

const stopBundledWithin = 2 * time.Minute

func bundledConfig(p *paths.Paths, log *slog.Logger) pgbundle.Config {
	return pgbundle.Config{
		Root:     p.Root(),
		Postgres: p.Postgres(),
		Data:     p.PGData(),
		Secrets:  p.Secrets(),
		Logs:     p.Logs(),
		Log:      log,
	}
}

func prepareDataDir(p *paths.Paths) error {
	if err := p.EnsureBase(); err != nil {
		return err
	}
	_, err := config.WriteTemplate(p.Config())
	return err
}

const envDatabasePasswordFile = "PWIKIT_DATABASE_PASSWORD_FILE"

func withPasswordFile(dsn string) (string, error) {
	name := os.Getenv(envDatabasePasswordFile)
	if name == "" || dsn == "" {
		return dsn, nil
	}
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn, nil
	}
	if _, has := u.User.Password(); has {
		return dsn, nil
	}
	secret, err := os.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read the database password from %s: %w", name, err)
	}
	u.User = url.UserPassword(u.User.Username(), strings.TrimSpace(string(secret)))
	return u.String(), nil
}

func resolveDatabase(ctx context.Context, explicit, dataDir string) (string, func(), error) {
	if explicit != "" {
		dsn, err := withPasswordFile(explicit)
		return dsn, func() {}, err
	}
	p, err := paths.New(dataDir)
	if err != nil {
		return "", nil, err
	}
	file, err := config.Load(p.Config())
	if err != nil {
		return "", nil, err
	}
	if file.Database != "" {
		dsn, err := withPasswordFile(file.Database)
		return dsn, func() {}, err
	}
	cfg := bundledConfig(p, slog.Default())

	server, err := pgbundle.Start(ctx, cfg, pgbundle.OwnerCommand)
	var held *pgbundle.HeldError
	if errors.As(err, &held) && held.Owner == pgbundle.OwnerServe {
		dsn, err := pgbundle.Attach(ctx, cfg)
		return dsn, func() {}, err
	}
	if err != nil {
		return "", nil, noDatabase(err)
	}
	return server.DSN(), func() { stopBundled(server) }, nil
}

func stopBundled(server *pgbundle.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), stopBundledWithin)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		slog.Default().Warn("pwikit could not stop its PostgreSQL cleanly", "err", err)
	}
}

func noDatabase(err error) error {
	return fmt.Errorf("%w\n  To use a PostgreSQL of your own instead, pass -database or set %s", err, envDatabase)
}

func startBundled(ctx context.Context, p *paths.Paths, log *slog.Logger) (*pgbundle.Server, error) {
	if pgbundle.DefaultPortTaken(300 * time.Millisecond) {
		fmt.Fprintln(os.Stderr, pgbundle.Hint())
	}
	server, err := pgbundle.Start(ctx, bundledConfig(p, log), pgbundle.OwnerServe)
	if err != nil {
		return nil, noDatabase(err)
	}
	return server, nil
}
