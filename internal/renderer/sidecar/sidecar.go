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

type Renderer struct {
	mu   sync.Mutex
	in   io.Writer
	out  *bufio.Reader
	cmd  *exec.Cmd
	stop func() error
}

var _ renderer.Renderer = (*Renderer)(nil)

func New(binary string) (*Renderer, error) {
	cmd := exec.Command(binary)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open sidecar stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open sidecar stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start sidecar %q: %w", binary, err)
	}

	return &Renderer{
		in:  stdin,
		out: bufio.NewReader(stdout),
		cmd: cmd,
		stop: func() error {
			stdin.Close()
			return cmd.Wait()
		},
	}, nil
}

func newOver(w io.Writer, r io.Reader) *Renderer {
	return &Renderer{in: w, out: bufio.NewReader(r), stop: func() error { return nil }}
}

func (r *Renderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stop == nil {
		return nil
	}
	stop := r.stop
	r.stop = nil
	return stop()
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

	r.mu.Lock()
	defer r.mu.Unlock()

	req := map[string]any{
		"type":      "render",
		"op":        op,
		"mode":      string(mode),
		"source":    source,
		"page_info": encodePageInfo(info),
	}
	if err := r.send(req); err != nil {
		return renderer.Result{}, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return renderer.Result{}, err
		}

		var msg wireResult
		if err := r.recv(&msg); err != nil {
			return renderer.Result{}, err
		}

		switch msg.Type {
		case "result":
			code := make([]renderer.CodeBlock, 0, len(msg.Code))
			for _, c := range msg.Code {
				code = append(code, renderer.CodeBlock{Language: c[0], Source: c[1]})
			}
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
			if err := r.send(map[string]any{"type": "callback_result", "value": value}); err != nil {
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

func (r *Renderer) send(v any) error {
	buf, err := marshal(v)
	if err != nil {
		return fmt.Errorf("encode outbound message: %w", err)
	}

	var head [4]byte
	binary.BigEndian.PutUint32(head[:], uint32(len(buf)))
	if _, err := r.in.Write(head[:]); err != nil {
		return fmt.Errorf("write message length: %w", err)
	}
	if _, err := r.in.Write(buf); err != nil {
		return fmt.Errorf("write message body: %w", err)
	}
	return nil
}

func (r *Renderer) recv(v any) error {
	var head [4]byte
	if _, err := io.ReadFull(r.out, head[:]); err != nil {
		return fmt.Errorf("read message length: %w", err)
	}
	n := binary.BigEndian.Uint32(head[:])
	if n > maxMessageBytes {
		return fmt.Errorf("message length %d exceeds limit %d", n, maxMessageBytes)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r.out, buf); err != nil {
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
