package db

import (
	"context"
	"testing"
	"time"
)

func TestOwnTicketsListsTicketsAndReports(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := scratchSite(t, d)
	author := scratchUser(t, d, "probe-own-tickets")
	other := scratchUser(t, d, "probe-own-tickets-other")
	staff := scratchUser(t, d, "probe-own-tickets-staff")
	now := time.Now().UTC().Truncate(time.Second)

	ticket, err := d.CreateTicket(ctx, site, TicketKind, "help", "body", "", author, now)
	if err != nil {
		t.Fatalf("CreateTicket() err = %v, want nil", err)
	}
	report, err := d.CreateReport(ctx, site, author, other, "spam", "[]", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateReport() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_userticket WHERE id = $1`, ticket)
		d.pool.Exec(context.Background(), `DELETE FROM web_userreport WHERE id = $1`, report)
	})
	if err := d.ReviewTicket(ctx, site, ticket, TicketApproved, "internal", "done", staff, nil, now); err != nil {
		t.Fatalf("ReviewTicket() err = %v, want nil", err)
	}

	total, err := d.OwnTicketCount(ctx, author)
	if err != nil || total != 2 {
		t.Fatalf("OwnTicketCount() = %d, %v, want 2, nil", total, err)
	}
	got, err := d.OwnTickets(ctx, author, 0, 10)
	if err != nil {
		t.Fatalf("OwnTickets() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(OwnTickets()) = %d, want 2", len(got))
	}
	if got[0].Kind != ReportKind || got[0].ID != report {
		t.Errorf("OwnTickets()[0] = %s %d, want %s %d", got[0].Kind, got[0].ID, ReportKind, report)
	}
	if got[1].Status != TicketApproved || got[1].Reply != "done" {
		t.Errorf("OwnTickets()[1] = %q %q, want %q %q", got[1].Status, got[1].Reply, TicketApproved, "done")
	}
	if n, err := d.OwnTicketCount(ctx, other); err != nil || n != 0 {
		t.Errorf("OwnTicketCount(reported user) = %d, %v, want 0, nil", n, err)
	}
}
