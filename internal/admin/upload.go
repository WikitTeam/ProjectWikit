package admin

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

const uploadRoot = "-"

const maxIconBytes = 2 << 20

var (
	errUploadType = errors.New("admin: the file is not of an accepted type")
	errUploadSize = errors.New("admin: the file is too large")
)

type uploadRule struct {
	dir   string
	exts  []string
	check func(head []byte) bool
}

var siteIcons = uploadRule{
	dir:  "sites",
	exts: []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico"},
	check: func(head []byte) bool {
		return strings.HasPrefix(http.DetectContentType(head), "image/")
	},
}

var roleIcon = uploadRule{
	dir:   "roles",
	exts:  []string{".svg"},
	check: func(head []byte) bool { return strings.Contains(string(head), "<svg") },
}

// An empty name answers a field that carried no file, which is how a save that
// does not touch the field keeps whatever is already stored.
func (h *Handler) store(r *http.Request, field string, rule uploadRule) (string, error) {
	if r.MultipartForm == nil {
		return "", nil
	}
	files := r.MultipartForm.File[field]
	if len(files) == 0 || files[0].Size == 0 {
		return "", nil
	}
	header := files[0]
	if header.Size > maxIconBytes {
		return "", errUploadSize
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !contains(rule.exts, ext) {
		return "", errUploadType
	}

	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	head := make([]byte, 1024)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	if !rule.check(head[:n]) {
		return "", errUploadType
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	name, full, err := h.freeName(rule.dir, cleanName(header.Filename, ext))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	dst, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, io.LimitReader(src, maxIconBytes)); err != nil {
		return "", err
	}
	return path.Join(uploadRoot, rule.dir, name), nil
}

func (h *Handler) freeName(dir, name string) (string, string, error) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	for i := 0; i < 8; i++ {
		full, err := paths.Resolve(h.deps.Files, filepath.Join(uploadRoot, dir, name))
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(full); errors.Is(err, os.ErrNotExist) {
			return name, full, nil
		}
		var suffix [3]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", "", err
		}
		name = base + "-" + hex.EncodeToString(suffix[:]) + ext
	}
	return "", "", errors.New("admin: no free name for the upload")
}

func cleanName(name, ext string) string {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		default:
			b.WriteByte('-')
		}
	}
	trimmed := strings.Trim(b.String(), "-")
	if trimmed == "" {
		trimmed = "icon"
	}
	if len(trimmed) > 60 {
		trimmed = trimmed[:60]
	}
	return trimmed + ext
}

func (h *Handler) pickIcon(r *http.Request, field, stored string, rule uploadRule) (string, error) {
	if r.PostFormValue("clear_"+field) != "" {
		return "", nil
	}
	name, err := h.store(r, field, rule)
	if err != nil {
		return "", err
	}
	if name == "" {
		return stored, nil
	}
	return name, nil
}

func iconMessage(loc *i18n.Localizer, err error) string {
	switch {
	case errors.Is(err, errUploadType):
		return loc.T("admin.icon-bad-type")
	case errors.Is(err, errUploadSize):
		return loc.T("admin.icon-too-large")
	}
	return ""
}

func (h *Handler) iconProblem(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, err error) error {
	if message := iconMessage(loc, err); message != "" {
		return h.siteForm(w, r, loc, message, "")
	}
	return err
}
