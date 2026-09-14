package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/entry"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/update"
	"github.com/WikitTeam/ProjectWikit/internal/version"
)

// Set by the official image only. A marker file would also catch a whole
// machine run inside a container, which updates itself like any other.
const envContainer = "PWIKIT_CONTAINER"

func inContainer() bool {
	return os.Getenv(envContainer) != ""
}

func serveHealth(p *paths.Paths, serving entry.Config, conn *db.DB, handler http.Handler, log *slog.Logger) (func(), error) {
	listener, err := update.ListenHealth()
	if err != nil {
		return nil, err
	}
	check := func(ctx context.Context) update.Health {
		h := update.Health{Version: version.String()}
		hosts, err := conn.SiteDomains(ctx)
		if err != nil {
			h.Problem = err.Error()
			return h
		}
		h.Database = true
		if len(hosts) == 0 {
			h.Page = http.StatusOK
			return h
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		req.Host = hosts[0]
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		h.Page = rec.Code
		return h
	}
	server := &http.Server{Handler: update.HealthHandler(check), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn("the update health check stopped answering", "err", err)
		}
	}()

	hosts, _ := conn.SiteHosts(context.Background())
	record := update.Runtime{
		PID:       os.Getpid(),
		Version:   version.String(),
		StartedAt: time.Now().UTC(),
		Health:    listener.Addr().String(),
		Mode:      string(serving.Mode),
		Plain:     serving.Plain,
		Secure:    serving.Secure,
		CertFile:  serving.CertFile,
		KeyFile:   serving.KeyFile,
		CacheDir:  serving.CacheDir,
		Email:     serving.Email,
		Directory: serving.Directory,
		Hosts:     hosts,
	}
	if err := update.WriteRuntime(p.Updates(), record); err != nil {
		server.Close()
		return nil, err
	}
	return func() { server.Close() }, nil
}
