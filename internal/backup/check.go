package backup

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

// This only answers for a server somebody else runs. The bundled one is read
// off its own data directory before it is even started.
func CheckServer(ctx context.Context, dsn string) (int, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return 0, fmt.Errorf("connect to check the postgres version: %w", err)
	}
	defer conn.Close(ctx)

	found, err := db.ServerVersion(ctx, conn)
	if err != nil {
		return 0, err
	}
	if found < MinimumPGVersion {
		return found, TooOld(found)
	}
	return found, nil
}
