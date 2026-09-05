package db

import (
	"context"
	"testing"
	"time"
)

func TestSendDirectMessage(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	sender := scratchUser(t, d, "probe-dm-a")
	recipient := scratchUser(t, d, "probe-dm-b")

	sent, err := d.SendDirectMessage(ctx, sender, recipient, "hello", time.Now().UTC())
	if err != nil {
		t.Fatalf("SendDirectMessage() err = %v, want nil", err)
	}
	if sent.ID == 0 {
		t.Errorf("SendDirectMessage().ID = 0, want a row id")
	}

	found, err := d.ConversationBefore(ctx, recipient, sender, nil, 10)
	if err != nil {
		t.Fatalf("ConversationBefore() err = %v, want nil", err)
	}
	if len(found) != 1 {
		t.Fatalf("len(ConversationBefore()) = %d, want 1", len(found))
	}
	if found[0].Body != "hello" {
		t.Errorf("ConversationBefore()[0].Body = %q, want %q", found[0].Body, "hello")
	}
	if found[0].IsRead {
		t.Errorf("ConversationBefore()[0].IsRead = true, want false")
	}
}

func TestMarkConversationRead(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	sender := scratchUser(t, d, "probe-dm-read-a")
	recipient := scratchUser(t, d, "probe-dm-read-b")
	if _, err := d.SendDirectMessage(ctx, sender, recipient, "one", time.Now().UTC()); err != nil {
		t.Fatalf("SendDirectMessage() err = %v, want nil", err)
	}

	read, err := d.MarkConversationRead(ctx, recipient, sender)
	if err != nil {
		t.Fatalf("MarkConversationRead() err = %v, want nil", err)
	}
	if read != 1 {
		t.Errorf("MarkConversationRead() = %d, want 1", read)
	}

	unread, err := d.UnreadMessages(ctx, recipient)
	if err != nil {
		t.Fatalf("UnreadMessages() err = %v, want nil", err)
	}
	if unread != 0 {
		t.Errorf("UnreadMessages() = %d, want 0", unread)
	}
}

func TestMarkConversationReadLeavesTheOtherDirection(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	a := scratchUser(t, d, "probe-dm-dir-a")
	b := scratchUser(t, d, "probe-dm-dir-b")
	if _, err := d.SendDirectMessage(ctx, a, b, "to b", time.Now().UTC()); err != nil {
		t.Fatalf("SendDirectMessage() err = %v, want nil", err)
	}
	if _, err := d.SendDirectMessage(ctx, b, a, "to a", time.Now().UTC()); err != nil {
		t.Fatalf("SendDirectMessage() err = %v, want nil", err)
	}

	if _, err := d.MarkConversationRead(ctx, b, a); err != nil {
		t.Fatalf("MarkConversationRead() err = %v, want nil", err)
	}
	unread, err := d.UnreadMessages(ctx, a)
	if err != nil {
		t.Fatalf("UnreadMessages() err = %v, want nil", err)
	}
	if unread != 1 {
		t.Errorf("UnreadMessages(a) = %d, want 1", unread)
	}
}

func TestConversationsCarryTheLastMessage(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	a := scratchUser(t, d, "probe-dm-list-a")
	b := scratchUser(t, d, "probe-dm-list-b")
	for _, body := range []string{"first", "second"} {
		if _, err := d.SendDirectMessage(ctx, a, b, body, time.Now().UTC()); err != nil {
			t.Fatalf("SendDirectMessage(%q) err = %v, want nil", body, err)
		}
	}

	found, err := d.Conversations(ctx, b)
	if err != nil {
		t.Fatalf("Conversations() err = %v, want nil", err)
	}
	if len(found) != 1 {
		t.Fatalf("len(Conversations()) = %d, want 1", len(found))
	}
	if found[0].PartnerID != a {
		t.Errorf("Conversations()[0].PartnerID = %d, want %d", found[0].PartnerID, a)
	}
	if found[0].Last.Body != "second" {
		t.Errorf("Conversations()[0].Last.Body = %q, want %q", found[0].Last.Body, "second")
	}
	if found[0].Unread != 2 {
		t.Errorf("Conversations()[0].Unread = %d, want 2", found[0].Unread)
	}
}

func TestMessagesByIDsRefusesAnotherPair(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	a := scratchUser(t, d, "probe-dm-pair-a")
	b := scratchUser(t, d, "probe-dm-pair-b")
	c := scratchUser(t, d, "probe-dm-pair-c")
	sent, err := d.SendDirectMessage(ctx, a, b, "private", time.Now().UTC())
	if err != nil {
		t.Fatalf("SendDirectMessage() err = %v, want nil", err)
	}

	found, err := d.MessagesByIDs(ctx, a, c, []int64{sent.ID})
	if err != nil {
		t.Fatalf("MessagesByIDs() err = %v, want nil", err)
	}
	if len(found) != 0 {
		t.Errorf("len(MessagesByIDs()) = %d, want 0", len(found))
	}
}

func TestBlockUserOnlyOnce(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	blocker := scratchUser(t, d, "probe-block-a")
	blocked := scratchUser(t, d, "probe-block-b")

	first, err := d.BlockUser(ctx, blocker, blocked, time.Now().UTC())
	if err != nil {
		t.Fatalf("BlockUser() err = %v, want nil", err)
	}
	if !first {
		t.Errorf("BlockUser() = false, want true")
	}
	again, err := d.BlockUser(ctx, blocker, blocked, time.Now().UTC())
	if err != nil {
		t.Fatalf("BlockUser() err = %v, want nil", err)
	}
	if again {
		t.Errorf("BlockUser() = true, want false")
	}

	blockedNow, err := d.IsBlocked(ctx, blocker, blocked)
	if err != nil {
		t.Fatalf("IsBlocked() err = %v, want nil", err)
	}
	if !blockedNow {
		t.Errorf("IsBlocked() = false, want true")
	}

	gone, err := d.UnblockUser(ctx, blocker, blocked)
	if err != nil {
		t.Fatalf("UnblockUser() err = %v, want nil", err)
	}
	if !gone {
		t.Errorf("UnblockUser() = false, want true")
	}
}

func TestCreateReport(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	reporter := scratchUser(t, d, "probe-report-a")
	reported := scratchUser(t, d, "probe-report-b")

	id, err := d.CreateReport(ctx, reporter, reported, "spam", `[{"id":1}]`, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateReport() err = %v, want nil", err)
	}
	got, err := d.Report(ctx, id)
	if err != nil {
		t.Fatalf("Report() err = %v, want nil", err)
	}
	if got.Reason != "spam" {
		t.Errorf("Report().Reason = %q, want %q", got.Reason, "spam")
	}
	if got.Status != ReportPending {
		t.Errorf("Report().Status = %q, want %q", got.Status, ReportPending)
	}

	count, err := d.ReportsSince(ctx, reporter, reported, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("ReportsSince() err = %v, want nil", err)
	}
	if count != 1 {
		t.Errorf("ReportsSince() = %d, want 1", count)
	}
}
