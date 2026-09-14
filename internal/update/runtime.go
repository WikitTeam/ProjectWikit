package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	RuntimeFile = "serve.json"
	HealthPath  = "/-/health"
)

type Runtime struct {
	PID       int       `json:"pid"`
	Version   string    `json:"version"`
	StartedAt time.Time `json:"started_at"`
	Health    string    `json:"health"`

	Mode      string   `json:"mode"`
	Plain     string   `json:"plain"`
	Secure    string   `json:"secure"`
	CertFile  string   `json:"cert_file"`
	KeyFile   string   `json:"key_file"`
	CacheDir  string   `json:"cache_dir"`
	Email     string   `json:"email"`
	Directory string   `json:"directory"`
	Hosts     []string `json:"hosts"`
}

func WriteRuntime(dir string, r Runtime) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, RuntimeFile)
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ReadRuntime(dir string) (Runtime, error) {
	body, err := os.ReadFile(filepath.Join(dir, RuntimeFile))
	if err != nil {
		return Runtime{}, err
	}
	var r Runtime
	if err := json.Unmarshal(body, &r); err != nil {
		return Runtime{}, fmt.Errorf("read %s: %w", RuntimeFile, err)
	}
	return r, nil
}

type Health struct {
	Version  string `json:"version"`
	Database bool   `json:"database"`
	Page     int    `json:"page"`
	Problem  string `json:"problem,omitempty"`
}

func (h Health) OK() bool {
	return h.Problem == "" && h.Database && h.Page > 0 && h.Page < 500 && h.Page != http.StatusNotFound
}

func HealthHandler(check func(context.Context) Health) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != HealthPath {
			http.NotFound(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		h := check(ctx)
		w.Header().Set("Content-Type", "application/json")
		if !h.OK() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(h)
	})
}

func ListenHealth() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func WaitHealthy(ctx context.Context, dir, want string, since time.Time, within time.Duration) (Health, error) {
	deadline := time.Now().Add(within)
	client := &http.Client{Timeout: 45 * time.Second}
	last := errors.New("the server has not written its runtime record yet")
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return Health{}, err
		}
		rt, err := ReadRuntime(dir)
		switch {
		case err != nil:
		case rt.StartedAt.Before(since):
			last = errors.New("the server has not started again yet")
		case rt.Version != want:
			return Health{}, fmt.Errorf("the server that started is %s, not %s", rt.Version, want)
		default:
			h, err := probe(ctx, client, rt.Health)
			if err == nil && h.OK() {
				return h, nil
			}
			if err != nil {
				last = err
			} else {
				last = fmt.Errorf("the health check answered database=%t page=%d %s", h.Database, h.Page, h.Problem)
			}
		}
		time.Sleep(2 * time.Second)
	}
	return Health{}, fmt.Errorf("no healthy %s within %s: %w", want, within, last)
}

func probe(ctx context.Context, client *http.Client, addr string) (Health, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+HealthPath, nil)
	if err != nil {
		return Health{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Health{}, err
	}
	defer resp.Body.Close()
	var h Health
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return Health{}, fmt.Errorf("read the health answer: %w", err)
	}
	return h, nil
}
