package archive

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bodgit/sevenzip"
)

// Sources reads the wikitext of every revision that carries one, keyed by
// revision number. A page whose archive is missing comes back empty rather than
// failing, which is what the backup does when a page was never fetched.
func (a *Archive) Sources(slug string, page Page) (map[int]string, error) {
	wanted := map[string]int{}
	for _, rev := range page.Revisions {
		if rev.HasSource() {
			wanted[strconv.Itoa(rev.Number)+".txt"] = rev.Number
		}
	}
	out := make(map[int]string, len(wanted))
	if len(wanted) == 0 {
		return out, nil
	}

	reader, err := sevenzip.OpenReader(a.sourcePath(slug, page.Stem))
	if err != nil {
		return out, nil
	}
	defer reader.Close()

	for _, entry := range reader.File {
		number, ok := wanted[strings.ToLower(entry.Name)]
		if !ok {
			continue
		}
		body, err := readEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("read revision %d of %q: %w", number, page.Name, err)
		}
		out[number] = body
	}
	return out, nil
}

func readEntry(entry *sevenzip.File) (string, error) {
	rc, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
