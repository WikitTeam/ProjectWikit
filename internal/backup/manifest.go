package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const (
	// Format is the version of the layout inside the archive, not of pwikit. It
	// changes only when an older build could no longer read a newer file.
	Format = 1

	ManifestName = "manifest.json"
	dataDir      = "data"
	filesDir     = "files"
	dataSuffix   = ".copy"

	Extension = ".pwbak"

	// The digest of nothing, which is what a table a site export leaves behind
	// has to carry so verify can tell it from a table that went missing.
	emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

type Table struct {
	Rows   int64  `json:"rows"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type Files struct {
	Included bool              `json:"included"`
	Count    int               `json:"count"`
	Bytes    int64             `json:"bytes"`
	Entries  map[string]string `json:"entries"`
}

type Manifest struct {
	Format     int       `json:"format"`
	CreatedAt  time.Time `json:"created_at"`
	Pwikit     string    `json:"pwikit_version"`
	PGVersion  int       `json:"pg_version"`
	Migrations []string  `json:"migrations"`
	// Site is empty for a whole instance and the slug when one site was taken
	// out on its own.
	Site          string           `json:"site,omitempty"`
	KeptPasswords bool             `json:"kept_passwords,omitempty"`
	Tables        map[string]Table `json:"tables"`
	Files         Files            `json:"files"`
}

func (m Manifest) TotalRows() int64 {
	var n int64
	for _, t := range m.Tables {
		n += t.Rows
	}
	return n
}

func (m Manifest) write(w io.Writer) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(raw, '\n'))
	return err
}

func readManifest(r io.Reader) (Manifest, error) {
	var m Manifest
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", ManifestName, err)
	}
	if m.Format == 0 {
		return Manifest{}, fmt.Errorf("%s names no format version", ManifestName)
	}
	if m.Format > Format {
		return Manifest{}, fmt.Errorf("the backup is format %d and this build reads up to %d; run the newer pwikit", m.Format, Format)
	}
	return m, nil
}
