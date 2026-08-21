package session

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fixedStore() *Store {
	store := New(testKey)
	store.Signer.Now = func() time.Time { return time.Unix(1700000000, 0) }
	return store
}

func TestDecodeDjangoSessionData(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]any
	}{
		{
			"empty",
			"e30:1r31eq:C7MzTeb-tO2S7n4gdKgSVYEs4pn1OuYtQvWVq0BOUEA",
			map[string]any{},
		},
		{
			"user id only",
			"eyJfYXV0aF91c2VyX2lkIjoiMSJ9:1r31eq:901W1QvvDfVyPTJvUtlSLvIV3In2WsSSt0PqU6EW4Go",
			map[string]any{AuthUserID: "1"},
		},
		{
			"full auth session, compressed",
			".eJyrVopPLC3JiC8tTi2Kz0xRslIyVNJBFktKTM5OzQNJpGQl5qXn6yXn55UUZSbpgZToQWWL9XzzU1JznKBqUQzISCzOAOpOTEo2NDJWqgUABVIoCQ:1r31eq:Xe8FCXIyDSHnSwZtqfF6N6SKw5yfqlTRsVGI595iAjU",
			map[string]any{
				AuthUserID:      "1",
				AuthUserBackend: "django.contrib.auth.backends.ModelBackend",
				AuthUserHash:    "abc123",
			},
		},
		{
			"large value, compressed",
			".eJyrVkrKTFeyUqocBSMOKNUCALgx70E:1r31eq:xqaKMmrsq2trIIN2u60Dc7hdNx3I9Fzk6zgmkzNuP70",
			map[string]any{"big": strings.Repeat("y", 500)},
		},
	}

	store := fixedStore()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := store.Decode(c.in)
			if err != nil {
				t.Fatalf("Decode() err = %v, want nil", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Decode() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDecodeRejectsTamperedSessionData(t *testing.T) {
	store := fixedStore()
	valid := "eyJfYXV0aF91c2VyX2lkIjoiMSJ9:1r31eq:901W1QvvDfVyPTJvUtlSLvIV3In2WsSSt0PqU6EW4Go"

	forged := "eyJfYXV0aF91c2VyX2lkIjoiOSJ9:1r31eq:901W1QvvDfVyPTJvUtlSLvIV3In2WsSSt0PqU6EW4Go"
	if _, err := store.Decode(forged); !errors.Is(err, ErrBadSignature) {
		t.Errorf("Decode(forged payload) err = %v, want ErrBadSignature", err)
	}
	if _, err := store.Decode(valid); err != nil {
		t.Errorf("Decode(valid) err = %v, want nil", err)
	}
}

func TestDecodeRejectsOtherSecretKey(t *testing.T) {
	other := New("another-secret")
	valid := "eyJfYXV0aF91c2VyX2lkIjoiMSJ9:1r31eq:901W1QvvDfVyPTJvUtlSLvIV3In2WsSSt0PqU6EW4Go"

	if _, err := other.Decode(valid); !errors.Is(err, ErrBadSignature) {
		t.Errorf("Decode(other key) err = %v, want ErrBadSignature", err)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	store := fixedStore()
	cases := []map[string]any{
		{},
		{AuthUserID: "1"},
		{AuthUserID: "42", AuthUserBackend: "django.contrib.auth.backends.ModelBackend", AuthUserHash: "deadbeef"},
		{"中文键": "中文值"},
		{"big": strings.Repeat("y", 5000)},
	}
	for _, data := range cases {
		encoded, err := store.Encode(data)
		if err != nil {
			t.Fatalf("Encode(%v) err = %v, want nil", data, err)
		}
		got, err := store.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(Encode(%v)) err = %v, want nil", data, err)
		}
		if !reflect.DeepEqual(got, data) {
			t.Errorf("Decode(Encode(%v)) = %v, want %v", data, got, data)
		}
	}
}

func TestEncodeCompressesLargePayload(t *testing.T) {
	store := fixedStore()

	small, err := store.Encode(map[string]any{AuthUserID: "1"})
	if err != nil {
		t.Fatalf("Encode(small) err = %v, want nil", err)
	}
	if strings.HasPrefix(small, compressedPrefix) {
		t.Errorf("Encode(small) = %q, want no %q prefix", small, compressedPrefix)
	}

	large, err := store.Encode(map[string]any{"big": strings.Repeat("y", 500)})
	if err != nil {
		t.Fatalf("Encode(large) err = %v, want nil", err)
	}
	if !strings.HasPrefix(large, compressedPrefix) {
		t.Errorf("Encode(large) = %q, want a %q prefix", large, compressedPrefix)
	}
}

func TestUserID(t *testing.T) {
	if got, ok := UserID(map[string]any{AuthUserID: "7"}); !ok || got != "7" {
		t.Errorf("UserID() = %q, %t, want %q, true", got, ok, "7")
	}
	if _, ok := UserID(map[string]any{}); ok {
		t.Error("UserID(empty session) ok = true, want false")
	}
	if _, ok := UserID(map[string]any{AuthUserID: ""}); ok {
		t.Error("UserID(blank id) ok = true, want false")
	}
	if _, ok := UserID(map[string]any{AuthUserID: 7}); ok {
		t.Error("UserID(int id) ok = true, want false")
	}
}

func TestNewKey(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := NewKey()
		if err != nil {
			t.Fatalf("NewKey() err = %v, want nil", err)
		}
		if len(key) != KeyLength {
			t.Fatalf("len(NewKey()) = %d, want %d", len(key), KeyLength)
		}
		for _, r := range key {
			if !strings.ContainsRune(keyAlphabet, r) {
				t.Fatalf("NewKey() = %q, want only characters from %q", key, keyAlphabet)
			}
		}
		if seen[key] {
			t.Fatalf("NewKey() = %q, want a value not already returned", key)
		}
		seen[key] = true
	}
}

func TestEncodePayloadIsASCII(t *testing.T) {
	store := fixedStore()

	for _, data := range []map[string]any{
		{"中文键": "中文值"},
		{AuthUserID: "1", "name": "Ünïcödé"},
		{"emoji": "\U0001f600"},
	} {
		encoded, err := store.Encode(data)
		if err != nil {
			t.Fatalf("Encode(%v) err = %v, want nil", data, err)
		}
		for i := 0; i < len(encoded); i++ {
			if encoded[i] >= 0x80 {
				t.Errorf("Encode(%v) = %q, want ASCII only", data, encoded)
				break
			}
		}
		got, err := store.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(Encode(%v)) err = %v, want nil", data, err)
		}
		if !reflect.DeepEqual(got, data) {
			t.Errorf("Decode(Encode(%v)) = %v, want %v", data, got, data)
		}
	}
}

func TestEscapeNonASCII(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"a":"b"}`, `{"a":"b"}`},
		{"{\"a\":\"中\"}", "{\"a\":\"\\u4e2d\"}"},
		{"{\"a\":\"\U0001f600\"}", "{\"a\":\"\\ud83d\\ude00\"}"},
	}
	for _, c := range cases {
		if got := string(escapeNonASCII([]byte(c.in))); got != c.want {
			t.Errorf("escapeNonASCII(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
