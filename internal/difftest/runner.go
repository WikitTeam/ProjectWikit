package difftest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxBodyBytes = 32 << 20

type Request struct {
	Method       string
	Target       string
	Header       http.Header
	Body         []byte
	KnownDiffers bool
}

func (r Request) String() string { return r.Method + " " + r.Target }

type Runner struct {
	A        string
	B        string
	Host     string
	Client   *http.Client
	Comparer *Comparer
}

func NewRunner(a, b string) (*Runner, error) {
	baseA, err := parseBase(a)
	if err != nil {
		return nil, err
	}
	baseB, err := parseBase(b)
	if err != nil {
		return nil, err
	}
	return &Runner{
		A:        baseA,
		B:        baseB,
		Comparer: NewComparer(),
		Client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func parseBase(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse base URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("base URL %q scheme = %q, want http or https", raw, u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("base URL %q has no host", raw)
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

func (r *Runner) Do(ctx context.Context, req Request) (Result, Response, Response, error) {
	respA, err := r.fetch(ctx, r.A, req)
	if err != nil {
		return Result{}, Response{}, Response{}, fmt.Errorf("side a: %w", err)
	}
	respB, err := r.fetch(ctx, r.B, req)
	if err != nil {
		return Result{}, Response{}, Response{}, fmt.Errorf("side b: %w", err)
	}
	return r.Comparer.Compare(respA, respB), respA, respB, nil
}

func (r *Runner) fetch(ctx context.Context, base string, req Request) (Response, error) {
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, base+req.Target, body)
	if err != nil {
		return Response{}, err
	}
	for name, values := range req.Header {
		for _, v := range values {
			httpReq.Header.Add(name, v)
		}
	}
	if r.Host != "" {
		httpReq.Host = r.Host
	}

	resp, err := r.Client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return Response{}, err
	}
	return Response{Status: resp.StatusCode, Header: resp.Header, Body: raw}, nil
}

func ParseCorpus(src string) ([]Request, error) {
	var out []Request
	for i, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		known := strings.HasPrefix(line, "!")
		line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
		fields := strings.Fields(line)
		var method, target string
		switch len(fields) {
		case 1:
			method, target = http.MethodGet, fields[0]
		case 2:
			method, target = strings.ToUpper(fields[0]), fields[1]
		default:
			return nil, fmt.Errorf("corpus line %d: %q has %d fields, want 1 or 2", i+1, line, len(fields))
		}
		if !strings.HasPrefix(target, "/") {
			return nil, fmt.Errorf("corpus line %d: target %q does not start with /", i+1, target)
		}
		out = append(out, Request{Method: method, Target: target, KnownDiffers: known})
	}
	return out, nil
}
