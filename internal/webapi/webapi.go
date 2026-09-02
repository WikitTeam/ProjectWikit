// Package webapi answers the JSON endpoints a page calls back to.
package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	ModulesPath = "/pw-api/modules"
	PreviewPath = "/pw-api/preview"
)

const (
	renderMethod = "render"
	jsonMime     = "application/json"
)

const maxParsed = 1 << 20

type Deps struct {
	DB     *db.DB
	Engine renderer.Renderer
	Bundle *i18n.Bundle
	Icons  roles.IconLoader
	Log    *slog.Logger
}

type Handler struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Handler)(nil)

func New(d Deps, upstream http.Handler) *Handler {
	return &Handler{deps: d, upstream: upstream}
}

func (d Deps) log() *slog.Logger {
	if d.Log == nil {
		return slog.Default()
	}
	return d.Log
}

type call struct {
	Module     string          `json:"module"`
	Method     string          `json:"method"`
	PageID     string          `json:"pageId"`
	Content    string          `json:"content"`
	Params     json.RawMessage `json:"params"`
	PathParams json.RawMessage `json:"pathParams"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.upstream.ServeHTTP(w, r)
		return
	}

	raw, rest, err := peek(r.Body)
	if err != nil {
		h.forward(w, r, raw, rest)
		return
	}

	var parsed call
	if json.Unmarshal(raw, &parsed) != nil || !h.answers(parsed) {
		h.forward(w, r, raw, rest)
		return
	}

	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	out, status, err := h.answer(r, loc, parsed)
	if err != nil {
		var moduleErr *callbacks.ModuleError
		if errors.As(err, &moduleErr) {
			writeJSON(w, http.StatusInternalServerError, field("error", moduleErr.Message))
			return
		}
		h.deps.log().Error("render module api", "module", parsed.Module, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, status, out)
}

// A method beyond render is answered only when the module registered it, and a
// module registers only what changes nothing. Everything else goes upstream,
// which is what keeps the CSRF check over there.
func (h *Handler) answers(parsed call) bool {
	if parsed.Method == renderMethod {
		return true
	}
	_, ok := module.LookupAPI(parsed.Module, parsed.Method)
	return ok
}

func (h *Handler) answer(r *http.Request, loc *i18n.Localizer, parsed call) (string, int, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		return "", 0, errors.New("webapi: the request carries no site")
	}
	user := auth.FromContext(ctx)

	var article *db.Article
	if parsed.PageID != "" {
		found, err := h.deps.DB.ArticleByName(ctx, parsed.PageID)
		if errors.Is(err, db.ErrNotFound) {
			return field("error", loc.T("api-page-not-found")), http.StatusNotFound, nil
		}
		if err != nil {
			return "", 0, err
		}
		article = found
	}

	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(ctx, loc, current, user)
	pc := page.NewContext(article, article, page.ParsePathParams(string(parsed.PathParams)), user)
	// Only a token the visitor already carries, since this response has no page
	// to plant the cookie a fresh one would need.
	if token, isNew := csrf.Token(r); !isNew {
		pc.CSRF = token
	}

	if parsed.Method != renderMethod {
		return h.callAPI(env, pc, parsed)
	}

	html, err := env.Callbacks(env.Vars(article), pc).RenderModule(parsed.Module, params(parsed.Params), parsed.Content)
	if err != nil {
		return "", 0, err
	}
	return field("result", html), http.StatusOK, nil
}

func (h *Handler) callAPI(env *pagerender.Env, pc *page.Context, parsed call) (string, int, error) {

	out, err := env.ModuleAPI(pc, parsed.Module, parsed.Method, params(parsed.Params))
	if err != nil {
		return "", 0, err
	}
	body, err := wikijson.Marshal(out)
	if err != nil {
		return "", 0, err
	}
	return body, http.StatusOK, nil
}

func peek(body io.ReadCloser) (raw []byte, rest io.Reader, err error) {
	raw, err = io.ReadAll(io.LimitReader(body, maxParsed))
	if err != nil {
		return raw, nil, err
	}
	if len(raw) < maxParsed {
		return raw, nil, nil
	}
	return raw, body, errors.New("webapi: body too large to read as a call")
}

func (h *Handler) forward(w http.ResponseWriter, r *http.Request, raw []byte, rest io.Reader) {
	var body io.Reader = bytes.NewReader(raw)
	if rest != nil {
		body = io.MultiReader(body, rest)
	}
	r.Body = io.NopCloser(body)
	h.upstream.ServeHTTP(w, r)
}

func params(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if dec.Decode(&decoded) != nil {
		return nil
	}
	out := make(map[string]string, len(decoded))
	for key, value := range decoded {
		text, ok := scalar(value)
		if !ok {
			continue
		}
		out[strings.ToLower(key)] = text
	}
	return out
}

func scalar(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	case bool:
		return strconv.FormatBool(v), true
	}
	return "", false
}

func field(key, value string) string {
	return "{" + wikijson.String(key) + ": " + wikijson.String(value) + "}"
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", jsonMime)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}
