package inbox

import (
	"errors"
	"testing"
)

func pending(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestNotifyReachesOnlyThatUser(t *testing.T) {
	h := New()
	mine, leaveMine, err := h.Subscribe(1)
	if err != nil {
		t.Fatalf("Subscribe(1) err = %v, want nil", err)
	}
	defer leaveMine()
	theirs, leaveTheirs, err := h.Subscribe(2)
	if err != nil {
		t.Fatalf("Subscribe(2) err = %v, want nil", err)
	}
	defer leaveTheirs()

	h.Notify(1)
	if !pending(mine) {
		t.Errorf("pending(user 1) = false, want true")
	}
	if pending(theirs) {
		t.Errorf("pending(user 2) = true, want false")
	}
}

func TestNotifyCoalesces(t *testing.T) {
	h := New()
	ch, leave, _ := h.Subscribe(1)
	defer leave()
	h.Notify(1)
	h.Notify(1, 1)
	if !pending(ch) {
		t.Errorf("first pending() = false, want true")
	}
	if pending(ch) {
		t.Errorf("second pending() = true, want false")
	}
}

func TestLeaveStopsSignals(t *testing.T) {
	h := New()
	ch, leave, _ := h.Subscribe(1)
	leave()
	leave()
	h.Notify(1)
	if pending(ch) {
		t.Errorf("pending() after leave = true, want false")
	}
	if len(h.subs) != 0 {
		t.Errorf("len(subs) = %d, want 0", len(h.subs))
	}
}

func TestSubscribeFull(t *testing.T) {
	h := New()
	for i := 0; i < PerUser; i++ {
		if _, _, err := h.Subscribe(1); err != nil {
			t.Fatalf("Subscribe() #%d err = %v, want nil", i, err)
		}
	}
	if _, _, err := h.Subscribe(1); !errors.Is(err, ErrFull) {
		t.Errorf("Subscribe() past cap err = %v, want ErrFull", err)
	}
	if _, _, err := h.Subscribe(2); err != nil {
		t.Errorf("Subscribe(2) err = %v, want nil", err)
	}
}

func TestClose(t *testing.T) {
	h := New()
	h.Close()
	h.Close()
	select {
	case <-h.Done():
	default:
		t.Errorf("Done() open after Close, want closed")
	}
	if _, _, err := h.Subscribe(1); !errors.Is(err, ErrClosed) {
		t.Errorf("Subscribe() after Close err = %v, want ErrClosed", err)
	}
}
