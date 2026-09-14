package admin

import (
	"context"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

func TestNoteRecordsTheSiteOfTheRequest(t *testing.T) {
	dsn := os.Getenv(db.EnvWriteDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the write test", db.EnvWriteDSN)
	}
	ctx := context.Background()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db.Open() err = %v, want nil", err)
	}
	t.Cleanup(conn.Close)
	current, err := conn.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}

	screen := "probe-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() {
		raw, err := pgx.Connect(context.Background(), dsn)
		if err != nil {
			t.Errorf("pgx.Connect() err = %v, want nil", err)
			return
		}
		defer raw.Close(context.Background())
		if _, err := raw.Exec(context.Background(), `DELETE FROM pwikit_admin_log WHERE screen = $1`, screen); err != nil {
			t.Errorf("clean up admin log err = %v, want nil", err)
		}
	})

	h := &Handler{deps: Deps{DB: conn}}
	r := httptest.NewRequest("POST", "/-/admin/", nil).WithContext(site.WithSite(ctx, current))
	h.note(r, db.AdminChanged, screen, "", "probe")

	screens, err := conn.AdminNoteScreens(ctx, current.ID)
	if err != nil {
		t.Fatalf("AdminNoteScreens() err = %v, want nil", err)
	}
	if !slices.Contains(screens, screen) {
		t.Errorf("AdminNoteScreens(%d) = %v, want it to contain %q", current.ID, screens, screen)
	}
}
