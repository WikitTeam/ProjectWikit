package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/WikitTeam/ProjectWikit/internal/logfile"
)

func openLog(name string) (*slog.Logger, func(), error) {
	if name == "" {
		return slog.Default(), func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create %s: %w", filepath.Dir(name), err)
	}
	w, err := logfile.Open(name, logfile.DefaultLimit, logfile.DefaultKeep)
	if err != nil {
		return nil, nil, err
	}
	log := slog.New(slog.NewTextHandler(w, nil))
	slog.SetDefault(log)
	return log, func() { w.Close() }, nil
}
