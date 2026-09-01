package thumb

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestFitMatchesTheSizesWikidotServes(t *testing.T) {
	const w, h = 1072, 876
	cases := []struct {
		size  string
		wantW int
		wantH int
	}{
		{"thumbnail", 100, 82},
		{"small", 240, 196},
		{"medium", 500, 409},
	}
	for _, c := range cases {
		s, ok := Lookup(c.size)
		if !ok {
			t.Fatalf("Lookup(%q) = _, false, want true", c.size)
		}
		gotW, gotH := fit(w, h, s.Max)
		if gotW != c.wantW {
			t.Errorf("fit(%d, %d, %d) width = %d, want %d", w, h, s.Max, gotW, c.wantW)
		}
		if gotH != c.wantH {
			t.Errorf("fit(%d, %d, %d) height = %d, want %d", w, h, s.Max, gotH, c.wantH)
		}
	}
}

func TestFitLeavesASmallImageAlone(t *testing.T) {
	gotW, gotH := fit(40, 30, 500)
	if gotW != 40 || gotH != 30 {
		t.Errorf("fit(40, 30, 500) = %d, %d, want 40, 30", gotW, gotH)
	}
}

func TestLookupRejectsTheSizeWikidotRejects(t *testing.T) {
	if _, ok := Lookup("large"); ok {
		t.Error("Lookup(\"large\") = _, true, want false")
	}
}

func TestLookupIgnoresCase(t *testing.T) {
	if _, ok := Lookup("Medium"); !ok {
		t.Error("Lookup(\"Medium\") = _, false, want true")
	}
}

func sourceImage(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("jpeg.Encode() err = %v, want nil", err)
	}
	return buf.Bytes()
}

func TestGenerateScalesToTheLongSide(t *testing.T) {
	src := sourceImage(t, 1072, 876)
	s, _ := Lookup("medium")

	out, err := Generate(bytes.NewReader(src), s)
	if err != nil {
		t.Fatalf("Generate() err = %v, want nil", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("jpeg.DecodeConfig() err = %v, want nil", err)
	}
	if cfg.Width != 500 || cfg.Height != 409 {
		t.Errorf("Generate(medium) = %dx%d, want 500x409", cfg.Width, cfg.Height)
	}
}

func TestGenerateCropsASquare(t *testing.T) {
	src := sourceImage(t, 1072, 876)
	s, _ := Lookup("square")

	out, err := Generate(bytes.NewReader(src), s)
	if err != nil {
		t.Fatalf("Generate() err = %v, want nil", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("jpeg.DecodeConfig() err = %v, want nil", err)
	}
	if cfg.Width != 75 || cfg.Height != 75 {
		t.Errorf("Generate(square) = %dx%d, want 75x75", cfg.Width, cfg.Height)
	}
}

func TestGenerateRejectsWhatIsNotAnImage(t *testing.T) {
	s, _ := Lookup("medium")

	if _, err := Generate(bytes.NewReader([]byte("not an image")), s); err == nil {
		t.Error("Generate(text) err = nil, want an error")
	}
}
