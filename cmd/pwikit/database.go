package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
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

func resolveDatabase(ctx context.Context, explicit, dataDir string) (string, func(), error) {
	if explicit != "" {
		return explicit, func() {}, nil
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
		return file.Database, func() {}, nil
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
