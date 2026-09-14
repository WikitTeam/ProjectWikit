package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/backup"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

func backupCommand(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "create", "list", "verify", "restore":
	default:
		fmt.Fprint(os.Stderr, `Usage: pwikit backup <create|list|verify|restore> [options]

  create   write a backup of the database and the uploaded files
  list     show the backups in the backup directory
  verify   read a backup through and report whether it is sound
  restore  put a backup back, replacing what is there now

Options:
  -output          where create writes; defaults to a timestamped name in backups/
  -no-files        leave the uploaded files out of a backup, or alone on restore
  -site            back up one site on its own instead of the whole instance
  -keep-passwords  carry the sign-in passwords into a single site backup
  -dir             directory list reads; defaults to backups/
  -force           let restore replace a database that already holds data
  -no-safety-backup  let restore skip the backup it takes of the current state
`)
		return errors.New("unknown backup subcommand")
	}

	fs := flag.NewFlagSet("backup "+sub, flag.ContinueOnError)
	output := fs.String("output", "", "where create writes the backup")
	noFiles := fs.Bool("no-files", false, "leave the uploaded files out of a backup, or alone on restore")
	site := fs.String("site", "", "back up one site on its own instead of the whole instance")
	keepPasswords := fs.Bool("keep-passwords", false, "carry the sign-in passwords into a single site backup")
	dir := fs.String("dir", "", "directory list reads")
	force := fs.Bool("force", false, "let restore replace a database that already holds data")
	noSafety := fs.Bool("no-safety-backup", false, "skip the backup restore takes of the current state")
	dataDir := fs.String("data-dir", "", "state directory; defaults to the directory holding the executable")
	database := fs.String("database", os.Getenv(envDatabase), "PostgreSQL connection string")
	loose, err := parseMixed(fs, args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	p, err := paths.New(*dataDir)
	if err != nil {
		return err
	}
	if sub == "list" {
		where := *dir
		if where == "" {
			where = p.Backups()
		}
		return listBackups(where)
	}
	if sub == "verify" {
		if len(loose) != 1 {
			return errors.New("verify needs one backup file")
		}
		return verifyBackup(loose[0])
	}
	dsn, release, err := resolveDatabase(context.Background(), *database, *dataDir)
	if err != nil {
		return err
	}
	defer release()

	files := p.Files()
	if *noFiles {
		files = ""
	}
	if sub == "create" {
		return createBackup(dsn, files, *output, p.Backups(), *site, *keepPasswords)
	}
	if len(loose) != 1 {
		return errors.New("restore needs one backup file")
	}
	return restoreBackup(loose[0], dsn, files, p.Backups(), *force, *noSafety)
}

// A file name reads naturally before the flags, and the flag package stops at
// the first thing that is not one, so the two are separated here instead.
func parseMixed(fs *flag.FlagSet, args []string) ([]string, error) {
	var loose []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return loose, nil
		}
		loose = append(loose, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func createBackup(dsn, files, output, backups, site string, keepPasswords bool) error {
	if output == "" {
		output = filepath.Join(backups, backup.DefaultName(time.Now(), site))
	}
	result, err := backup.Create(context.Background(), backup.CreateOptions{
		DSN: dsn, Files: files, Output: output, Site: site,
		KeepPasswords: keepPasswords, Report: progress,
	})
	if err != nil {
		return err
	}
	m := result.Manifest
	fmt.Printf("%s\n", result.Path)
	fmt.Printf("%d tables, %d rows, %d files, %s\n",
		len(m.Tables), m.TotalRows(), m.Files.Count, size(result.Bytes))
	return nil
}

func verifyBackup(name string) error {
	report, err := backup.Verify(name)
	if err != nil {
		return err
	}
	printReport(name, report)
	if !report.OK() {
		return fmt.Errorf("%s is not sound", filepath.Base(name))
	}
	return nil
}

func printReport(name string, report backup.Report) {
	m := report.Manifest
	fmt.Printf("%s\n", name)
	fmt.Printf("  made      %s by pwikit %s\n", m.CreatedAt.Format(time.RFC3339), m.Pwikit)
	fmt.Printf("  postgres  %s\n", backup.Describe(m.PGVersion))
	fmt.Printf("  holds     %d tables, %d rows, %d files\n", len(m.Tables), m.TotalRows(), m.Files.Count)
	for _, note := range report.Notes {
		fmt.Printf("  note      %s\n", note)
	}
	for _, problem := range report.Problems {
		fmt.Printf("  PROBLEM   %s\n", problem)
	}
	if report.OK() {
		fmt.Println("  sound")
	}
}

func restoreBackup(name, dsn, files, backups string, force, noSafety bool) error {
	holdsData, err := backup.Ready(context.Background(), dsn, force)
	if err != nil {
		return err
	}
	if !holdsData && !noSafety {
		fmt.Println("the database holds no data yet, so there is nothing to back up first")
	}
	if holdsData && !noSafety {
		safety := filepath.Join(backups, "before-restore-"+backup.DefaultName(time.Now(), ""))
		fmt.Println("backing up the current state first")
		result, err := backup.Create(context.Background(), backup.CreateOptions{
			DSN: dsn, Files: files, Output: safety, Report: progress,
		})
		if err != nil {
			return fmt.Errorf("could not back up the current state, so nothing was changed: %w", err)
		}
		fmt.Printf("the state before this restore is in %s\n", result.Path)
	}

	result, err := backup.Restore(context.Background(), name, backup.RestoreOptions{
		DSN: dsn, Files: files, Force: force, Report: progress,
	})
	if err != nil {
		return err
	}
	fmt.Printf("restored %d rows into %d tables\n", result.Rows, len(result.Manifest.Tables))
	if result.FilesPut > 0 {
		fmt.Printf("restored %d files\n", result.FilesPut)
	}
	if result.ReplacedDir != "" {
		fmt.Printf("the files that were there are in %s\n", result.ReplacedDir)
	}
	if len(result.MigratedUp) > 0 {
		fmt.Printf("brought the schema forward with %s\n", strings.Join(result.MigratedUp, ", "))
	}
	return nil
}

func listBackups(dir string) error {
	found, err := backup.List(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("no backups in %s\n", dir)
			return nil
		}
		return err
	}
	if len(found) == 0 {
		fmt.Printf("no backups in %s\n", dir)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FILE\tMADE\tPWIKIT\tPOSTGRES\tROWS\tFILES\tSIZE")
	for _, one := range found {
		if one.Problem != "" {
			fmt.Fprintf(w, "%s\tUNREADABLE\t\t\t\t\t%s\n", filepath.Base(one.Path), size(one.Bytes))
			continue
		}
		m := one.Manifest
		files := "no"
		if m.Files.Included {
			files = fmt.Sprint(m.Files.Count)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			filepath.Base(one.Path), m.CreatedAt.Format("2006-01-02 15:04"),
			m.Pwikit, backup.Describe(m.PGVersion), m.TotalRows(), files, size(one.Bytes))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	for _, one := range found {
		if one.Problem != "" {
			fmt.Fprintf(os.Stderr, "%s: %s\n", filepath.Base(one.Path), one.Problem)
		}
	}
	return nil
}

func progress(line string) {
	fmt.Fprintf(os.Stderr, "\r\033[K%s", line)
	if !strings.HasPrefix(line, "[") {
		fmt.Fprintln(os.Stderr)
	}
}

func size(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
