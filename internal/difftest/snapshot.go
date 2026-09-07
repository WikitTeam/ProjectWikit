package difftest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const snapshotRoot = "testdata/snapshots"

func NewSnapshotRunner(base string) (*Runner, error) { return NewRunner(base, base) }

func SnapshotPath(set string, req Request) string {
	return filepath.Join(snapshotRoot, set, dumpName(req)+".golden")
}

func EncodeSnapshot(resp Response) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%d\n", resp.Status)

	names := make([]string, 0, len(resp.Header))
	for name := range resp.Header {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		for _, value := range resp.Header.Values(name) {
			fmt.Fprintf(&out, "%s: %s\n", name, value)
		}
	}

	out.WriteString("\n")
	out.Write(resp.Body)
	return out.Bytes()
}

func DecodeSnapshot(raw []byte) (Response, error) {
	head, body, ok := bytes.Cut(raw, []byte("\n\n"))
	if !ok {
		return Response{}, fmt.Errorf("snapshot has no blank line before the body")
	}
	lines := strings.Split(string(head), "\n")

	status, err := strconv.Atoi(lines[0])
	if err != nil {
		return Response{}, fmt.Errorf("snapshot status %q: %w", lines[0], err)
	}
	header := http.Header{}
	for _, line := range lines[1:] {
		name, value, ok := strings.Cut(line, ": ")
		if !ok {
			return Response{}, fmt.Errorf("snapshot header line %q has no colon", line)
		}
		header.Add(name, value)
	}
	return Response{Status: status, Header: header, Body: body}, nil
}

// Two targets differing only in case land on one file on a case-insensitive
// filesystem, so the digest of the exact target is part of the name.
func dumpName(req Request) string {
	sum := sha256.Sum256([]byte(req.Method + " " + req.Target))
	name := req.Method + strings.ReplaceAll(req.Target, "/", "_")
	return readable(name) + "." + hex.EncodeToString(sum[:3])
}

func readable(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-' || r == '.':
			return r
		}
		return '-'
	}, name)
}
