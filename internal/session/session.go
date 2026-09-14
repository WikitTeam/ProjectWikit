package session

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	CookieName = "pwikit_sessionid"
	CookieAge  = 14 * 24 * time.Hour

	Salt = "django.contrib.sessions.SessionStore"

	KeyLength   = 32
	keyAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

	AuthUserID      = "_auth_user_id"
	AuthUserBackend = "_auth_user_backend"
	AuthUserHash    = "_auth_user_hash"

	authHashSalt = "django.contrib.auth.models.AbstractBaseUser.get_session_auth_hash"

	compressedPrefix = "."
)

type Store struct {
	Signer TimestampSigner
}

func New(secretKey string, fallbacks ...string) *Store {
	return &Store{Signer: TimestampSigner{
		Signer: Signer{Key: secretKey, Fallbacks: fallbacks, Salt: Salt},
	}}
}

func (s *Store) Encode(data map[string]any) (string, error) {
	raw, err := marshalCompact(data)
	if err != nil {
		return "", err
	}

	payload := raw
	prefix := ""
	if compressed, err := deflate(raw); err == nil && len(compressed) < len(raw)-1 {
		payload = compressed
		prefix = compressedPrefix
	}
	return s.Signer.Sign(prefix + b64Encode(payload)), nil
}

func (s *Store) Decode(sessionData string) (map[string]any, error) {
	value, err := s.Signer.Unsign(sessionData, 0)
	if err != nil {
		return nil, err
	}

	compressed := strings.HasPrefix(value, compressedPrefix)
	if compressed {
		value = value[len(compressedPrefix):]
	}
	payload, err := b64Decode(value)
	if err != nil {
		return nil, ErrUnsupportedFormat
	}
	if compressed {
		if payload, err = inflate(payload); err != nil {
			return nil, ErrUnsupportedFormat
		}
	}

	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, ErrUnsupportedFormat
	}
	return out, nil
}

// AuthHash ties a session to the password it was opened under. Without this
// check a stolen session keeps working after the password changes.
func (s *Store) AuthHash(passwordHash string) string {
	return hex.EncodeToString(saltedHMAC(authHashSalt, s.Signer.Signer.Key, passwordHash))
}

// AuthHashMatches tries the fallback secrets too, so a key rotation does not
// sign everyone out.
func (s *Store) AuthHashMatches(passwordHash, want string) bool {
	if want == "" {
		return false
	}
	for _, key := range s.Signer.Signer.keys() {
		got := hex.EncodeToString(saltedHMAC(authHashSalt, key, passwordHash))
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1 {
			return true
		}
	}
	return false
}

func UserID(data map[string]any) (string, bool) {
	id, ok := data[AuthUserID].(string)
	return id, ok && id != ""
}

func NewKey() (string, error) {
	buf := make([]byte, KeyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = keyAlphabet[int(b)%len(keyAlphabet)]
	}
	return string(buf), nil
}

func marshalCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return escapeNonASCII(bytes.TrimRight(buf.Bytes(), "\n")), nil
}

func escapeNonASCII(b []byte) []byte {
	if isASCII(b) {
		return b
	}
	var out bytes.Buffer
	for _, r := range string(b) {
		switch {
		case r < utf8.RuneSelf:
			out.WriteByte(byte(r))
		case r > 0xFFFF:
			r -= 0x10000
			fmt.Fprintf(&out, "\\u%04x\\u%04x", 0xD800+(r>>10), 0xDC00+(r&0x3FF))
		default:
			fmt.Fprintf(&out, "\\u%04x", r)
		}
	}
	return out.Bytes()
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func deflate(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write(b); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func inflate(b []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
