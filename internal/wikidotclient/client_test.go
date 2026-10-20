package wikidotclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type fakeWikidot struct {
	mu        sync.Mutex
	logins    int
	acceptPW  string
	sessionOK bool
	statuses  []string
	sent      []string
	lookup    string
}

func (f *fakeWikidot) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/default--flow/login__LoginPopupScreen", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.logins++
		f.sessionOK = r.PostFormValue("password") == f.acceptPW
	})
	mux.HandleFunc("/account/activity", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.sessionOK {
			_, _ = w.Write([]byte("<h1>Recent activity</h1>"))
			return
		}
		_, _ = w.Write([]byte("<a>Sign in</a>"))
	})
	mux.HandleFunc("/quickmodule.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(f.lookup))
	})
	mux.HandleFunc("/ajax-module-connector.php", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		cookie, err := r.Cookie(tokenCookie)
		if err != nil || cookie.Value != r.PostFormValue("wikidot_token7") {
			_, _ = w.Write([]byte(`{"status":"wrong_token7"}`))
			return
		}
		status := "ok"
		if len(f.statuses) > 0 {
			status, f.statuses = f.statuses[0], f.statuses[1:]
		}
		if status == "ok" {
			f.sent = append(f.sent, r.PostFormValue("to_user_id"))
		}
		_, _ = w.Write([]byte(`{"status":"` + status + `"}`))
	})
	return mux
}

func newFake(t *testing.T) (*fakeWikidot, *Client, *time.Time) {
	t.Helper()
	fake := &fakeWikidot{acceptPW: "right"}
	server := httptest.NewServer(fake.handler())
	t.Cleanup(server.Close)
	client := New(server.URL, "bot", "right")
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	client.now = func() time.Time { return now }
	return fake, client, &now
}

func TestLookUp(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		query  string
		want   int64
		err    error
	}{
		{"exact", `{"users":[{"name":"AceAttackAssassin","user_id":"8305255"}]}`, "AceAttackAssassin", 8305255, nil},
		{"case differs", `{"users":[{"name":"AshlyS_M","user_id":9777891}]}`, "ashlys_m", 9777891, nil},
		{"trailing space in wikidot", `{"users":[{"name":"KAKUSHITSU ","user_id":"42"}]}`, "KAKUSHITSU", 42, nil},
		{"exact beats folded", `{"users":[{"name":"ab","user_id":"1"},{"name":"AB","user_id":"2"}]}`, "AB", 2, nil},
		{"two folded", `{"users":[{"name":"ab","user_id":"1"},{"name":"Ab","user_id":"2"}]}`, "AB", 0, ErrNoUser},
		{"none", `{"users":false}`, "nobody", 0, ErrNoUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, client, _ := newFake(t)
			fake.lookup = tt.answer
			got, err := client.LookUp(context.Background(), tt.query)
			if !errors.Is(err, tt.err) {
				t.Fatalf("LookUp(%q) err = %v, want %v", tt.query, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("LookUp(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestSendSignsInOnce(t *testing.T) {
	fake, client, _ := newFake(t)
	for i := 0; i < 2; i++ {
		if err := client.Send(context.Background(), 7, "s", "b"); err != nil {
			t.Fatalf("Send() err = %v, want nil", err)
		}
	}
	if fake.logins != 1 {
		t.Errorf("logins = %d, want 1", fake.logins)
	}
	if len(fake.sent) != 2 {
		t.Errorf("sent = %d, want 2", len(fake.sent))
	}
}

func TestSendRefused(t *testing.T) {
	fake, client, _ := newFake(t)
	fake.statuses = []string{"no_permission"}
	err := client.Send(context.Background(), 7, "s", "b")
	if !errors.Is(err, ErrRefused) {
		t.Errorf("Send() err = %v, want ErrRefused", err)
	}
	if fake.logins != 1 {
		t.Errorf("logins = %d, want 1", fake.logins)
	}
}

func TestSendSignsInAgainWhenSignedOut(t *testing.T) {
	fake, client, _ := newFake(t)
	if err := client.Send(context.Background(), 7, "s", "b"); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	fake.sessionOK = false
	fake.statuses = []string{"no_permission"}
	if err := client.Send(context.Background(), 7, "s", "b"); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	if fake.logins != 2 {
		t.Errorf("logins = %d, want 2", fake.logins)
	}
}

func TestSendWaitsAfterAFailedSignIn(t *testing.T) {
	fake, client, now := newFake(t)
	fake.acceptPW = "other"
	if err := client.Send(context.Background(), 7, "s", "b"); !errors.Is(err, ErrLogin) || errors.Is(err, ErrPaused) {
		t.Fatalf("Send() err = %v, want ErrLogin", err)
	}
	if err := client.Send(context.Background(), 7, "s", "b"); !errors.Is(err, ErrPaused) {
		t.Fatalf("Send() during the pause err = %v, want ErrPaused", err)
	}
	if fake.logins != 1 {
		t.Errorf("logins = %d, want 1", fake.logins)
	}
	*now = now.Add(2 * loginPause)
	fake.acceptPW = "right"
	if err := client.Send(context.Background(), 7, "s", "b"); err != nil {
		t.Errorf("Send() err = %v, want nil", err)
	}
}

func TestSendUnreachable(t *testing.T) {
	client := New("http://127.0.0.1:1", "bot", "pw")
	if err := client.Send(context.Background(), 7, "s", "b"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Send() err = %v, want ErrUnavailable", err)
	}
}
