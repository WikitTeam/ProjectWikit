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
	verifyTimeout = 25 * time.Second
)

const (
	CodeUserNotFound       = "user_not_found"
	CodeInvalidUser        = "invalid_user"
	CodeMissingUser        = "missing_user"
	CodeMissingCode        = "missing_code"
	CodeInvalidCode        = "invalid_code"
	CodeRateLimited        = "rate_limited"
	CodeAlreadySent        = "already_sent"
	CodeMessagesRefused    = "messages_refused"
	CodeLoginFailed        = "login_failed"
	CodeWikidotUnavailable = "wikidot_unavailable"
	CodeSignInPaused       = "sign_in_paused"
	CodeNoPendingCode      = "no_pending_code"
	CodeExpired            = "code_expired"
	CodeWrong              = "wrong_code"
	CodeTooManyAttempts    = "too_many_attempts"
	CodeInternal           = "internal_error"
)

var ErrVerifierUnreachable = errors.New("account: the verifying service did not answer")

type RefusedError struct {
	Code    string
	Message string
}

func (e *RefusedError) Error() string {
	if e.Code == "" {
		return "account: verification refused: " + e.Message
	}
	return "account: verification refused: " + e.Code
}

func refused(code string) error { return &RefusedError{Code: code} }

type ClaimTarget struct {
	UserID int64
	Name   string
	Site   string
}

type Verifier interface {
	Send(ctx context.Context, target ClaimTarget) error
	Verify(ctx context.Context, target ClaimTarget, code string) error
}

type WikitVerifier struct {
	Base   string
	Client *http.Client
}

var _ Verifier = (*WikitVerifier)(nil)

func NewVerifier() *WikitVerifier {
	return &WikitVerifier{Base: verifyAPI, Client: &http.Client{Timeout: verifyTimeout}}
}

func (v *WikitVerifier) Send(ctx context.Context, target ClaimTarget) error {
	return v.call(ctx, "/send", url.Values{"user": {target.Name}})
}

func (v *WikitVerifier) Verify(ctx context.Context, target ClaimTarget, code string) error {
	return v.call(ctx, "/verify", url.Values{"user": {target.Name}, "code": {code}})
}

func (v *WikitVerifier) call(ctx context.Context, path string, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.Base+path, strings.NewReader(form.Encode()))
	if err != nil {
		return ErrVerifierUnreachable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.Client.Do(req)
	if err != nil {
		return ErrVerifierUnreachable
	}
	defer resp.Body.Close()

	var answer struct {
		Status  string `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.NewDecoder(resp.Body).Decode(&answer) != nil {
		return ErrVerifierUnreachable
	}
	if answer.Status == "success" {
		return nil
	}
	return &RefusedError{Code: answer.Code, Message: answer.Message}
}
