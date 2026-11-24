// Package inbox tells a person's open pages that their direct messages changed.
package inbox

import (
	"errors"
	"sync"
)

// A person rarely has more than a few tabs open. The cap keeps one account from
// holding open an unbounded number of connections.
const PerUser = 16

var (
	ErrFull   = errors.New("inbox: too many open streams")
	ErrClosed = errors.New("inbox: closed")
)

type Hub struct {
	mu     sync.Mutex
	subs   map[int64]map[chan struct{}]struct{}
	closed bool
	done   chan struct{}
}

func New() *Hub {
	return &Hub{subs: map[int64]map[chan struct{}]struct{}{}, done: make(chan struct{})}
}

// Each channel holds at most one pending signal, so a burst of changes reaches
// a slow reader as one and Notify never waits on it.
func (h *Hub) Subscribe(user int64) (<-chan struct{}, func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, nil, ErrClosed
	}
	if len(h.subs[user]) >= PerUser {
		return nil, nil, ErrFull
	}
	ch := make(chan struct{}, 1)
	if h.subs[user] == nil {
		h.subs[user] = map[chan struct{}]struct{}{}
	}
	h.subs[user][ch] = struct{}{}
	var once sync.Once
	leave := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(h.subs[user], ch)
			if len(h.subs[user]) == 0 {
				delete(h.subs, user)
			}
		})
	}
	return ch, leave, nil
}

func (h *Hub) Notify(users ...int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, user := range users {
		for ch := range h.subs[user] {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
}

func (h *Hub) Done() <-chan struct{} { return h.done }

// A server shutting down waits for every open request, so the streams are told
// to end rather than holding the shutdown until its deadline.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.closed {
		h.closed = true
		close(h.done)
	}
}
