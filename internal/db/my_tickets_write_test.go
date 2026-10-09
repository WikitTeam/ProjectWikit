package db

import (
	"context"
	"errors"
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

func TestOwnTicketDetailOnlyForTheOwner(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := scratchSite(t, d)
	author := scratchUser(t, d, "probe-own-detail")
	other := scratchUser(t, d, "probe-own-detail-other")
	now := time.Now().UTC().Truncate(time.Second)

	ticket, err := d.CreateTicket(ctx, site, TicketKind, "help", "the body", "start", author, now)
	if err != nil {
		t.Fatalf("CreateTicket() err = %v, want nil", err)
	}
	report, err := d.CreateReport(ctx, site, author, other, "spam", `[{"sender_name":"x","body":"hi"}]`, now)
	if err != nil {
		t.Fatalf("CreateReport() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_userticket WHERE id = $1`, ticket)
		d.pool.Exec(context.Background(), `DELETE FROM web_userreport WHERE id = $1`, report)
	})

	got, err := d.OwnTicketDetail(ctx, author, TicketKind, ticket)
	if err != nil {
		t.Fatalf("OwnTicketDetail(ticket) err = %v, want nil", err)
	}
	if got.Body != "the body" {
		t.Errorf("OwnTicketDetail(ticket).Body = %q, want %q", got.Body, "the body")
	}
	if got.SourcePage != "start" {
		t.Errorf("OwnTicketDetail(ticket).SourcePage = %q, want %q", got.SourcePage, "start")
	}
	rep, err := d.OwnTicketDetail(ctx, author, ReportKind, report)
	if err != nil {
		t.Fatalf("OwnTicketDetail(report) err = %v, want nil", err)
	}
	if rep.Body != "spam" {
		t.Errorf("OwnTicketDetail(report).Body = %q, want %q", rep.Body, "spam")
	}
	if rep.Messages == "" {
		t.Error(`OwnTicketDetail(report).Messages = "", want the snapshot`)
	}
	if _, err := d.OwnTicketDetail(ctx, other, TicketKind, ticket); !errors.Is(err, ErrNotFound) {
		t.Errorf("OwnTicketDetail(someone else's ticket) err = %v, want ErrNotFound", err)
	}
	if _, err := d.OwnTicketDetail(ctx, other, ReportKind, report); !errors.Is(err, ErrNotFound) {
		t.Errorf("OwnTicketDetail(someone else's report) err = %v, want ErrNotFound", err)
	}
	if _, err := d.OwnTicketDetail(ctx, author, "membershipapply", ticket); !errors.Is(err, ErrNotFound) {
		t.Errorf("OwnTicketDetail(wrong kind) err = %v, want ErrNotFound", err)
	}
}

func TestHideOwnTicketsOnlyHidesForTheSubmitter(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := scratchSite(t, d)
	author := scratchUser(t, d, "probe-own-hide")
	other := scratchUser(t, d, "probe-own-hide-other")
	now := time.Now().UTC().Truncate(time.Second)

	ticket, err := d.CreateTicket(ctx, site, TicketKind, "help", "body", "", author, now)
	if err != nil {
		t.Fatalf("CreateTicket() err = %v, want nil", err)
	}
	report, err := d.CreateReport(ctx, site, author, other, "spam", "[]", now)
	if err != nil {
		t.Fatalf("CreateReport() err = %v, want nil", err)
	}
	foreign, err := d.CreateTicket(ctx, site, TicketKind, "theirs", "body", "", other, now)
	if err != nil {
		t.Fatalf("CreateTicket(other) err = %v, want nil", err)
	}
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_userticket WHERE id = ANY($1)`, []int64{ticket, foreign})
		d.pool.Exec(context.Background(), `DELETE FROM web_userreport WHERE id = $1`, report)
	})

	hidden, err := d.HideOwnTickets(ctx, author, []OwnTicketRef{{Kind: TicketKind, ID: ticket}, {Kind: TicketKind, ID: foreign}}, now)
	if err != nil {
		t.Fatalf("HideOwnTickets() err = %v, want nil", err)
	}
	if hidden != 1 {
		t.Errorf("HideOwnTickets() = %d, want 1", hidden)
	}
	if n, _ := d.OwnTicketCount(ctx, author); n != 1 {
		t.Errorf("OwnTicketCount(after hiding one) = %d, want 1", n)
	}
	if _, err := d.OwnTicketDetail(ctx, author, TicketKind, ticket); !errors.Is(err, ErrNotFound) {
		t.Errorf("OwnTicketDetail(hidden) err = %v, want ErrNotFound", err)
	}
	if _, err := d.AdminTicket(ctx, site, ticket); err != nil {
		t.Errorf("AdminTicket(hidden by submitter) err = %v, want nil", err)
	}
	if n, _ := d.OwnTicketCount(ctx, other); n != 1 {
		t.Errorf("OwnTicketCount(other submitter) = %d, want 1", n)
	}
	if _, err := d.HideAllOwnTickets(ctx, author, now); err != nil {
		t.Fatalf("HideAllOwnTickets() err = %v, want nil", err)
	}
	if n, _ := d.OwnTicketCount(ctx, author); n != 0 {
		t.Errorf("OwnTicketCount(after hiding all) = %d, want 0", n)
	}
	if _, err := d.AdminReport(ctx, site, report); err != nil {
		t.Errorf("AdminReport(hidden by reporter) err = %v, want nil", err)
	}
}
