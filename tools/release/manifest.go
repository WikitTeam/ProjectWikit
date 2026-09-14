package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/pgbundle"
	"github.com/WikitTeam/ProjectWikit/internal/update"
)

func manifestCommand(args []string) error {
	fs := flag.NewFlagSet("manifest", flag.ContinueOnError)
	dir := fs.String("dir", "dist", "directory holding the release packages")
	version := fs.String("version", "", "release version the packages were built as")
	repository := fs.String("repository", "WikitTeam/ProjectWikit", "GitHub repository the release notes live in")
	published := fs.String("published", "", "publication time in RFC 3339; defaults to now")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if !versionPattern.MatchString(*version) {
		return fmt.Errorf("-version %q is not a version such as v1.0.0", *version)
	}
	at := time.Now().UTC()
	if *published != "" {
		parsed, err := time.Parse(time.RFC3339, *published)
		if err != nil {
			return fmt.Errorf("-published: %w", err)
		}
		at = parsed.UTC()
	}

	entries, err := os.ReadDir(*dir)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta("pwikit-"+*version+"-") + `([a-z0-9]+-[a-z0-9]+)\.(tar\.gz|zip)$`)

	m := update.Manifest{
		Format:      update.ManifestFormat,
		Version:     *version,
		PublishedAt: at.Format(time.RFC3339),
		Notes:       "https://github.com/" + *repository + "/releases/tag/" + *version,
		Postgres:    pgbundle.Version,
		Packages:    map[string]update.Package{},
	}
	var sums []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == update.ChecksumsName || name == update.ManifestName {
			continue
		}
		sum, size, err := hashFile(filepath.Join(*dir, name))
		if err != nil {
			return err
		}
		sums = append(sums, sum+"  "+name)
		if match := pattern.FindStringSubmatch(name); match != nil {
			m.Packages[match[1]] = update.Package{File: name, SHA256: sum, Size: size}
		}
	}
	if len(m.Packages) == 0 {
		return fmt.Errorf("no package for %s in %s", *version, *dir)
	}
	sort.Slice(sums, func(i, j int) bool { return sums[i][66:] < sums[j][66:] })

	if err := os.WriteFile(filepath.Join(*dir, update.ChecksumsName), []byte(strings.Join(sums, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*dir, update.ManifestName), append(body, '\n'), 0o644); err != nil {
		return err
	}
	platforms := make([]string, 0, len(m.Packages))
	for p := range m.Packages {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)
	fmt.Printf("wrote %s and %s for %s: %s\n", update.ChecksumsName, update.ManifestName, *version, strings.Join(platforms, ", "))
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
