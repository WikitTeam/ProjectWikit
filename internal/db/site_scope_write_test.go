package db

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func scratchSite(t *testing.T, d *DB) int64 {
	t.Helper()
	ctx := context.Background()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	id, err := d.CreateSite(ctx, NewSite{
		Slug: "probe-" + stamp, Title: "Probe", Headline: "Probe",
		Domain: "probe-" + stamp + ".test", MediaDomain: "probe-" + stamp + ".test",
	})
	if err != nil {
		t.Fatalf("CreateSite() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_user_roles WHERE role_id IN (SELECT id FROM web_role WHERE site_id = $1)`,
			`DELETE FROM web_role_permissions WHERE role_id IN (SELECT id FROM web_role WHERE site_id = $1)`,
			`DELETE FROM web_role WHERE site_id = $1`,
			`DELETE FROM web_settings WHERE site_id = $1`,
			`DELETE FROM pwikit_admin_log WHERE site_id = $1`,
			`DELETE FROM web_userreport WHERE site_id = $1`,
			`DELETE FROM web_site WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(ctx, sql, id); err != nil {
				t.Errorf("clean up site %d err = %v, want nil", id, err)
			}
		}
	})
	return id
}

func scratchRole(t *testing.T, d *DB, siteID int64) int64 {
	t.Helper()
	var id int64
	slug := "probe-role-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := d.pool.QueryRow(context.Background(), qInsertBuiltInRole, siteID, slug, 99).Scan(&id); err != nil {
		t.Fatalf("insert role err = %v, want nil", err)
	}
	return id
}

func heldRoles(t *testing.T, d *DB, userID int64) map[int64]bool {
	t.Helper()
	rows, err := d.pool.Query(context.Background(), `SELECT role_id FROM web_user_roles WHERE user_id = $1`, userID)
	if err != nil {
		t.Fatalf("read roles err = %v, want nil", err)
	}
	defer rows.Close()
	held := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		held[id] = true
	}
	return held
}

func TestSaveAdminUserLeavesRolesOnOtherSites(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	here, there := scratchSite(t, d), scratchSite(t, d)
	roleHere, roleThere := scratchRole(t, d, here), scratchRole(t, d, there)

	userID, err := d.CreateUser(ctx, scratchName(t), "Probe Roles", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, userID)
	for site, role := range map[int64]int64{here: roleHere, there: roleThere} {
		if err := d.GrantRole(ctx, site, userID, role); err != nil {
			t.Fatalf("GrantRole(%d) err = %v, want nil", role, err)
		}
	}

	row, err := d.AdminUser(ctx, here, userID)
	if err != nil {
		t.Fatalf("AdminUser() err = %v, want nil", err)
	}
	if len(row.Roles) != 1 || row.Roles[0] != roleHere {
		t.Errorf("AdminUser(here).Roles = %v, want [%d]", row.Roles, roleHere)
	}

	row.Roles = nil
	if err := d.SaveAdminUser(ctx, here, row, []string{"registered", "everyone"}, true, false); err != nil {
		t.Fatalf("SaveAdminUser() err = %v, want nil", err)
	}
	held := heldRoles(t, d, userID)
	if held[roleHere] {
		t.Errorf("role %d on this site after clearing = held, want removed", roleHere)
	}
	if !held[roleThere] {
		t.Errorf("role %d on the other site after clearing here = removed, want held", roleThere)
	}
}

func TestReviewReportStoresTheDecision(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := scratchSite(t, d)
	var id int64
	err := d.pool.QueryRow(ctx, `
INSERT INTO web_userreport (reason, reported_messages, status, admin_notes, created_at, site_id)
VALUES ('probe', '[]', $1, '', now(), $2) RETURNING id`, ReportPending, site).Scan(&id)
	if err != nil {
		t.Fatalf("insert report err = %v, want nil", err)
	}

	userID, err := d.CreateUser(ctx, scratchName(t), "Probe Reviewer", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, userID)
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_userreport WHERE id = $1`, id)
	})

	if err := d.ReviewReport(ctx, site, id, ReportReviewed, "handled", userID, time.Now().UTC()); err != nil {
		t.Fatalf("ReviewReport() err = %v, want nil", err)
	}
	got, err := d.AdminReport(ctx, site, id)
	if err != nil {
		t.Fatalf("AdminReport() err = %v, want nil", err)
	}
	if got.Status != ReportReviewed {
		t.Errorf("AdminReport().Status = %q, want %q", got.Status, ReportReviewed)
	}
	if got.AdminNotes != "handled" {
		t.Errorf("AdminReport().AdminNotes = %q, want %q", got.AdminNotes, "handled")
	}
}

func TestUserByVerifiedEmailSkipsAnUnverifiedAddress(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	email := scratchName(t) + "@example.invalid"

	typo, err := d.CreateUser(ctx, scratchName(t)+"-a", "Probe Typo", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, typo)
	if err := d.SetEmail(ctx, typo, email); err != nil {
		t.Fatalf("SetEmail() err = %v, want nil", err)
	}

	if _, err := d.UserByVerifiedEmail(ctx, email); !errors.Is(err, ErrNotFound) {
		t.Errorf("UserByVerifiedEmail() with only an unverified holder err = %v, want ErrNotFound", err)
	}

	owner, err := d.CreateUser(ctx, scratchName(t)+"-b", "Probe Owner", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, owner)
	if err := d.SetEmail(ctx, owner, email); err != nil {
		t.Fatalf("SetEmail() err = %v, want nil", err)
	}
	if ok, err := d.MarkEmailVerified(ctx, owner, email, time.Now().UTC()); err != nil || !ok {
		t.Fatalf("MarkEmailVerified() = %t, %v, want true, nil", ok, err)
	}

	got, err := d.UserByVerifiedEmail(ctx, email)
	if err != nil {
		t.Fatalf("UserByVerifiedEmail() err = %v, want nil", err)
	}
	if got.ID != owner {
		t.Errorf("UserByVerifiedEmail().ID = %d, want %d", got.ID, owner)
	}
}
