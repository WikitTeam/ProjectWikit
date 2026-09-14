package session

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testKey = "test-secret-key-do-not-use"

func fixedSigner() TimestampSigner {
	return TimestampSigner{
		Signer: Signer{Key: testKey, Salt: Salt},
		Now:    func() time.Time { return time.Unix(1700000000, 0) },
	}
}

func TestSignerSign(t *testing.T) {
	s := Signer{Key: testKey, Salt: Salt}
	cases := []struct{ in, want string }{
		{"", ":FzRimgS7ruXRai6zfrnzT14GTW9N2hrqJvE71ZoiZzw"},
		{"a", "a:URQ8cyCMMAyPvF3FEW2ti_QEthrxnwZFpAHgh2-Ue0o"},
		{"hello:world", "hello:world:4sW025UTjG_sh0qqc-xOvx1QdLnMjn91kGU7a1JEBAA"},
		{"中文", "中文:5EKMCJEfU_mpuY9jQ_SFTG4ORc8r_RrTb6OoSsNoGDM"},
		{strings.Repeat("x", 100), strings.Repeat("x", 100) + ":BIOeayUpmJcOWshL79FkiFJViuzFTeZZboUvrwYTnqM"},
	}
	for _, c := range cases {
		if got := s.Sign(c.in); got != c.want {
			t.Errorf("Sign(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSignerUnsignRoundTrip(t *testing.T) {
	s := Signer{Key: testKey, Salt: Salt}
	for _, value := range []string{"", "a", "hello:world", "中文"} {
		got, err := s.Unsign(s.Sign(value))
		if err != nil {
			t.Errorf("Unsign(Sign(%q)) err = %v, want nil", value, err)
			continue
		}
		if got != value {
			t.Errorf("Unsign(Sign(%q)) = %q, want %q", value, got, value)
		}
	}
}

func TestSignerUnsignRejectsTamperedValue(t *testing.T) {
	s := Signer{Key: testKey, Salt: Salt}
	signed := s.Sign("payload")

	if _, err := s.Unsign("x" + signed); !errors.Is(err, ErrBadSignature) {
		t.Errorf("Unsign(tampered value) err = %v, want ErrBadSignature", err)
	}
	if _, err := s.Unsign(signed + "x"); !errors.Is(err, ErrBadSignature) {
		t.Errorf("Unsign(tampered signature) err = %v, want ErrBadSignature", err)
	}
	if _, err := s.Unsign("nosep"); !errors.Is(err, ErrMalformed) {
		t.Errorf("Unsign(%q) err = %v, want ErrMalformed", "nosep", err)
	}
}

func TestSignerUnsignRejectsOtherKey(t *testing.T) {
	signed := Signer{Key: testKey, Salt: Salt}.Sign("payload")
	other := Signer{Key: "another-key", Salt: Salt}

	if _, err := other.Unsign(signed); !errors.Is(err, ErrBadSignature) {
		t.Errorf("Unsign(signed with other key) err = %v, want ErrBadSignature", err)
	}
}

func TestSignerUnsignAcceptsFallbackKey(t *testing.T) {
	signed := Signer{Key: "old-key", Salt: Salt}.Sign("payload")
	rotated := Signer{Key: testKey, Fallbacks: []string{"old-key"}, Salt: Salt}

	got, err := rotated.Unsign(signed)
	if err != nil {
		t.Fatalf("Unsign(signed with rotated-out key) err = %v, want nil", err)
	}
	if got != "payload" {
		t.Errorf("Unsign() = %q, want %q", got, "payload")
	}
}

func TestSignerUnsignRejectsOtherSalt(t *testing.T) {
	signed := Signer{Key: testKey, Salt: Salt}.Sign("payload")
	other := Signer{Key: testKey, Salt: "django.core.signing"}

	if _, err := other.Unsign(signed); !errors.Is(err, ErrBadSignature) {
		t.Errorf("Unsign(signed with other salt) err = %v, want ErrBadSignature", err)
	}
}

func TestTimestampSignerSign(t *testing.T) {
	s := fixedSigner()
	cases := []struct{ in, want string }{
		{"payload", "payload:1r31eq:iQ1488k_heunTWUtV6k_LJjaGJmNkeoN3BZYL9E8fl4"},
		{"", ":1r31eq:3HUAMJk1BHsY54BfWCXUyCqnP2KtW3EWdO6kOb6ZrLw"},
	}
	for _, c := range cases {
		if got := s.Sign(c.in); got != c.want {
			t.Errorf("Sign(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTimestampSignerUnsignChecksAge(t *testing.T) {
	signed := fixedSigner().Sign("payload")
	later := TimestampSigner{
		Signer: Signer{Key: testKey, Salt: Salt},
		Now:    func() time.Time { return time.Unix(1700000000+3600, 0) },
	}

	if _, err := later.Unsign(signed, 2*time.Hour); err != nil {
		t.Errorf("Unsign(age 1h, maxAge 2h) err = %v, want nil", err)
	}
	if _, err := later.Unsign(signed, 30*time.Minute); !errors.Is(err, ErrSignatureExpired) {
		t.Errorf("Unsign(age 1h, maxAge 30m) err = %v, want ErrSignatureExpired", err)
	}
	if _, err := later.Unsign(signed, 0); err != nil {
		t.Errorf("Unsign(maxAge 0) err = %v, want nil", err)
	}
}

func TestBase62(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{61, "z"},
		{62, "10"},
		{1700000000, "1r31eq"},
		{-5, "-5"},
	}
	for _, c := range cases {
		if got := b62Encode(c.in); got != c.want {
			t.Errorf("b62Encode(%d) = %q, want %q", c.in, got, c.want)
		}
		got, err := b62Decode(c.want)
		if err != nil {
			t.Errorf("b62Decode(%q) err = %v, want nil", c.want, err)
			continue
		}
		if got != c.in {
			t.Errorf("b62Decode(%q) = %d, want %d", c.want, got, c.in)
		}
	}
}

func TestBase62DecodeRejectsBadInput(t *testing.T) {
	for _, in := range []string{"", "!", "1!"} {
		if _, err := b62Decode(in); !errors.Is(err, ErrMalformed) {
			t.Errorf("b62Decode(%q) err = %v, want ErrMalformed", in, err)
		}
	}
}
