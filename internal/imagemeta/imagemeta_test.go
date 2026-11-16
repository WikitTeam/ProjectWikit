package imagemeta

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func sample() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for x := 0; x < 8; x++ {
		img.Set(x, 1, color.RGBA{200, 10, 10, 255})
	}
	return img
}

func exifSegment(orientation uint16, extra string) []byte {
	tiff := []byte{'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00}
	entry := make([]byte, 12)
	binary.LittleEndian.PutUint16(entry[0:], 0x0112)
	binary.LittleEndian.PutUint16(entry[2:], 3)
	binary.LittleEndian.PutUint32(entry[4:], 1)
	binary.LittleEndian.PutUint16(entry[8:], orientation)
	tiff = append(tiff, entry...)
	tiff = append(tiff, 0, 0, 0, 0)
	tiff = append(tiff, extra...)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func segment(marker byte, payload string) []byte {
	seg := []byte{0xFF, marker, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func jpegWith(extra ...[]byte) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, sample(), nil); err != nil {
		panic(err)
	}
	raw := buf.Bytes()
	out := append([]byte{}, raw[:2]...)
	for _, e := range extra {
		out = append(out, e...)
	}
	return append(out, raw[2:]...)
}

func TestStripJPEGDropsMetadata(t *testing.T) {
	in := jpegWith(exifSegment(1, "GPS 31.2N 121.5E"), segment(0xE1, "http://ns.adobe.com/xap/1.0/\x00<x:xmpmeta/>"),
		segment(0xFE, "taken by someone"), segment(0xED, "Photoshop 3.0\x00"))
	out, changed := Strip(in)
	if !changed {
		t.Fatalf("Strip() changed = false, want true")
	}
	for _, gone := range []string{"GPS", "xmpmeta", "taken by", "Photoshop"} {
		if bytes.Contains(out, []byte(gone)) {
			t.Errorf("Strip() kept %q", gone)
		}
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("jpeg.Decode(Strip()) err = %v, want nil", err)
	}
	if got := Orientation(out); got != 1 {
		t.Errorf("Orientation(Strip()) = %d, want 1", got)
	}
}

func TestStripJPEGKeepsOrientation(t *testing.T) {
	in := jpegWith(exifSegment(6, "Canon EOS"))
	if got := Orientation(in); got != 6 {
		t.Fatalf("Orientation(in) = %d, want 6", got)
	}
	out, changed := Strip(in)
	if !changed {
		t.Fatalf("Strip() changed = false, want true")
	}
	if bytes.Contains(out, []byte("Canon")) {
		t.Errorf("Strip() kept the camera")
	}
	if got := Orientation(out); got != 6 {
		t.Errorf("Orientation(Strip()) = %d, want 6", got)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("jpeg.Decode(Strip()) err = %v, want nil", err)
	}
}

func TestStripJPEGWithoutMetadata(t *testing.T) {
	in := jpegWith()
	if _, changed := Strip(in); changed {
		t.Errorf("Strip() changed = true, want false")
	}
}

func pngChunk(kind, data string) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	copy(out[4:], kind)
	out = append(out, data...)
	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, crc32.ChecksumIEEE(append([]byte(kind), data...)))
	return append(out, crc...)
}

func TestStripPNGDropsText(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	in := append(append(append([]byte{}, raw[:33]...), pngChunk("tEXt", "Author\x00someone")...), raw[33:]...)
	out, changed := Strip(in)
	if !changed {
		t.Fatalf("Strip() changed = false, want true")
	}
	if bytes.Contains(out, []byte("someone")) {
		t.Errorf("Strip() kept the text chunk")
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("png.Decode(Strip()) err = %v, want nil", err)
	}
}

func riffChunk(kind string, data []byte) []byte {
	out := make([]byte, 8)
	copy(out, kind)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(data)))
	out = append(out, data...)
	if len(data)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func TestStripWebPDropsExif(t *testing.T) {
	vp8x := make([]byte, 10)
	vp8x[0] = 0x08 | 0x04 | 0x10
	body := append(riffChunk("VP8X", vp8x), riffChunk("VP8L", []byte{1, 2, 3})...)
	body = append(body, riffChunk("EXIF", []byte("GPS here"))...)
	body = append(body, riffChunk("XMP ", []byte("<x/>"))...)
	in := append([]byte("RIFF\x00\x00\x00\x00WEBP"), body...)
	binary.LittleEndian.PutUint32(in[4:], uint32(len(in)-8))

	out, changed := Strip(in)
	if !changed {
		t.Fatalf("Strip() changed = false, want true")
	}
	if bytes.Contains(out, []byte("GPS")) || bytes.Contains(out, []byte("<x/>")) {
		t.Errorf("Strip() kept metadata")
	}
	if got := int(binary.LittleEndian.Uint32(out[4:])); got != len(out)-8 {
		t.Errorf("Strip() RIFF size = %d, want %d", got, len(out)-8)
	}
	if got := out[20]; got != 0x10 {
		t.Errorf("Strip() VP8X flags = %#x, want 0x10", got)
	}
}
