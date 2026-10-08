package pgbundle

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeldDespiteUnreadableLockWithReadyServer(t *testing.T) {
	data := t.TempDir()
	pid := "4242\n" + strings.Repeat("x\n", 6) + "ready   \n"
	if err := os.WriteFile(filepath.Join(data, postmasterFile), []byte(pid), 0o600); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
	denied := fmt.Errorf("open postgres.lock: %w", fs.ErrPermission)
	var held *HeldError
	if err := heldDespite(denied, data); !errors.As(err, &held) || held.Owner != OwnerServe {
		t.Errorf("heldDespite(denied, ready) = %v, want a HeldError from serve", err)
	}
}

func TestHeldDespiteUnreadableLockWithoutServer(t *testing.T) {
	denied := fmt.Errorf("open postgres.lock: %w", fs.ErrPermission)
	if err := heldDespite(denied, t.TempDir()); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("heldDespite(denied, no server) = %v, want the permission error", err)
	}
}

func TestHeldDespiteOtherError(t *testing.T) {
	other := errors.New("disk full")
	if err := heldDespite(other, t.TempDir()); err != other {
		t.Errorf("heldDespite(other) = %v, want %v", err, other)
	}
}
