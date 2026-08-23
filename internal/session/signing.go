package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var (
	ErrBadSignature      = errors.New("session: signature does not match")
	ErrMalformed         = errors.New("session: malformed signed value")
	ErrSignatureExpired  = errors.New("session: signature expired")
	ErrUnsupportedFormat = errors.New("session: unsupported payload format")
)

const separator = ":"

type Signer struct {
	Key       string
	Fallbacks []string
	Salt      string
}

func (s Signer) signature(value, key string) string {
	return b64Encode(saltedHMAC(s.Salt+"signer", key, value))
}

// keys puts the current secret first, the order Django tries them in while a
// key rotation is in flight.
func (s Signer) keys() []string {
	return append([]string{s.Key}, s.Fallbacks...)
}

// saltedHMAC derives a key from the salt and the secret before signing, so two
// different salts never produce the same signature for the same value.
func saltedHMAC(keySalt, secret, value string) []byte {
	derived := sha256.Sum256([]byte(keySalt + secret))
	mac := hmac.New(sha256.New, derived[:])
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (s Signer) Sign(value string) string {
	return value + separator + s.signature(value, s.Key)
}

func (s Signer) Unsign(signed string) (string, error) {
	index := strings.LastIndex(signed, separator)
	if index < 0 {
		return "", ErrMalformed
	}
	value, want := signed[:index], signed[index+1:]
	for _, key := range append([]string{s.Key}, s.Fallbacks...) {
		got := s.signature(value, key)
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1 {
			return value, nil
		}
	}
	return "", ErrBadSignature
}

type TimestampSigner struct {
	Signer
	Now func() time.Time
}

func (s TimestampSigner) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s TimestampSigner) Sign(value string) string {
	return s.Signer.Sign(value + separator + b62Encode(s.now().Unix()))
}

func (s TimestampSigner) Unsign(signed string, maxAge time.Duration) (string, error) {
	stamped, err := s.Signer.Unsign(signed)
	if err != nil {
		return "", err
	}
	index := strings.LastIndex(stamped, separator)
	if index < 0 {
		return "", ErrMalformed
	}
	value, stamp := stamped[:index], stamped[index+1:]
	signedAt, err := b62Decode(stamp)
	if err != nil {
		return "", ErrMalformed
	}
	if maxAge > 0 && s.now().Sub(time.Unix(signedAt, 0)) > maxAge {
		return "", ErrSignatureExpired
	}
	return value, nil
}

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func b62Encode(n int64) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var out []byte
	for n > 0 {
		out = append([]byte{base62Alphabet[n%62]}, out...)
		n /= 62
	}
	return sign + string(out)
}

func b62Decode(s string) (int64, error) {
	if s == "" {
		return 0, ErrMalformed
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	var n int64
	for _, r := range s {
		index := strings.IndexRune(base62Alphabet, r)
		if index < 0 {
			return 0, ErrMalformed
		}
		n = n*62 + int64(index)
	}
	if negative {
		n = -n
	}
	return n, nil
}

func b64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
