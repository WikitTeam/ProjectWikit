// Package thumb makes the scaled copies a gallery links to.
package thumb

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strings"

	_ "image/gif"
	_ "image/png"
)

type Size struct {
	Name   string
	Max    int
	Square bool
}

var sizes = []Size{
	{Name: "square", Max: 75, Square: true},
	{Name: "thumbnail", Max: 100},
	{Name: "small", Max: 240},
	{Name: "medium", Max: 500},
}

func Lookup(name string) (Size, bool) {
	for _, s := range sizes {
		if strings.EqualFold(name, s.Name) {
			return s, true
		}
	}
	return Size{}, false
}

func Generate(r io.Reader, s Size) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	if s.Square {
		src = centreSquare(src)
	}
	w, h := fit(src.Bounds().Dx(), src.Bounds().Dy(), s.Max)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scale(src, w, h), &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// An image already smaller than the box is left alone, because stretching it
// costs bytes and shows nothing that was not there.
func fit(w, h, max int) (int, int) {
	long := w
	if h > long {
		long = h
	}
	if long <= max || long == 0 {
		return w, h
	}
	return round(w*max, long), round(h*max, long)
}

func round(n, d int) int {
	v := (n + d/2) / d
	if v < 1 {
		return 1
	}
	return v
}

func centreSquare(src image.Image) image.Image {
	b := src.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	r := image.Rect(0, 0, side, side).Add(image.Pt(
		b.Min.X+(b.Dx()-side)/2,
		b.Min.Y+(b.Dy()-side)/2,
	))
	if sub, ok := src.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(r)
	}
	return src
}

// Averaging over the source pixels a destination pixel covers is what keeps a
// downscale from aliasing, and it needs nothing the standard library lacks.
func scale(src image.Image, w, h int) image.Image {
	b := src.Bounds()
	if w == b.Dx() && h == b.Dy() {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := span(b.Min.Y, b.Dy(), y, h)
		for x := 0; x < w; x++ {
			x0, x1 := span(b.Min.X, b.Dx(), x, w)
			var sr, sg, sb, sa, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					r, g, bl, a := src.At(xx, yy).RGBA()
					sr, sg, sb, sa = sr+uint64(r), sg+uint64(g), sb+uint64(bl), sa+uint64(a)
					n++
				}
			}
			dst.Set(x, y, color.RGBA64{
				R: uint16(sr / n), G: uint16(sg / n), B: uint16(sb / n), A: uint16(sa / n),
			})
		}
	}
	return dst
}

func span(min, length, i, out int) (int, int) {
	a := min + i*length/out
	b := min + (i+1)*length/out
	if b <= a {
		b = a + 1
	}
	return a, b
}
