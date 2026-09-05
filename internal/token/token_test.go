package token

import (
	"testing"
	"time"
)

const probeSecret = "WGspp33d4lVMRwwW6pBzpSLE93k9gpvA681QrxUHbhqGiY3QcLKREpZynWKK"

const probeHash = "pbkdf2_sha256$1000000$abc$def"

func TestBase36String(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{35, "z"},
		{36, "10"},
		{800000000, "d8ary8"},
	}
	for _, c := range cases {
		if got := base36String(c.in); got != c.want {
			t.Errorf("base36String(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBase36(t *testing.T) {
	got, err := base36("d8ary8")
	if err != nil {
		t.Fatalf("base36(%q) err = %v, want nil", "d8ary8", err)
	}
	if got != 800000000 {
		t.Errorf("base36(%q) = %d, want %d", "d8ary8", got, 800000000)
	}
}

func TestInviteTokenMatchesTheStoredVector(t *testing.T) {
	g := Generator{Secret: probeSecret}
	got := g.at(InviteValue(42, false), 800000000, probeSecret)
	want := "d8ary8-a81c622f74019e4aa39dedd404948fc4"
	if got != want {
		t.Errorf("invite token = %q, want %q", got, want)
	}
}

func TestResetTokenMatchesTheStoredVector(t *testing.T) {
	g := Generator{Secret: probeSecret}
	email := "probe@example.invalid"

	got := g.at(ResetValue(42, probeHash, nil, email), 800000000, probeSecret)
	want := "d8ary8-e5d7394bc440f5998763e5a97bae025a"
	if got != want {
		t.Errorf("reset token without a last login = %q, want %q", got, want)
	}

	last := time.Date(2026, 9, 5, 11, 4, 34, 123456000, time.UTC)
	got = g.at(ResetValue(42, probeHash, &last, email), 800000000, probeSecret)
	want = "d8ary8-162f7a3cddde7425ee22e0df0ce75649"
	if got != want {
		t.Errorf("reset token with a last login = %q, want %q", got, want)
	}
}

func TestCheckAcceptsWhatMakeMinted(t *testing.T) {
	g := Generator{Secret: probeSecret}
	now := time.Now()
	value := InviteValue(7, false)
	if got := g.Check(g.Make(value, now), value, now); !got {
		t.Errorf("Check(Make()) = false, want true")
	}
}

func TestCheckRefusesAnExpiredToken(t *testing.T) {
	g := Generator{Secret: probeSecret, Timeout: time.Hour}
	minted := time.Now().Add(-2 * time.Hour)
	value := InviteValue(7, false)
	if got := g.Check(g.Make(value, minted), value, time.Now()); got {
		t.Errorf("Check(two hours old) = true, want false")
	}
}

func TestCheckRefusesAChangedValue(t *testing.T) {
	g := Generator{Secret: probeSecret}
	now := time.Now()
	minted := g.Make(InviteValue(7, false), now)
	if got := g.Check(minted, InviteValue(7, true), now); got {
		t.Errorf("Check(after activation) = true, want false")
	}
}

func TestCheckRefusesMalformedTokens(t *testing.T) {
	g := Generator{Secret: probeSecret}
	for _, in := range []string{"", "nodash", "!!-abc", "d8ary8-"} {
		if got := g.Check(in, InviteValue(7, false), time.Now()); got {
			t.Errorf("Check(%q) = true, want false", in)
		}
	}
}

func TestCheckTriesTheFallbackSecret(t *testing.T) {
	old := Generator{Secret: "old-secret"}
	now := time.Now()
	value := InviteValue(7, false)
	minted := old.Make(value, now)

	rotated := Generator{Secret: probeSecret, Fallbacks: []string{"old-secret"}}
	if got := rotated.Check(minted, value, now); !got {
		t.Errorf("Check(minted under the old secret) = false, want true")
	}
}
