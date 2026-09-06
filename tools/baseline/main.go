package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const usage = `Usage: pg_dump --schema-only --no-owner --no-privileges --no-comments | go run ./tools/baseline > internal/migrate/sql/0001_baseline.sql

Strips what pgx cannot execute from a schema-only dump and leaves the rest verbatim.
`

func main() {
	if len(os.Args) > 1 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err := clean(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "baseline: "+err.Error())
		os.Exit(1)
	}
}

func clean(r io.Reader, w io.Writer) error {
	in := bufio.NewScanner(r)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	out := bufio.NewWriter(w)

	statements := 0
	blank := true
	for in.Scan() {
		line := strings.TrimRight(in.Text(), " \t\r")
		if drop(line) {
			continue
		}
		if line == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
			statements++
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	if err := in.Err(); err != nil {
		return err
	}
	if statements == 0 {
		return fmt.Errorf("no statements on stdin")
	}
	return out.Flush()
}

func drop(line string) bool {
	switch {
	case strings.HasPrefix(line, `\`):
		return true
	case strings.HasPrefix(line, "--"):
		return true
	case strings.HasPrefix(line, "SET "):
		return true
	case strings.HasPrefix(line, "SELECT pg_catalog.set_config("):
		return true
	}
	return false
}
