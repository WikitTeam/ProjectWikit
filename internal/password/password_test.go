package password

import (
	"errors"
	"strings"
	"testing"
)

var djangoVectors = []struct {
	plain   string
	encoded string
}{
	{plain: "correcthorsebatterystaple", encoded: "pbkdf2_sha256$1000$abcdefghijkl$M2T6/VMpP+pv6574Q6rE1HZjkIlFvhvAWQlQFfgUncY="},
	{plain: "p", encoded: "pbkdf2_sha256$1$0123456789ab$ZYeXRJ1IKaitaR+tVMtfwRoaCmtfvWEeGZKPuEHE55w="},
	{plain: "", encoded: "pbkdf2_sha256$100$saltsaltsalt$0dnkxOdo5UI8+lv6MjQ8bizY4JHzDAy3XnoP9bTdlbI="},
	{plain: "密码带中文", encoded: "pbkdf2_sha256$5000$zZ9aA0bB1cC2$AkWlqnlCcYg6ztQSFkj/YOqgMPaLxBgkjEOFSvcL01I="},
	{plain: "emoji😀pw", encoded: "pbkdf2_sha256$260000$qwertyuiopas$Jqj65m/ULEd6isdZMIhEbtRX78bl3yCtXZfdDFbpo9w="},
	{plain: "has$dollar", encoded: "pbkdf2_sha256$1000000$AAAAAAAAAAAA$/JBeAbrjboPXQutZ0T/2o6tweC7dZlPwNHYsJbtX0ow="},
	{plain: strings.Repeat("x", 200), encoded: "pbkdf2_sha256$12345$longpassword$yzqzDvqr/T9CJtexJxNc2dH7YtwOFoYUAom3dauirMA="},
}

func TestVerifyAcceptsDjangoHashes(t *testing.T) {
	for _, v := range djangoVectors {
		ok, err := Verify(v.plain, v.encoded)
		if err != nil {
			t.Errorf("Verify(%q) err = %v，期望 nil", v.plain, err)
			continue
		}
		if !ok {
			t.Errorf("Verify(%q) = false，期望 true", v.plain)
		}
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	for _, v := range djangoVectors {
		ok, err := Verify(v.plain+"x", v.encoded)
		if err != nil {
			t.Errorf("Verify(%q+x) err = %v，期望 nil", v.plain, err)
			continue
		}
		if ok {
			t.Errorf("Verify(%q+x) = true，期望 false", v.plain)
		}
	}
}

func TestVerifyRejectsTamperedFields(t *testing.T) {
	v := djangoVectors[0]
	parts := strings.Split(v.encoded, "$")

	tampered := []struct {
		name    string
		encoded string
	}{
		{"迭代次数被改", strings.Join([]string{parts[0], "1001", parts[2], parts[3]}, "$")},
		{"salt 被改", strings.Join([]string{parts[0], parts[1], "abcdefghijkm", parts[3]}, "$")},
	}
	for _, tt := range tampered {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := Verify(v.plain, tt.encoded)
			if err != nil {
				t.Fatalf("Verify() err = %v，期望 nil", err)
			}
			if ok {
				t.Error("Verify() = true，期望 false")
			}
		})
	}
}

func TestHashRoundTrip(t *testing.T) {
	for _, plain := range []string{"", "p", "密码带中文", "emoji😀pw", "has$dollar", strings.Repeat("x", 200)} {
		encoded, err := Hash(plain)
		if err != nil {
			t.Fatalf("Hash(%q) err = %v，期望 nil", plain, err)
		}
		ok, err := Verify(plain, encoded)
		if err != nil {
			t.Fatalf("Verify(%q) err = %v，期望 nil", plain, err)
		}
		if !ok {
			t.Errorf("Verify(%q, Hash(%q)) = false，期望 true", plain, plain)
		}
		if ok, _ := Verify(plain+"x", encoded); ok {
			t.Errorf("Verify(%q+x, Hash(%q)) = true，期望 false", plain, plain)
		}
	}
}

