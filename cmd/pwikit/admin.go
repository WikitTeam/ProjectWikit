package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/password"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

func adminCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	if sub != "create" && sub != "grant" && sub != "revoke" {
		fmt.Fprint(os.Stderr, `Usage: pwikit admin <create|grant|revoke> -name <name> [options]

  create  take over an imported account or make a new one, and give it every right
  grant   give every right to an account that already exists
  revoke  take every right back from an account, leaving the account itself alone

Import an archive before creating the first administrator. An account the
archive brought in is taken over in place, so the pages and posts it wrote stay
with the account you sign in as.

Options:
  -name            name to sign in as; spaces and other scripts are allowed
  -password-stdin  read the password from standard input instead of asking
  -yes             create a new account without asking
`)
		return errors.New("unknown admin subcommand")
	}
	fs := flag.NewFlagSet("admin "+sub, flag.ContinueOnError)
	name := fs.String("name", "", "name to sign in as")
	fromStdin := fs.Bool("password-stdin", false, "read the password from standard input instead of asking")
	yes := fs.Bool("yes", false, "create a new account without asking")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *database == "" {
		return errors.New("no database, pass -database or set " + envDatabase)
	}

	raw, canonical, err := adminName(*name)
	if err != nil {
		return err
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, *database)
	if err != nil {
		return err
	}
	defer conn.Close()

	switch sub {
	case "grant":
		return grantAdmin(ctx, conn, raw, canonical)
	case "revoke":
		return revokeAdmin(ctx, conn, raw, canonical)
	}
	return createAdmin(ctx, conn, raw, canonical, *fromStdin, *yes)
}

func adminName(given string) (raw, canonical string, err error) {
	raw = wikidot.NormalizeDisplayName(given)
	if wikidot.ValidateDisplayName(raw) != wikidot.DisplayNameOK {
		return "", "", errors.New("-name is empty, too long, or starts with a mark")
	}
	canonical = wikidot.CanonicalizeUsername(raw)
	if canonical == "" {
		return "", "", fmt.Errorf("name %q holds no letters or digits", raw)
	}
	if wikidot.ReservedUsername(canonical) {
		return "", "", fmt.Errorf("name %q is reserved", canonical)
	}
	return raw, canonical, nil
}

func createAdmin(ctx context.Context, conn *db.DB, raw, canonical string, fromStdin, yes bool) error {
	found, hash, err := conn.UserToClaim(ctx, canonical, strings.ToLower(raw))
	switch {
	case err != nil && !errors.Is(err, db.ErrNotFound):
		return err
	case err == nil && password.IsUsable(hash):
		return fmt.Errorf("%s (#%d) already has a password; run `pwikit admin grant -name %q` instead",
			found.Username, found.ID, raw)
	}

	// One reader for the whole command. A second one over os.Stdin would find
	// the first had already buffered past the line it read.
	in := bufio.NewReader(os.Stdin)
	if found == nil {
		if err := confirmNewAccount(in, canonical, yes); err != nil {
			return err
		}
	}

	plain, err := readPassword(in, fromStdin)
	if err != nil {
		return err
	}
	if err := password.Validate(plain, password.Attributes{Username: canonical, DisplayName: raw}); err != nil {
		return err
	}
	encoded, err := password.Hash(plain)
	if err != nil {
		return err
	}

	var display *string
	if raw != canonical {
		display = &raw
	}
	if found != nil {
		if err := conn.ActivateUser(ctx, found.ID, canonical, display, encoded); err != nil {
			return err
		}
		if err := conn.SetSuperuser(ctx, found.ID, true); err != nil {
			return err
		}
		fmt.Printf("took over %s (#%d), imported as %q\n", canonical, found.ID, found.WikidotUsername)
		return nil
	}

	id, err := conn.CreateUser(ctx, canonical, raw, encoded, true, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := conn.SetSuperuser(ctx, id, true); err != nil {
		return err
	}
	fmt.Printf("created %s (#%d)\n", canonical, id)
	return nil
}

func confirmNewAccount(in *bufio.Reader, canonical string, yes bool) error {
	if yes {
		return nil
	}
	fmt.Fprintf(os.Stderr, `No account is named %q.

A new one will be created, holding nothing. If you meant to sign in as an
account an archive brought in, stop here: import the archive first, and spell
the name the way that site spelled it.

Create a new account? [y/N] `, canonical)
	answer, err := in.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		return errors.New("cancelled")
	}
	return nil
}

func grantAdmin(ctx context.Context, conn *db.DB, raw, canonical string) error {
	found, _, err := conn.UserToClaim(ctx, canonical, strings.ToLower(raw))
	if errors.Is(err, db.ErrNotFound) {
		return fmt.Errorf("no account is named %q", canonical)
	}
	if err != nil {
		return err
	}
	if found.IsSuperuser {
		fmt.Printf("%s (#%d) already has every right\n", found.Username, found.ID)
		return nil
	}
	if err := conn.SetSuperuser(ctx, found.ID, true); err != nil {
		return err
	}
	fmt.Printf("granted every right to %s (#%d)\n", found.Username, found.ID)
	return nil
}

func revokeAdmin(ctx context.Context, conn *db.DB, raw, canonical string) error {
	found, _, err := conn.UserToClaim(ctx, canonical, strings.ToLower(raw))
	if errors.Is(err, db.ErrNotFound) {
		return fmt.Errorf("no account is named %q", canonical)
	}
	if err != nil {
		return err
	}
	if !found.IsSuperuser {
		fmt.Printf("%s (#%d) does not have every right\n", found.Username, found.ID)
		return nil
	}
	left, err := conn.SuperuserCount(ctx)
	if err != nil {
		return err
	}
	if err := conn.SetSuperuser(ctx, found.ID, false); err != nil {
		return err
	}
	fmt.Printf("took every right back from %s (#%d)\n", found.Username, found.ID)
	if left <= 1 {
		fmt.Fprintln(os.Stderr, "that was the last one; nobody can reach the admin pages until `pwikit admin grant` runs")
	}
	return nil
}

// A password typed at a prompt would otherwise stay on screen and in the
// scrollback of whoever runs this.
func readPassword(in *bufio.Reader, fromStdin bool) (string, error) {
	if fromStdin || !term.IsTerminal(int(os.Stdin.Fd())) {
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	fmt.Fprint(os.Stderr, "Password: ")
	typed, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(typed), nil
}
