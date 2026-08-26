package static

import (
	"testing"
	"testing/fstest"
)

func TestAssetsURLCarriesContentDigest(t *testing.T) {
	assets := NewAssets(fstest.MapFS{"app.js": {Data: []byte("x")}})
	want := "/-/static/app.js?v=9dd4e461"
	if got := assets.URL("app.js"); got != want {
		t.Errorf("URL(%q) = %q, want %q", "app.js", got, want)
	}
}

func TestAssetsURLChangesWithContent(t *testing.T) {
	first := NewAssets(fstest.MapFS{"app.js": {Data: []byte("a")}}).URL("app.js")
	second := NewAssets(fstest.MapFS{"app.js": {Data: []byte("b")}}).URL("app.js")
	if first == second {
		t.Errorf("URL(%q) = %q for both contents, want different", "app.js", first)
	}
}

func TestAssetsURLOfMissingFile(t *testing.T) {
	assets := NewAssets(fstest.MapFS{})
	want := "/-/static/app.js"
	if got := assets.URL("app.js"); got != want {
		t.Errorf("URL(%q) = %q, want %q", "app.js", got, want)
	}
}

func TestAssetsURLWithoutBundle(t *testing.T) {
	assets := NewAssets(nil)
	want := "/-/static/app.js"
	if got := assets.URL("app.js"); got != want {
		t.Errorf("URL(%q) = %q, want %q", "app.js", got, want)
	}
}

func TestAssetsURLKeepsFirstAnswer(t *testing.T) {
	fsys := fstest.MapFS{"app.js": {Data: []byte("x")}}
	assets := NewAssets(fsys)
	first := assets.URL("app.js")
	fsys["app.js"] = &fstest.MapFile{Data: []byte("y")}
	if got := assets.URL("app.js"); got != first {
		t.Errorf("URL(%q) = %q, want %q", "app.js", got, first)
	}
}
