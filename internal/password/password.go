package password

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	Algorithm         = "pbkdf2_sha256"
	DefaultIterations = 1000000
)

const (
	saltLength     = 12
	saltChars      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	unusablePrefix = "!"
)

var (
	ErrMalformed   = errors.New("malformed password hash")
	ErrUnsupported = errors.New("unsupported password hash algorithm")
)

type parsed struct {
	iterations int
	salt       string
	digest     []byte
}

func IsUsable(encoded string) bool {
	return !strings.HasPrefix(encoded, unusablePrefix)
}

func Verify(plain, encoded string) (bool, error) {
	if !IsUsable(encoded) {
		return false, nil
	}
	p, err := parse(encoded)
	if err != nil {
		return false, err
	}
	got, err := derive(plain, p.salt, p.iterations)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(got, p.digest) == 1, nil
}

func Hash(plain string) (string, error) {
	salt, err := newSalt()
	if err != nil {
		return "", err
	}
	return encode(plain, salt, DefaultIterations)
}

func NeedsRehash(encoded string) bool {
	if !IsUsable(encoded) {
		return false
	}
	p, err := parse(encoded)
	if err != nil {
		return true
	}
	return p.iterations != DefaultIterations
}

func parse(encoded string) (parsed, error) {
	if encoded == "" {
		return parsed{}, fmt.Errorf("%w: empty string", ErrMalformed)
	}
	parts := strings.SplitN(encoded, "$", 4)
	if parts[0] != Algorithm {
		return parsed{}, fmt.Errorf("%w: %q", ErrUnsupported, parts[0])
	}
	if len(parts) != 4 {
		return parsed{}, fmt.Errorf("%w: want 4 segments, got %d", ErrMalformed, len(parts))
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 {
		return parsed{}, fmt.Errorf("%w: iterations %q", ErrMalformed, parts[1])
	}
	if parts[2] == "" {
		return parsed{}, fmt.Errorf("%w: empty salt", ErrMalformed)
	}
	digest, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return parsed{}, fmt.Errorf("%w: hash segment is not valid base64: %v", ErrMalformed, err)
	}
	return parsed{iterations: iterations, salt: parts[2], digest: digest}, nil
}

func encode(plain, salt string, iterations int) (string, error) {
	if salt == "" || strings.Contains(salt, "$") {
		return "", fmt.Errorf("%w: salt %q must be non-empty and must not contain $", ErrMalformed, salt)
	}
	digest, err := derive(plain, salt, iterations)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		Algorithm,
		strconv.Itoa(iterations),
		salt,
		base64.StdEncoding.EncodeToString(digest),
	}, "$"), nil
}

func derive(plain, salt string, iterations int) ([]byte, error) {
	return pbkdf2.Key(sha256.New, plain, []byte(salt), iterations, sha256.Size)
}

func newSalt() (string, error) {
	limit := byte(256 - 256%len(saltChars))
	out := make([]byte, 0, saltLength)
	buf := make([]byte, saltLength)
	for len(out) < saltLength {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate salt: %w", err)
		}
		for _, b := range buf {
			if b < limit && len(out) < saltLength {
				out = append(out, saltChars[int(b)%len(saltChars)])
			}
		}
	}
	return string(out), nil
}
