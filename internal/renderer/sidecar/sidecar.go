package sidecar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

const EnvBinary = "PWIKIT_FTML_SIDECAR"

const maxMessageBytes = 64 << 20

// Renderer keeps a pool rather than one process because a module may render
// again while the render that called it is still waiting for an answer.
type Renderer struct {
	binary string

	mu    sync.Mutex
	idle  []*session
	fixed bool
}

type session struct {
	in   io.Writer
	out  *bufio.Reader
	stop func() error
}

var _ renderer.Renderer = (*Renderer)(nil)

// New starts one process straight away so a wrong path is reported here rather
// than on the first page that needs it.
func New(binary string) (*Renderer, error) {
	r := &Renderer{binary: binary}
	first, err := r.start()
	if err != nil {
		return nil, err
	}
	r.idle = append(r.idle, first)
	return r, nil
}

func (r *Renderer) start() (*session, error) {
	cmd := exec.Command(r.binary)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open sidecar stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open sidecar stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start sidecar %q: %w", r.binary, err)
	}
	return &session{
		in:  stdin,
		out: bufio.NewReader(stdout),
		stop: func() error {
			stdin.Close()
			return cmd.Wait()
		},
	}, nil
}

// The caller owns the pipes, so a render that nests inside another one has to
// wait its turn.
func newOver(w io.Writer, rd io.Reader) *Renderer {
	return &Renderer{
		fixed: true,
		idle:  []*session{{in: w, out: bufio.NewReader(rd), stop: func() error { return nil }}},
	}
}

func (r *Renderer) acquire() (*session, error) {
	r.mu.Lock()
	if n := len(r.idle); n > 0 {
		s := r.idle[n-1]
		r.idle = r.idle[:n-1]
		r.mu.Unlock()
		return s, nil
	}
	r.mu.Unlock()
	if r.fixed {
		return nil, fmt.Errorf("sidecar: nested render has no second process")
	}
	return r.start()
}

func (r *Renderer) release(s *session) {
	r.mu.Lock()
	r.idle = append(r.idle, s)
	r.mu.Unlock()
}

// Retired rather than reused, since the protocol is no longer at a message
// boundary.
func (r *Renderer) drop(s *session) {
	if r.fixed {
		r.release(s)
		return
	}
	s.stop()
}

func (r *Renderer) Close() error {
	r.mu.Lock()
	idle := r.idle
	r.idle = nil
	r.mu.Unlock()

	var err error
	for _, s := range idle {
		if stopErr := s.stop(); stopErr != nil && err == nil {
			err = stopErr
		}
	}
	return err
}

func (r *Renderer) RenderHTML(ctx context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return r.render(ctx, "html", source, info, cb, mode)
}

func (r *Renderer) RenderText(ctx context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return r.render(ctx, "text", source, info, cb, mode)
}

func (r *Renderer) CollectBacklinks(ctx context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	return r.render(ctx, "html", source, info, cb, mode)
}

func (r *Renderer) CollectCodeAndHTML(ctx context.Context, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Parts, error) {
	res, err := r.render(ctx, "html", source, info, cb, mode)
	if err != nil {
		return renderer.Parts{}, err
	}
	return renderer.Parts{Code: res.Code, HTML: res.HTML}, nil
}

type wireResult struct {
	Type          string          `json:"type"`
	Message       string          `json:"message"`
	Method        string          `json:"method"`
	Args          json.RawMessage `json:"args"`
	Body          string          `json:"body"`
	IncludedPages []string        `json:"included_pages"`
	LinkedPages   []string        `json:"linked_pages"`
	Code          [][2]string     `json:"code"`
	HTML          []string        `json:"html"`
}

func (r *Renderer) render(ctx context.Context, op, source string, info renderer.PageInfo, cb renderer.Callbacks, mode renderer.Mode) (renderer.Result, error) {
	if !mode.Valid() {
		return renderer.Result{}, fmt.Errorf("unknown render mode %q", mode)
	}
	if cb == nil {
		return renderer.Result{}, fmt.Errorf("callbacks is nil")
	}

	s, err := r.acquire()
	if err != nil {
		return renderer.Result{}, err
	}
	done := false
	defer func() {
		if done {
			r.release(s)
			return
		}
		r.drop(s)
	}()

	req := map[string]any{
		"type":      "render",
		"op":        op,
		"mode":      string(mode),
		"source":    source,
		"page_info": encodePageInfo(info),
	}
	if err := s.send(req); err != nil {
		return renderer.Result{}, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return renderer.Result{}, err
		}

		var msg wireResult
		if err := s.recv(&msg); err != nil {
			return renderer.Result{}, err
		}

		switch msg.Type {
		case "result":
			code := make([]renderer.CodeBlock, 0, len(msg.Code))
			for _, c := range msg.Code {
				code = append(code, renderer.CodeBlock{Language: c[0], Source: c[1]})
			}
			done = true
			return renderer.Result{
				Body:          msg.Body,
				IncludedPages: msg.IncludedPages,
				LinkedPages:   msg.LinkedPages,
				Code:          code,
				HTML:          msg.HTML,
			}, nil

		case "error":
			return renderer.Result{}, fmt.Errorf("sidecar error: %s", msg.Message)

		case "callback":
			value, err := dispatch(cb, msg.Method, msg.Args)
			if err != nil {
				return renderer.Result{}, err
			}
			if err := s.send(map[string]any{"type": "callback_result", "value": value}); err != nil {
				return renderer.Result{}, err
			}

		default:
			return renderer.Result{}, fmt.Errorf("sidecar message type = %q, want result/error/callback", msg.Type)
		}
	}
}

func marshal(v any) ([]byte, error) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	buf := body.Bytes()
	if n := len(buf); n > 0 && buf[n-1] == '\n' {
		buf = buf[:n-1]
	}
	return buf, nil
}

func (s *session) send(v any) error {
	buf, err := marshal(v)
	if err != nil {
		return fmt.Errorf("encode outbound message: %w", err)
	}

	var head [4]byte
	binary.BigEndian.PutUint32(head[:], uint32(len(buf)))
	if _, err := s.in.Write(head[:]); err != nil {
		return fmt.Errorf("write message length: %w", err)
	}
	if _, err := s.in.Write(buf); err != nil {
		return fmt.Errorf("write message body: %w", err)
	}
	return nil
}

func (s *session) recv(v any) error {
	var head [4]byte
	if _, err := io.ReadFull(s.out, head[:]); err != nil {
		return fmt.Errorf("read message length: %w", err)
	}
	n := binary.BigEndian.Uint32(head[:])
	if n > maxMessageBytes {
		return fmt.Errorf("message length %d exceeds limit %d", n, maxMessageBytes)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(s.out, buf); err != nil {
		return fmt.Errorf("read message body: %w", err)
	}
	if err := json.Unmarshal(buf, v); err != nil {
		return fmt.Errorf("decode inbound message: %w", err)
	}
	return nil
}

func encodePageInfo(info renderer.PageInfo) map[string]any {
	tags := info.Tags
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{
		"page":         info.Page,
		"category":     info.Category,
		"site":         info.Site,
		"title":        info.Title,
		"domain":       info.Domain,
		"media_domain": info.MediaDomain,
		"rating":       info.Rating,
		"tags":         tags,
		"language":     info.Language,
	}
}
