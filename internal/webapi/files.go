package webapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/imagemeta"
	"github.com/WikitTeam/ProjectWikit/internal/media"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/quota"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const FilesPrefix = "/pw-api/files/"

const uploadChunk = 102400

var errFileNotFound = errors.New("webapi: no such file")

// A file's own path carries no page name, so it reaches the same handlers
// through an entry point of its own.
type FileItems struct {
	articles *Articles
}

var _ http.Handler = (*FileItems)(nil)

func NewFileItems(d Deps, next http.Handler) *FileItems {
	return &FileItems{articles: NewArticles(d, next)}
}

func (h *FileItems) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a := h.articles
	loc := a.deps.Bundle.For(r.Context())

	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, FilesPrefix), 10, 64)
	if err != nil {
		a.next.ServeHTTP(w, r)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		a.answerFile(w, r, loc, id, a.removeFile)
	case http.MethodPut:
		a.answerFile(w, r, loc, id, a.renameFile)
	default:
		a.next.ServeHTTP(w, r)
	}
}

type fileFunc func(r *http.Request, loc *i18n.Localizer, file *db.FileRow, article *db.Article) (string, int, error)

func (h *Articles) answerFile(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	id int64, fn fileFunc) {

	file, article, err := h.fileAndArticle(r, id)
	if err != nil {
		h.failFile(w, loc, err)
		return
	}
	if err := h.viewable(r, article.FullName()); err != nil {
		h.failFile(w, loc, err)
		return
	}
	if err := h.mayManageFiles(r, article); err != nil {
		h.failFile(w, loc, err)
		return
	}
	body, status, err := fn(r, loc, file, article)
	if err != nil {
		h.failFile(w, loc, err)
		return
	}
	writeJSON(w, status, body)
}

func (h *Articles) failFile(w http.ResponseWriter, loc *i18n.Localizer, err error) {
	if errors.Is(err, errFileNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-file-not-found")))
		return
	}
	h.fail(w, loc, err)
}

func (h *Articles) fileAndArticle(r *http.Request, id int64) (*db.FileRow, *db.Article, error) {
	ctx := r.Context()
	file, err := h.deps.DB.FileByID(ctx, id)
	if errors.Is(err, db.ErrNotFound) || (err == nil && file.Deleted) {
		return nil, nil, errFileNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	article, err := h.deps.DB.ArticleByID(ctx, siteID(ctx), file.ArticleID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil, errNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return file, article, nil
}

func (h *Articles) mayManageFiles(r *http.Request, article *db.Article) error {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	perm := repo.NewPerms(ctx, h.deps.DB)

	subject, err := perm.Subject(user, time.Now())
	if err != nil {
		return err
	}
	object, err := perm.Article(article, user)
	if err != nil {
		return err
	}
	if !perms.Resolve(subject, object).Has(perms.ManageArticleFiles) {
		return errForbidden
	}
	return nil
}

func (h *Articles) files(r *http.Request, _ *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	found, err := h.deps.DB.ArticleFileList(ctx, article.ID)
	if err != nil {
		return "", 0, err
	}
	rendered := make(wikijson.Array, 0, len(found))
	for i := range found {
		one, err := h.fileJSON(r, found[i])
		if err != nil {
			return "", 0, err
		}
		rendered = append(rendered, one)
	}

	live, total, err := h.deps.DB.FileSpaceUsage(ctx)
	if err != nil {
		return "", 0, err
	}
	limits, err := h.limits(ctx, site.FromContext(ctx))
	if err != nil {
		return "", 0, err
	}
	used, err := h.deps.DB.SiteFileUsage(ctx, siteID(ctx))
	if err != nil {
		return "", 0, err
	}
	policy := h.policy()
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "pageId", Value: article.FullName()},
		{Key: "files", Value: rendered},
		{Key: "softLimit", Value: policy.InstanceLive},
		{Key: "hardLimit", Value: policy.InstanceTotal},
		{Key: "softUsed", Value: live},
		{Key: "hardUsed", Value: total},
		{Key: "maxFile", Value: limits.MaxFile},
		{Key: "siteQuota", Value: limits.SiteQuota},
		{Key: "siteUsed", Value: used},
	})
	return body, http.StatusOK, err
}

func (h *Articles) policy() *quota.Policy {
	if h.deps.Uploads == nil {
		return &quota.Policy{}
	}
	return h.deps.Uploads
}

func (h *Articles) limits(ctx context.Context, current *db.Site) (quota.Limits, error) {
	if current == nil {
		return h.policy().For("", quota.Rule{}), nil
	}
	maxFile, siteQuota, err := h.deps.DB.SiteUploadLimit(ctx, current.ID)
	if err != nil {
		return quota.Limits{}, err
	}
	return h.policy().For(current.Slug, quota.Rule{MaxFile: maxFile, SiteQuota: siteQuota}), nil
}

