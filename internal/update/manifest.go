package update

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	ManifestName   = "latest.json"
	ChecksumsName  = "SHA256SUMS"
	ManifestFormat = 1
)

type Manifest struct {
	Format      int                `json:"format"`
	Version     string             `json:"version"`
	PublishedAt string             `json:"published_at"`
	Notes       string             `json:"notes"`
	Postgres    string             `json:"postgres"`
	Packages    map[string]Package `json:"packages"`
}

type Package struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func ParseManifest(body []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", ManifestName, err)
	}
	if m.Format > ManifestFormat {
		return Manifest{}, fmt.Errorf("%s is format %d and this pwikit reads up to %d; update by hand once", ManifestName, m.Format, ManifestFormat)
	}
	if _, ok := ParseVersion(m.Version); !ok {
		return Manifest{}, fmt.Errorf("%s names %q, which is not a release", ManifestName, m.Version)
	}
	return m, nil
}

func (m Manifest) Published() time.Time {
	at, err := time.Parse(time.RFC3339, m.PublishedAt)
	if err != nil {
		return time.Time{}
	}
	return at
}

func PackageFile(version, goos, goarch string) string {
	base := "pwikit-" + version + "-" + goos + "-" + goarch
	if goos == "windows" {
		return base + ".zip"
	}
	return base + ".tar.gz"
}

func PackageDir(version, goos, goarch string) string {
	return "pwikit-" + version + "-" + goos + "-" + goarch
}

func ExecutableName(goos string) string {
	if goos == "windows" {
		return "pwikit.exe"
	}
	return "pwikit"
}
