package userpage

import (
	"image"
	"image/color"
	"testing"
)

func TestOrientTurnsThePixels(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	red := color.NRGBA{255, 0, 0, 255}
	src.Set(0, 0, red)

	cases := []struct {
		o          int
		w, h, x, y int
	}{
		{1, 4, 2, 0, 0},
		{3, 4, 2, 3, 1},
		{6, 2, 4, 1, 0},
		{8, 2, 4, 0, 3},
	}
	for _, c := range cases {
		got := orient(src, c.o)
		if b := got.Bounds(); b.Dx() != c.w || b.Dy() != c.h {
			t.Errorf("orient(%d) size = %dx%d, want %dx%d", c.o, b.Dx(), b.Dy(), c.w, c.h)
			continue
		}
		if got.At(c.x, c.y) != color.Color(red) {
			r, _, _, _ := got.At(c.x, c.y).RGBA()
			t.Errorf("orient(%d) at (%d,%d) red = %d, want 65535", c.o, c.x, c.y, r)
		}
	}
}