func (h *Articles) backend(ctx context.Context) *storage.Backend {
	slug := ""
	if current := site.FromContext(ctx); current != nil {
		slug = current.Slug
	}
	return h.deps.Storage.For(slug)
}

func (h *Articles) fileJSON(r *http.Request, file db.FileRecord) (wikijson.Object, error) {
	ctx := r.Context()
	var author *db.User
	if file.AuthorID != nil {
		found, err := h.deps.DB.UserByID(ctx, *file.AuthorID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		author = found
	}
	rendered, err := repo.UserJSON(ctx, h.deps.DB, author)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "id", Value: file.ID},
		{Key: "name", Value: file.Name},
		{Key: "size", Value: file.Size},
		{Key: "createdAt", Value: isoTime(file.CreatedAt)},
		{Key: "author", Value: rendered},
		{Key: "mimeType", Value: file.MimeType},
	}, nil
}

func (h *Articles) upload(r *http.Request, loc *i18n.Localizer, name string) (string, int, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		body, status := csrfRefusal(r, loc)
		return body, status, nil
	}

	article, err := h.article(r, name)
	if err != nil {
		return "", 0, err
	}
	if err := h.mayManageFiles(r, article); err != nil {
		return "", 0, err
	}

	fileName, ok := uploadName(r)
	if !ok {
		return field("error", loc.T("api-missing-file-name")), http.StatusBadRequest, nil
	}
	if _, err := h.deps.DB.LiveFileNamed(ctx, article.ID, fileName); err == nil {
		return field("error", loc.T("api-file-exists")), http.StatusConflict, nil
	} else if !errors.Is(err, db.ErrNotFound) {
		return "", 0, err
	}

	unique, err := randomName()
	if err != nil {
		return "", 0, err
	}
	mediaName := unique + filepath.Ext(fileName)
	stored := storage.Join("media", media.QuoteName(article.MediaName), media.QuoteName(mediaName))
	limits, err := h.limits(ctx, current)
	if err != nil {
		return "", 0, err
	}
	temp, size, refusal, err := h.receive(r, loc, limits)
	if err != nil {
		return "", 0, err
	}
	if refusal != "" {
		return field("error", refusal), http.StatusRequestEntityTooLarge, nil
	}
	defer os.Remove(temp)
	if size, err = scrub(temp, size); err != nil {
		return "", 0, err
	}
	store := h.backend(ctx).Store
	if err := putFile(ctx, store, stored, temp, size); err != nil {
		return "", 0, err
	}

	var userID *int64
	if user := auth.FromContext(ctx); user != nil {
		userID = &user.ID
	}
	at := time.Now().UTC()
	id, err := h.deps.DB.AddArticleFile(ctx, db.FileWrite{
		ArticleID: article.ID,
		Name:      fileName,
		MediaName: mediaName,
		MimeType:  uploadMime(r),
		Size:      size,
		AuthorID:  userID,
		At:        at,
	})
	if err != nil {
		_ = store.Remove(ctx, stored)
		return "", 0, err
	}
	meta, err := json.Marshal(map[string]any{"name": fileName, "id": id})
	if err != nil {
		return "", 0, err
	}
	rev, err := h.deps.DB.AddArticleLogEntry(ctx, article.ID, userID,
		db.LogFileAdded, "", string(meta), at)
	if err != nil {
		return "", 0, err
	}
	if err := h.notifyRevision(r, article, rev); err != nil {
		return "", 0, err
	}
	return field("status", "ok"), http.StatusOK, nil
}

// Reading the whole body first would hold an upload of any size in memory, so
// the ceilings are checked as the bytes land, in a temporary file that only
// reaches the store once the upload is whole.
func (h *Articles) receive(r *http.Request, loc *i18n.Localizer, limits quota.Limits) (temp string, size int64, refusal string, err error) {
	ctx := r.Context()
	live, total, err := h.deps.DB.FileSpaceUsage(ctx)
	if err != nil {
		return "", 0, "", err
	}
	used, err := h.deps.DB.SiteFileUsage(ctx, siteID(ctx))
	if err != nil {
		return "", 0, "", err
	}
	policy := h.policy()
	refuse := func(size int64) string {
		switch {
		case overLimit(limits.MaxFile, size):
			return loc.T("api-upload-file-too-large", "limit", quota.Format(limits.MaxFile))
		case overLimit(limits.SiteQuota, used+size):
			return loc.T("api-upload-site-full", "limit", quota.Format(limits.SiteQuota))
		case overLimit(policy.InstanceLive, live+size) || overLimit(policy.InstanceTotal, total+size):
			return loc.T("api-upload-too-large")
		}
		return ""
	}
	if r.ContentLength > 0 {
		if refusal := refuse(r.ContentLength); refusal != "" {
			drain(r)
			return "", 0, refusal, nil
		}
	}

	out, err := os.CreateTemp("", "pwikit-upload-*")
	if err != nil {
		return "", 0, "", err
	}
	fail := func(refusal string, err error) (string, int64, string, error) {
		out.Close()
		os.Remove(out.Name())
		if refusal != "" {
			drain(r)
		}
		return "", 0, refusal, err
	}
	buf := make([]byte, uploadChunk)
	for {
		n, readErr := r.Body.Read(buf)
		size += int64(n)
		if refusal := refuse(size); refusal != "" {
			return fail(refusal, nil)
		}
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return fail("", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fail("", readErr)
		}
	}
	if err := out.Close(); err != nil {
		os.Remove(out.Name())
		return "", 0, "", err
	}
	return out.Name(), size, "", nil
}

