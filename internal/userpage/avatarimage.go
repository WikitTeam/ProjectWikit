package userpage

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/WikitTeam/ProjectWikit/internal/imagemeta"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const avatarSide = 256

const avatarQuality = 85

func shrinkAvatar(data []byte, ext string) ([]byte, string) {
	if ext == ".gif" {
		return data, ext
	}
	data, _ = imagemeta.Strip(data)
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, ext
	}
	// Re-encoding drops the orientation tag, so the turn it asked for is made
	// in the pixels instead.
	src = orient(src, imagemeta.Orientation(data))
	img := src
	if b := src.Bounds(); b.Dx() > avatarSide || b.Dy() > avatarSide {
		w, h := fit(b.Dx(), b.Dy(), avatarSide)
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		img = dst
	}

	var out bytes.Buffer
	outExt := ".jpg"
	if opaque(img) {
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: avatarQuality})
	} else {
		outExt = ".png"
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, img)
	}
	if err != nil || out.Len() >= len(data) {
		return data, ext
	}
	return out.Bytes(), outExt
}

func orient(src image.Image, o int) image.Image {
	if o < 2 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if o >= 5 {
		w, h = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = b.Dx()-1-x, y
			case 3:
				dx, dy = b.Dx()-1-x, b.Dy()-1-y
			case 4:
				dx, dy = x, b.Dy()-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = b.Dy()-1-y, x
			case 7:
				dx, dy = b.Dy()-1-y, b.Dx()-1-x
			case 8:
				dx, dy = y, b.Dx()-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func fit(w, h, side int) (int, int) {
	if w >= h {
		return side, max(1, h*side/w)
	}
	return max(1, w*side/h), side
}

func opaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return false
}
