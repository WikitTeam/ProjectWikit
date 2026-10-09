package webapi

import "testing"

func TestOwnTicketRef(t *testing.T) {
	cases := []struct {
		rest string
		kind string
		id   int64
		ok   bool
	}{
		{"ticket/3", "ticket", 3, true},
		{"membershipapply/7", "membershipapply", 7, true},
		{"report/9", "report", 9, true},
		{"other/3", "", 0, false},
		{"ticket/x", "", 0, false},
		{"ticket/0", "", 0, false},
		{"ticket", "", 0, false},
		{"ticket/3/4", "", 0, false},
	}
	for _, c := range cases {
		kind, id, ok := ownTicketRef(c.rest)
		if kind != c.kind || id != c.id || ok != c.ok {
			t.Errorf("ownTicketRef(%q) = %q, %d, %v, want %q, %d, %v", c.rest, kind, id, ok, c.kind, c.id, c.ok)
		}
	}
}

func TestReportedMessages(t *testing.T) {
	got := reportedMessages(`[{"sender_name":"bob","body":"hi","created_at":"2026-10-01T00:00:00Z"}]`)
	if len(got) != 1 {
		t.Fatalf("len(reportedMessages()) = %d, want 1", len(got))
	}
	if len(reportedMessages("not json")) != 0 {
		t.Errorf("reportedMessages(bad) has rows, want none")
	}
	if len(reportedMessages("")) != 0 {
		t.Errorf(`reportedMessages("") has rows, want none`)
	}
}