const drainLimit = 512 << 20

// A browser still sending the body drops an early answer and reports a broken
// connection, so the rest is read and thrown away, up to a point.
func drain(r *http.Request) {
	_, _ = io.CopyN(io.Discard, r.Body, drainLimit)
}

func putFile(ctx context.Context, store storage.Store, key, from string, size int64) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}
	defer f.Close()
	return store.Put(ctx, key, f, size)
}

const scrubLimit = 64 << 20

// Photos from phones carry the place they were taken, which a wiki should not
// hand to every reader. Larger files are left as they came.
func scrub(name string, size int64) (int64, error) {
	if size > scrubLimit {
		return size, nil
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return 0, err
	}
	clean, changed := imagemeta.Strip(data)
	if !changed {
		return size, nil
	}
	if err := os.WriteFile(name, clean, 0o600); err != nil {
		return 0, err
	}
	return int64(len(clean)), nil
}

func randomName() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func overLimit(limit, used int64) bool {
	return limit > 0 && used > limit
}

func uploadName(r *http.Request) (string, bool) {
	raw := r.Header.Get("x-file-name")
	if raw == "" {
		return "", false
	}
	// Percent escapes only. A '+' is a character a file name may legitimately
	// carry, not a stand-in for a space.
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw, true
	}
	return decoded, true
}

func uploadMime(r *http.Request) string {
	if mime := r.Header.Get("Content-Type"); mime != "" {
		return mime
	}
	return "application/octet-stream"
}

func (h *Articles) removeFile(r *http.Request, loc *i18n.Localizer,
	file *db.FileRow, article *db.Article) (string, int, error) {

	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		body, status := csrfRefusal(r, loc)
		return body, status, nil
	}

	var userID *int64
	if user := auth.FromContext(ctx); user != nil {
		userID = &user.ID
	}
	at := time.Now().UTC()
	if _, _, err := h.deleteFile(ctx, file.ID, at, userID); err != nil {
		return "", 0, err
	}
	meta, err := json.Marshal(map[string]any{"name": file.Name, "id": file.ID})
	if err != nil {
		return "", 0, err
	}
	rev, err := h.deps.DB.AddArticleLogEntry(ctx, article.ID, userID,
		db.LogFileDeleted, "", string(meta), at)
	if err != nil {
		return "", 0, err
	}
	if err := h.notifyRevision(r, article, rev); err != nil {
		return "", 0, err
	}
	return field("status", "ok"), http.StatusOK, nil
}

func (h *Articles) renameFile(r *http.Request, loc *i18n.Localizer,
	file *db.FileRow, article *db.Article) (string, int, error) {

	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		body, status := csrfRefusal(r, loc)
		return body, status, nil
	}

	raw, err := readBody(r)
	if err != nil {
		return field("error", loc.T("api-bad-request")), http.StatusBadRequest, nil
	}
	var body fields
	if json.Unmarshal(raw, &body) != nil || !body.has("name") {
		return field("error", loc.T("api-bad-request")), http.StatusBadRequest, nil
	}
	wanted, _ := body.text("name")
	if wanted == "" {
		return field("error", loc.T("api-missing-file-name")), http.StatusBadRequest, nil
	}

	taken, err := h.deps.DB.LiveFileNamed(ctx, article.ID, wanted)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", 0, err
	}
	if err == nil && taken != file.ID {
		return field("error", loc.T("api-file-exists")), http.StatusConflict, nil
	}

	previous, _, err := h.deps.DB.RenameFile(ctx, file.ID, wanted)
	if err != nil {
		return "", 0, err
	}
	if previous == wanted {
		return field("status", "ok"), http.StatusOK, nil
	}

	var userID *int64
	if user := auth.FromContext(ctx); user != nil {
		userID = &user.ID
	}
	meta, err := json.Marshal(map[string]any{"name": wanted, "prev_name": previous, "id": file.ID})
	if err != nil {
		return "", 0, err
	}
	rev, err := h.deps.DB.AddArticleLogEntry(ctx, article.ID, userID,
		db.LogFileRenamed, "", string(meta), time.Now().UTC())
	if err != nil {
		return "", 0, err
	}
	if err := h.notifyRevision(r, article, rev); err != nil {
		return "", 0, err
	}
	return field("status", "ok"), http.StatusOK, nil
}
