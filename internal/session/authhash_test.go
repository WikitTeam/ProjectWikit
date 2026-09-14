package session

import "testing"

var authHashVectors = []struct {
	secret       string
	passwordHash string
	want         string
}{
	{"test-secret-key", "pbkdf2_sha256$1000000$abc$def=", "a56c13d7300d3e79d8da7d40a06e91a86ae9d9953c8776915b53fea461fb2b49"},
	{"test-secret-key", "", "24e1355b81410e8aa16511eaed2aee138d8a36bc5771440c070100f3fcad2744"},
	{"中文密钥", "pbkdf2_sha256$1000000$abc$def=", "af3bdce41cde90fd4834d23e402cdd6e6646486dc7cab56ab04b7bb315692dce"},
	{"another", "x", "9c2bacf6821df50e1424a1e04b7aaa2dfbaafd4d45a89990bae1b8c67c941ee6"},
}

func TestAuthHash(t *testing.T) {
	for _, tt := range authHashVectors {
		t.Run(tt.secret+"/"+tt.passwordHash, func(t *testing.T) {
			if got := New(tt.secret).AuthHash(tt.passwordHash); got != tt.want {
				t.Errorf("AuthHash(%q) = %q, want %q", tt.passwordHash, got, tt.want)
			}
		})
	}
}

func TestAuthHashMatches(t *testing.T) {
	const password = "pbkdf2_sha256$1000000$abc$def="
	current := New("current-secret")
	rotated := New("current-secret", "old-secret")

	tests := []struct {
		name  string
		store *Store
		want  string
		match bool
	}{
		{"current secret", current, current.AuthHash(password), true},
		{"other secret", current, New("old-secret").AuthHash(password), false},
		{"fallback secret", rotated, New("old-secret").AuthHash(password), true},
		{"unknown secret", rotated, New("never-used").AuthHash(password), false},
		{"empty", current, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.store.AuthHashMatches(password, tt.want); got != tt.match {
				t.Errorf("AuthHashMatches(password, %q) = %v, want %v", tt.want, got, tt.match)
			}
		})
	}
}

func TestAuthHashChangesWithPassword(t *testing.T) {
	s := New("secret")
	before := s.AuthHash("pbkdf2_sha256$1000000$abc$old=")
	after := s.AuthHash("pbkdf2_sha256$1000000$abc$new=")

	if before == after {
		t.Errorf("AuthHash(old) = AuthHash(new) = %q, want them to differ", before)
	}
	if s.AuthHashMatches("pbkdf2_sha256$1000000$abc$new=", before) {
		t.Error("AuthHashMatches(new password, old hash) = true, want false")
	}
}
