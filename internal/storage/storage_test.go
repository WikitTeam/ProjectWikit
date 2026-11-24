package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
)

func contract(t *testing.T, s Store) {
	ctx := context.Background()

	t.Run("PutOpen", func(t *testing.T) {
		if err := s.Put(ctx, "media/a/one.txt", strings.NewReader("hello"), 5); err != nil {
			t.Fatalf("Put() err = %v, want nil", err)
		}
		got, err := ReadAll(ctx, s, "media/a/one.txt")
		if err != nil {
			t.Fatalf("ReadAll() err = %v, want nil", err)
		}
		if string(got) != "hello" {
			t.Errorf("ReadAll() = %q, want %q", got, "hello")
		}
	})

	t.Run("Seek", func(t *testing.T) {
		f, err := s.Open(ctx, "media/a/one.txt")
		if err != nil {
			t.Fatalf("Open() err = %v, want nil", err)
		}
		defer f.Close()
		if f.Info().Size != 5 {
			t.Errorf("Info().Size = %d, want 5", f.Info().Size)
		}
		if _, err := f.Seek(2, io.SeekStart); err != nil {
			t.Fatalf("Seek() err = %v, want nil", err)
		}
		rest, _ := io.ReadAll(f)
		if string(rest) != "llo" {
			t.Errorf("read after Seek(2) = %q, want %q", rest, "llo")
		}
	})

	t.Run("Overwrite", func(t *testing.T) {
		if err := s.Put(ctx, "media/a/one.txt", bytes.NewReader([]byte("bye")), 3); err != nil {
			t.Fatalf("Put() err = %v, want nil", err)
		}
		info, err := s.Stat(ctx, "media/a/one.txt")
		if err != nil {
			t.Fatalf("Stat() err = %v, want nil", err)
		}
		if info.Size != 3 {
			t.Errorf("Stat().Size = %d, want 3", info.Size)
		}
	})

	t.Run("Missing", func(t *testing.T) {
		if _, err := s.Open(ctx, "media/none"); !errors.Is(err, ErrNotExist) {
			t.Errorf("Open(missing) err = %v, want ErrNotExist", err)
		}
		if err := s.Remove(ctx, "media/none"); err != nil {
			t.Errorf("Remove(missing) err = %v, want nil", err)
		}
	})

	t.Run("Walk", func(t *testing.T) {
		if err := s.Put(ctx, "media/b/two.txt", strings.NewReader("2"), 1); err != nil {
			t.Fatal(err)
		}
		if err := s.Put(ctx, "theme/x.css", strings.NewReader("3"), 1); err != nil {
			t.Fatal(err)
		}
		var keys []string
		if err := s.Walk(ctx, "media", func(key string, _ Info) error {
			keys = append(keys, key)
			return nil
		}); err != nil {
			t.Fatalf("Walk() err = %v, want nil", err)
		}
		slices.Sort(keys)
		want := []string{"media/a/one.txt", "media/b/two.txt"}
		if !slices.Equal(keys, want) {
			t.Errorf("Walk(media) = %v, want %v", keys, want)
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		if err := s.RemoveAll(ctx, "media/a"); err != nil {
			t.Fatalf("RemoveAll() err = %v, want nil", err)
		}
		if Exists(ctx, s, "media/a/one.txt") {
			t.Errorf("Exists(media/a/one.txt) = true after RemoveAll, want false")
		}
		if !Exists(ctx, s, "media/b/two.txt") {
			t.Errorf("Exists(media/b/two.txt) = false, want true")
		}
	})

	t.Run("Remove", func(t *testing.T) {
		if err := s.Remove(ctx, "theme/x.css"); err != nil {
			t.Fatalf("Remove() err = %v, want nil", err)
		}
		if Exists(ctx, s, "theme/x.css") {
			t.Errorf("Exists(theme/x.css) = true after Remove, want false")
		}
	})

	t.Run("BadKey", func(t *testing.T) {
		for _, key := range []string{"../x", "/x", "a/../../x", ""} {
			if err := s.Put(ctx, key, strings.NewReader("x"), 1); !errors.Is(err, ErrBadKey) {
				t.Errorf("Put(%q) err = %v, want ErrBadKey", key, err)
			}
		}
	})
}

func TestLocal(t *testing.T) {
	contract(t, NewLocal(t.TempDir()))
}
