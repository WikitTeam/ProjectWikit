package account

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	verifyAPI     = "https://wikit.unitreaty.org/projwikit"
	verifyTimeout = 5 * time.Second
)

var ErrVerifierUnreachable = errors.New("account: the verifying service did not answer")

type Verifier interface {
	Send(ctx context.Context, wikidotName string) (string, error)
	Verify(ctx context.Context, wikidotName, code string) (string, error)
}

type WikitVerifier struct {
	Base   string
	Client *http.Client
}

var _ Verifier = (*WikitVerifier)(nil)

func NewVerifier() *WikitVerifier {
	return &WikitVerifier{Base: verifyAPI, Client: &http.Client{Timeout: verifyTimeout}}
}

func (v *WikitVerifier) Send(ctx context.Context, wikidotName string) (string, error) {
	return v.call(ctx, "/send", url.Values{"user": {wikidotName}})
}

func (v *WikitVerifier) Verify(ctx context.Context, wikidotName, code string) (string, error) {
	return v.call(ctx, "/verify", url.Values{"user": {wikidotName}, "code": {code}})
}

func (v *WikitVerifier) call(ctx context.Context, path string, form url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.Base+path, strings.NewReader(form.Encode()))
	if err != nil {
		return "", ErrVerifierUnreachable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.Client.Do(req)
	if err != nil {
		return "", ErrVerifierUnreachable
	}
	defer resp.Body.Close()

	var answer struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if json.NewDecoder(resp.Body).Decode(&answer) != nil {
		return "", ErrVerifierUnreachable
	}
	if answer.Status == "success" {
		return "", nil
	}
	return answer.Message, errRefused
}

var errRefused = errors.New("account: the verifying service refused")
