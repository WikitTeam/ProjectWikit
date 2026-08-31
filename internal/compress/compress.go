// Package compress gzips a response with a deterministic header, so the same
// body twice produces the same bytes.
package compress

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Below this size the response goes out untouched, Vary included, so that
// header is not a sign compression was considered.
const minSize = 200

const (
	maxPadding = 100
	headerSize = 10
	flagName   = 0b00001000
	level      = 6
)

var acceptsGzip = regexp.MustCompile(`\bgzip\b`)

func New(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := &buffer{header: make(http.Header), status: http.StatusOK}
		next.ServeHTTP(buf, r)

		body := buf.body.Bytes()
		for name, values := range buf.header {
			w.Header()[name] = values
		}

		if len(body) < minSize || buf.header.Get("Content-Encoding") != "" {
			write(w, buf.status, body)
			return
		}
		patchVary(w.Header())
		if !acceptsGzip.MatchString(r.Header.Get("Accept-Encoding")) {
			write(w, buf.status, body)
			return
		}
		packed, err := pack(body)
		if err != nil || len(packed) >= len(body) {
			write(w, buf.status, body)
			return
		}
		if tag := w.Header().Get("ETag"); strings.HasPrefix(tag, `"`) {
			w.Header().Set("ETag", "W/"+tag)
		}
		w.Header().Set("Content-Encoding", "gzip")
		write(w, buf.status, packed)
	})
}

func write(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	w.Write(body)
}

func patchVary(h http.Header) {
	const name = "Accept-Encoding"
	current := h.Get("Vary")
	for _, part := range strings.Split(current, ",") {
		if strings.EqualFold(strings.TrimSpace(part), name) {
			return
		}
	}
	if strings.TrimSpace(current) == "" {
		h.Set("Vary", name)
		return
	}
	h.Set("Vary", current+", "+name)
}

func pack(body []byte) ([]byte, error) {
	var out bytes.Buffer
	zw, err := gzip.NewWriterLevel(&out, level)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(body); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return withPadding(out.Bytes())
}

// withPadding hides a run of filler in the header. Only its length is random,
// which is all it takes to stop a compressed length from measuring the page.
func withPadding(packed []byte) ([]byte, error) {
	if len(packed) < headerSize {
		return packed, nil
	}
	size, err := rand.Int(rand.Reader, big.NewInt(maxPadding))
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(packed)+int(size.Int64())+1)
	out = append(out, packed[:3]...)
	out = append(out, flagName)
	out = append(out, packed[4:headerSize]...)
	out = append(out, bytes.Repeat([]byte("a"), int(size.Int64()))...)
	out = append(out, 0)
	return append(out, packed[headerSize:]...), nil
}

type buffer struct {
	header http.Header
	status int
	body   bytes.Buffer
	wrote  bool
}

func (b *buffer) Header() http.Header { return b.header }

func (b *buffer) WriteHeader(status int) {
	if !b.wrote {
		b.status = status
		b.wrote = true
	}
}

func (b *buffer) Write(p []byte) (int, error) {
	b.wrote = true
	return b.body.Write(p)
}
