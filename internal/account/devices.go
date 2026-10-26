package account

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

func (h *SettingsHandler) devices(r *http.Request, user *db.User) (string, error) {
	ctx := r.Context()
	if r.PostFormValue("action") == "others" {
		ended, err := h.deps.DB.EndOtherDevices(ctx, user.ID, currentKey(r))
		if err != nil {
			return "", err
		}
		h.deps.logger().Info("sign out other devices", "user", user.ID, "count", ended)
		return "devices-others-ended", nil
	}
	id, err := strconv.ParseInt(r.PostFormValue("device"), 10, 64)
	if err != nil {
		return "devices-gone", nil
	}
	done, err := h.deps.DB.EndDevice(ctx, user.ID, id)
	if err != nil || !done {
		return "devices-gone", err
	}
	h.deps.logger().Info("sign out device", "user", user.ID, "device", id)
	return "devices-ended", nil
}

func currentKey(r *http.Request) string {
	if c, err := r.Cookie(session.CookieName); err == nil {
		return c.Value
	}
	return ""
}

func (d Deps) siteID(r *http.Request) *int64 {
	if s := site.FromContext(r.Context()); s != nil {
		return &s.ID
	}
	return nil
}

// Every signed-in request passes through here, so a session seen recently
// costs no query at all.
type DeviceTracker struct {
	deps Deps
	mu   sync.Mutex
	seen map[string]time.Time
}

func NewDeviceTracker(d Deps) *DeviceTracker {
	return &DeviceTracker{deps: d, seen: map[string]time.Time{}}
}

const trackedLimit = 50000

func (t *DeviceTracker) Seen(r *http.Request, key string, user *db.User) {
	now := time.Now()
	t.mu.Lock()
	if last, ok := t.seen[key]; ok && now.Sub(last) < db.DeviceQuiet {
		t.mu.Unlock()
		return
	}
	if len(t.seen) >= trackedLimit {
		for k, at := range t.seen {
			if now.Sub(at) >= db.DeviceQuiet {
				delete(t.seen, k)
			}
		}
	}
	t.seen[key] = now
	t.mu.Unlock()

	if err := t.deps.DB.SeenDevice(r.Context(), key, user.ID, t.deps.siteID(r), false,
		t.deps.clientIP(r), r.UserAgent(), now); err != nil {
		t.deps.logger().Error("record device", "user", user.ID, "err", err)
	}
}

type Agent struct {
	Browser string
	System  string
}

var browsers = []struct{ mark, name string }{
	{"Edg", "Edge"}, {"OPR/", "Opera"}, {"Opera", "Opera"}, {"SamsungBrowser", "Samsung Internet"},
	{"FxiOS", "Firefox"}, {"Firefox/", "Firefox"}, {"CriOS", "Chrome"}, {"Chrome/", "Chrome"},
	{"Safari/", "Safari"},
}

var systems = []struct{ mark, name string }{
	{"iPhone", "iPhone"}, {"iPad", "iPad"}, {"Android", "Android"}, {"Windows", "Windows"},
	{"CrOS", "ChromeOS"}, {"Mac OS X", "macOS"}, {"Macintosh", "macOS"}, {"Linux", "Linux"},
}

func ParseAgent(ua string) Agent {
	var out Agent
	for _, b := range browsers {
		if strings.Contains(ua, b.mark) {
			out.Browser = b.name
			break
		}
	}
	for _, s := range systems {
		if strings.Contains(ua, s.mark) {
			out.System = s.name
			break
		}
	}
	return out
}
