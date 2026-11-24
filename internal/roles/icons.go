package roles

import (
	"context"
	"sync"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

const iconsKept = time.Minute

// Every user link on a page reads its role's icon, and a bucket answers each
// read over the network, so an icon is kept for a minute once read.
func FileIcons(store storage.Store) IconLoader {
	type kept struct {
		body string
		err  error
		at   time.Time
	}
	var mu sync.Mutex
	cache := map[string]kept{}
	return func(name string) (string, error) {
		mu.Lock()
		if k, ok := cache[name]; ok && time.Since(k.at) < iconsKept {
			mu.Unlock()
			return k.body, k.err
		}
		mu.Unlock()
		data, err := storage.ReadAll(context.Background(), store, name)
		mu.Lock()
		cache[name] = kept{body: string(data), err: err, at: time.Now()}
		mu.Unlock()
		return string(data), err
	}
}
