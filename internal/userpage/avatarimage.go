package userpage

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const avatarSide = 256

const avatarQuality = 85

func shrinkAvatar(data []byte, ext string) ([]byte, string) {
	if ext == ".gif" {
		return data, ext
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, ext
	}
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
