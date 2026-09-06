package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const usage = `Usage: go run ./tools/baseline > internal/migrate/sql/0001_baseline.sql

Reads pg_dump output on stdin, strips what pgx cannot execute, leaves the rest
verbatim. Feed it two dumps in this order, from two different databases.

  Schema, from a deployed database, because the baseline has to describe the
  schema sites already run:

    pg_dump --schema-only --no-owner --no-privileges --no-comments

  Reference data, from a database built by migrating an empty one, because a
  deployed database has years of edits in these tables:

    pg_dump --data-only --inserts --no-owner --no-privileges --no-comments \
      -t django_content_type -t auth_permission -t web_rolecategory \
      -t web_role -t web_role_permissions -t web_theme

It also makes the test fixture, from the seeded test database, as one dump on
its own:

  go run ./tools/baseline > internal/migrate/testdata/fixture.sql

    pg_dump --data-only --inserts --no-owner --no-privileges --no-comments \
      --exclude-table=django_content_type --exclude-table=auth_permission \
      --exclude-table=web_rolecategory --exclude-table=web_role \
      --exclude-table=web_role_permissions --exclude-table=web_theme \
      --exclude-table=django_migrations --exclude-table=django_session \
      --exclude-table=django_admin_log

Applying the baseline and then that fixture to an empty database rebuilds the
test database. TestFixtureRebuildsTheTestDatabase checks it still does.
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
