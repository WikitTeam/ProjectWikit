// Package token mints and checks the one-time links sent by mail.
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

const Salt = "django.contrib.auth.tokens.PasswordResetTokenGenerator"

var epoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.Local)

const DefaultTimeout = 3 * 24 * time.Hour

var ErrMalformed = errors.New("token: malformed")

type Generator struct {
	Secret    string
	Fallbacks []string
	Timeout   time.Duration
}

type Value func(timestamp int64) string

func (g Generator) Make(value Value, now time.Time) string {
	return g.at(value, seconds(now), g.Secret)
}

func (g Generator) Check(token string, value Value, now time.Time) bool {
	raw, _, found := strings.Cut(token, "-")
	if !found {
		return false
	}
	stamp, err := base36(raw)
	if err != nil {
		return false
	}
	matched := false
	for _, secret := range append([]string{g.Secret}, g.Fallbacks...) {
		want := g.at(value, stamp, secret)
		if subtle.ConstantTimeCompare([]byte(want), []byte(token)) == 1 {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	timeout := g.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return seconds(now)-stamp <= int64(timeout/time.Second)
}

func (g Generator) at(value Value, stamp int64, secret string) string {
	full := hex.EncodeToString(saltedHMAC(Salt, secret, value(stamp)))
	var short strings.Builder
	for i := 0; i < len(full); i += 2 {
		short.WriteByte(full[i])
	}
	return base36String(stamp) + "-" + short.String()
}

func seconds(now time.Time) int64 {
	return int64(now.Sub(epoch) / time.Second)
}

func saltedHMAC(keySalt, secret, value string) []byte {
	derived := sha256.Sum256([]byte(keySalt + secret))
	mac := hmac.New(sha256.New, derived[:])
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

const base36Alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

func base36String(n int64) string {
	if n < 0 {
		return ""
	}
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{base36Alphabet[n%36]}, out...)
		n /= 36
	}
	return string(out)
}

func base36(s string) (int64, error) {
	if s == "" || len(s) > 13 {
		return 0, ErrMalformed
	}
	var n int64
	for _, r := range s {
		index := strings.IndexRune(base36Alphabet, r)
		if index < 0 {
			return 0, ErrMalformed
		}
		n = n*36 + int64(index)
	}
	return n, nil
}

func InviteValue(userID int64, active bool) Value {
	return func(stamp int64) string {
		return "v2:" + strconv.FormatInt(userID, 10) + strconv.FormatInt(stamp, 10) + pyBool(active)
	}
}

func ResetValue(userID int64, hash string, lastLogin *time.Time, email string) Value {
	stamped := ""
	if lastLogin != nil {
		stamped = lastLogin.UTC().Format("2006-01-02 15:04:05")
	}
	return func(stamp int64) string {
		return strconv.FormatInt(userID, 10) + hash + stamped + strconv.FormatInt(stamp, 10) + email
	}
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
