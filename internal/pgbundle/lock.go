package pgbundle

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const lockFile = "postgres.lock"

type Owner string

const (
	OwnerServe   Owner = "serve"
	OwnerCommand Owner = "command"
)

type HeldError struct {
	PID   int
	Owner Owner
}

func (e *HeldError) Error() string {
	who := "another pwikit"
	switch e.Owner {
	case OwnerServe:
		who = "pwikit serve"
	case OwnerCommand:
		who = "another pwikit command"
	}
	if e.PID > 0 {
		who += fmt.Sprintf(", process %d,", e.PID)
	}
	return who + " is already running the bundled PostgreSQL in this directory"
}

type Lock struct {
	f *os.File
}

func TryLock(name string, owner Owner) (*Lock, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	if err := lockFD(f); err != nil {
		f.Close()
		if errors.Is(err, errLocked) {
			return nil, readHolder(name)
		}
		return nil, fmt.Errorf("lock %s: %w", name, err)
	}
	if err := f.Truncate(0); err == nil {
		f.WriteAt([]byte(strconv.Itoa(os.Getpid())+" "+string(owner)+"\n"), 0)
	}
	return &Lock{f: f}, nil
}

func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	l.f.Truncate(0)
	err := l.f.Close()
	l.f = nil
	return err
}

func readHolder(name string) error {
	held := &HeldError{}
	raw, err := os.ReadFile(name)
	if err != nil {
		return held
	}
	fields := strings.Fields(string(raw))
	if len(fields) > 0 {
		held.PID, _ = strconv.Atoi(fields[0])
	}
	if len(fields) > 1 {
		held.Owner = Owner(fields[1])
	}
	return held
}
