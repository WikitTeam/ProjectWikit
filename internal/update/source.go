package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	GitHubReleases = "https://github.com/WikitTeam/ProjectWikit/releases"

	EnvReleases = "PWIKIT_RELEASES_URL"

	maxManifest = 1 << 20
)

type Source struct {
	Releases string
	Mirror   string
	Log      func(string)
	client   *http.Client
}

func NewSource(mirror string) Source {
	releases := GitHubReleases
	if env := os.Getenv(EnvReleases); env != "" {
		releases = env
	}
	return Source{
		Releases: strings.TrimSuffix(releases, "/"),
		Mirror:   strings.TrimSuffix(mirror, "/"),
		client:   &http.Client{Timeout: 30 * time.Minute},
	}
}

func (s Source) Latest(ctx context.Context) (Manifest, error) {
	body, err := s.small(ctx, "latest/download/"+ManifestName)
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifest(body)
}

func (s Source) Checksums(ctx context.Context, version string) (map[string]string, error) {
	body, err := s.small(ctx, "download/"+version+"/"+ChecksumsName)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 {
			out[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
		}
	}
	return out, scanner.Err()
}

func (s Source) Download(ctx context.Context, version, file, dest, want string) error {
	return s.each(ctx, "download/"+version+"/"+file, func(url string) error {
		tmp := dest + ".part"
		out, err := os.Create(tmp)
		if err != nil {
			return err
		}
		h := sha256.New()
		err = s.fetch(ctx, url, io.MultiWriter(out, h), 0)
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
				err = fmt.Errorf("%s has sha256 %s, want %s; the download is damaged or was altered", file, got, want)
			}
		}
		if err != nil {
			os.Remove(tmp)
			return err
		}
		return os.Rename(tmp, dest)
	})
}

func (s Source) small(ctx context.Context, path string) ([]byte, error) {
	var body []byte
	err := s.each(ctx, path, func(url string) error {
		var buf bytes.Buffer
		if err := s.fetch(ctx, url, &buf, maxManifest); err != nil {
			return err
		}
		body = buf.Bytes()
		return nil
	})
	return body, err
}

func (s Source) each(ctx context.Context, path string, attempt func(url string) error) error {
	err := attempt(s.Releases + "/" + path)
	if err == nil || s.Mirror == "" || ctx.Err() != nil {
		return err
	}
	s.note(fmt.Sprintf("GitHub could not be reached (%v), trying %s", err, s.Mirror))
	if mirrorErr := attempt(s.Mirror + "/" + path); mirrorErr != nil {
		return fmt.Errorf("%w; the mirror failed too: %v", err, mirrorErr)
	}
	return nil
}

func (s Source) fetch(ctx context.Context, url string, into io.Writer, limit int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pwikit-updater")
	client := s.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	body := io.Reader(resp.Body)
	if limit > 0 {
		body = io.LimitReader(resp.Body, limit)
	}
	if _, err := io.Copy(into, body); err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	return nil
}

func (s Source) note(line string) {
	if s.Log != nil {
		s.Log(line)
	}
}