func TestHashOutputMatchesDjangoFormat(t *testing.T) {
	encoded, err := Hash("pw")
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(encoded, "$")
	if len(parts) != 4 {
		t.Fatalf("段数 = %d，期望 4：%q", len(parts), encoded)
	}
	if parts[0] != Algorithm {
		t.Errorf("算法 = %q，期望 %q", parts[0], Algorithm)
	}
	if parts[1] != "1000000" {
		t.Errorf("迭代次数 = %q，期望 %d", parts[1], DefaultIterations)
	}
	if len(parts[2]) != saltLength {
		t.Errorf("len(salt) = %d，期望 %d", len(parts[2]), saltLength)
	}
	for _, c := range parts[2] {
		if !strings.ContainsRune(saltChars, c) {
			t.Errorf("salt = %q，含 saltChars 之外的字符 %q", parts[2], c)
		}
	}
}

func TestHashSaltIsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for range 20 {
		encoded, err := Hash("pw")
		if err != nil {
			t.Fatal(err)
		}
		salt := strings.Split(encoded, "$")[2]
		if seen[salt] {
			t.Fatalf("salt = %q，与前一次生成的重复", salt)
		}
		seen[salt] = true
	}
}

func TestVerifyUnusablePassword(t *testing.T) {
	for _, encoded := range []string{"!", "!xyzzy", "!" + djangoVectors[0].encoded} {
		if IsUsable(encoded) {
			t.Errorf("IsUsable(%q) = true，期望 false", encoded)
		}
		ok, err := Verify(djangoVectors[0].plain, encoded)
		if err != nil {
			t.Errorf("Verify(%q) err = %v，期望 nil", encoded, err)
		}
		if ok {
			t.Errorf("Verify(%q) = true，期望 false", encoded)
		}
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	malformed := []string{
		"",
		"pbkdf2_sha256",
		"pbkdf2_sha256$1000",
		"pbkdf2_sha256$1000$salt",
		"pbkdf2_sha256$abc$salt$aGFzaA==",
		"pbkdf2_sha256$0$salt$aGFzaA==",
		"pbkdf2_sha256$-1$salt$aGFzaA==",
		"pbkdf2_sha256$1000$$aGFzaA==",
		"pbkdf2_sha256$1000$salt$不是base64",
	}
	for _, encoded := range malformed {
		ok, err := Verify("pw", encoded)
		if ok {
			t.Errorf("Verify(%q) = true，期望 false", encoded)
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("Verify(%q) err = %v，期望 ErrMalformed", encoded, err)
		}
	}
}

func TestVerifyRejectsOtherAlgorithms(t *testing.T) {
	others := []string{
		"argon2$argon2id$v=19$m=102400,t=2,p=8$c2FsdA$aGFzaA",
		"bcrypt_sha256$$2b$12$saltsaltsaltsaltsaltsuhash",
		"md5$salt$5f4dcc3b5aa765d61d8327deb882cf99",
		"pbkdf2_sha1$260000$salt$aGFzaA==",
		"scrypt$16384$salt$8$1$aGFzaA==",
	}
	for _, encoded := range others {
		ok, err := Verify("pw", encoded)
		if ok {
			t.Errorf("Verify(%q) = true，期望 false", encoded)
		}
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Verify(%q) err = %v，期望 ErrUnsupported", encoded, err)
		}
	}
}

func TestNeedsRehash(t *testing.T) {
	fresh, err := Hash("pw")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		encoded string
		want    bool
	}{
		{"刚生成的", fresh, false},
		{"迭代次数偏低", djangoVectors[0].encoded, true},
		{"迭代次数已达标", djangoVectors[5].encoded, false},
		{"不可用密码", "!xyzzy", false},
		{"格式损坏", "garbage", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedsRehash(tt.encoded); got != tt.want {
				t.Errorf("NeedsRehash(%q) = %v，期望 %v", tt.encoded, got, tt.want)
			}
		})
	}
}
