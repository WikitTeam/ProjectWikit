package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/migrate"
)

type Report struct {
	Manifest Manifest
	Problems []string
	Notes    []string
}

func (r Report) OK() bool { return len(r.Problems) == 0 }

func (r *Report) problem(format string, args ...any) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
}

func (r *Report) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

// Verify reads the whole archive rather than trusting the manifest, and keeps
// going after the first complaint so one run tells you everything.
func Verify(name string) (Report, error) {
	var r Report

	m, err := ReadManifestOf(name)
	if err != nil {
		return r, err
	}
	r.Manifest = m

	file, err := os.Open(name)
	if err != nil {
		return r, err
	}
	defer file.Close()

	zip, err := gzip.NewReader(file)
	if err != nil {
		return r, fmt.Errorf("%s is not gzip: %w", name, err)
	}
	defer zip.Close()

	seenTables := map[string]bool{}
	seenFiles := map[string]bool{}
	tr := tar.NewReader(zip)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return r, fmt.Errorf("read %s: %w", name, err)
		}
		switch {
		case head.Name == ManifestName:
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return r, err
			}
		case strings.HasPrefix(head.Name, dataDir+"/"):
			table := strings.TrimSuffix(path.Base(head.Name), dataSuffix)
			seenTables[table] = true
			checkTable(&r, tr, table, m.Tables[table])
		case strings.HasPrefix(head.Name, filesDir+"/"):
			rel := strings.TrimPrefix(head.Name, filesDir+"/")
			seenFiles[rel] = true
			checkFile(&r, tr, rel, m.Files.Entries[rel])
		default:
			r.problem("the archive holds %q, which does not belong to this format", head.Name)
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return r, err
			}
		}
	}

	for table := range m.Tables {
		if !seenTables[table] {
			r.problem("the manifest names table %s but the archive has no data for it", table)
		}
	}
	if m.Files.Included {
		for rel := range m.Files.Entries {
			if !seenFiles[rel] {
				r.problem("the manifest names file %s but the archive does not hold it", rel)
			}
		}
	}
	checkMigrations(&r, m)
	sort.Strings(r.Problems)
	return r, nil
}

func checkTable(r *Report, body io.Reader, table string, want Table) {
	if want.SHA256 == "" {
		r.problem("the archive holds data for %s, which the manifest does not name", table)
		io.Copy(io.Discard, body)
		return
	}
	sum := sha256.New()
	rows, size, err := countRows(io.TeeReader(body, sum))
	if err != nil {
		r.problem("read the data of %s: %v", table, err)
		return
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != want.SHA256 {
		r.problem("%s is damaged; its checksum is %s and the manifest says %s", table, short(got), short(want.SHA256))
	}
	if size != want.Bytes {
		r.problem("%s holds %d bytes and the manifest says %d", table, size, want.Bytes)
	}
	if rows != want.Rows {
		r.problem("%s holds %d rows and the manifest says %d", table, rows, want.Rows)
	}
}

func checkFile(r *Report, body io.Reader, rel, want string) {
	sum := sha256.New()
	if _, err := io.Copy(sum, body); err != nil {
		r.problem("read the file %s: %v", rel, err)
		return
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if want == "" {
		r.problem("the archive holds the file %s, which the manifest does not name", rel)
		return
	}
	if got != want {
		r.problem("the file %s is damaged; its checksum is %s and the manifest says %s", rel, short(got), short(want))
	}
}

func checkMigrations(r *Report, m Manifest) {
	carried := map[string]bool{}
	for _, name := range migrate.Names() {
		carried[name] = true
	}
	var unknown []string
	for _, name := range m.Migrations {
		if !carried[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		r.problem("the backup was made by a newer pwikit; it carries %s, which this build does not know. Restore it with the pwikit that made it",
			strings.Join(unknown, ", "))
	}
	if m.PGVersion != 0 && m.PGVersion < MinimumPGVersion {
		r.note("the backup came from postgres %s, which is older than this build was tested against", Describe(m.PGVersion))
	}
	if !m.Files.Included {
		r.note("this backup holds no uploaded files")
	}
}

// A COPY line always ends in a newline and every newline inside a value is
// escaped, so the lines are the rows.
func countRows(r io.Reader) (rows, size int64, err error) {
	buf := make([]byte, 64*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			size += int64(n)
			rows += int64(bytes.Count(buf[:n], []byte{'\n'}))
		}
		if err == io.EOF {
			return rows, size, nil
		}
		if err != nil {
			return rows, size, err
		}
	}
}

func ReadManifestOf(name string) (Manifest, error) {
	file, err := os.Open(name)
	if err != nil {
		return Manifest{}, err
	}
	defer file.Close()

	zip, err := gzip.NewReader(file)
	if err != nil {
		return Manifest{}, fmt.Errorf("%s is not gzip: %w", name, err)
	}
	defer zip.Close()

	tr := tar.NewReader(zip)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			return Manifest{}, fmt.Errorf("%s holds no %s", name, ManifestName)
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("read %s: %w", name, err)
		}
		if head.Name == ManifestName {
			return readManifest(tr)
		}
	}
}

func short(sum string) string {
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}
