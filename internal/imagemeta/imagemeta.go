// Package imagemeta takes the camera and location data out of uploaded images.
package imagemeta

import (
	"bytes"
	"encoding/binary"
)

// A format it does not know comes back untouched rather than refused, since
// the upload itself is still wanted.
func Strip(data []byte) ([]byte, bool) {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8}):
		return stripJPEG(data)
	case bytes.HasPrefix(data, pngSignature):
		return stripPNG(data)
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return stripWebP(data)
	}
	return data, false
}

func Orientation(data []byte) int {
	if !bytes.HasPrefix(data, []byte{0xFF, 0xD8}) {
		return 1
	}
	o := 1
	walkJPEG(data, func(marker byte, payload []byte, _, _ int) {
		if marker == 0xE1 && bytes.HasPrefix(payload, exifHeader) {
			if v := exifOrientation(payload[len(exifHeader):]); v != 0 {
				o = v
			}
		}
	})
	return o
}

var (
	exifHeader   = []byte("Exif\x00\x00")
	xmpHeader    = []byte("http://ns.adobe.com/xap/1.0/\x00")
	xmpExtended  = []byte("http://ns.adobe.com/xmp/extension/\x00")
	pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
)

// Only the segments ahead of the scan are walked, since what follows is image
// data that may itself contain 0xFF bytes. A file cut short gives -1.
func walkJPEG(data []byte, visit func(marker byte, payload []byte, from, to int)) int {
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return -1
		}
		marker := data[i+1]
		if marker == 0xFF {
			i++
			continue
		}
		if marker == 0xD9 || marker == 0xDA {
			return i
		}
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		length := int(binary.BigEndian.Uint16(data[i+2:]))
		if length < 2 || i+2+length > len(data) {
			return -1
		}
		visit(marker, data[i+4:i+2+length], i, i+2+length)
		i += 2 + length
	}
	return -1
}

// The orientation is the one piece of EXIF a browser acts on, so a photo
// taken sideways keeps it in a block that says nothing else.
func stripJPEG(data []byte) ([]byte, bool) {
	orientation := 1
	dropped := false
	var kept [][2]int
	start := walkJPEG(data, func(marker byte, payload []byte, from, to int) {
		segment := [2]int{from, to}
		switch {
		case marker == 0xE1 && bytes.HasPrefix(payload, exifHeader):
			if v := exifOrientation(payload[len(exifHeader):]); v != 0 {
				orientation = v
			}
			dropped = true
		case marker == 0xE1 && (bytes.HasPrefix(payload, xmpHeader) || bytes.HasPrefix(payload, xmpExtended)):
			dropped = true
		case marker == 0xED, marker == 0xFE:
			dropped = true
		default:
			kept = append(kept, segment)
		}
	})
	if start < 0 || !dropped {
		return data, false
	}

	var out bytes.Buffer
	out.Grow(len(data))
	out.Write(data[:2])
	for n, seg := range kept {
		out.Write(data[seg[0]:seg[1]])
		if n == 0 && orientation != 1 {
			out.Write(orientationSegment(orientation))
		}
	}
	if len(kept) == 0 && orientation != 1 {
		out.Write(orientationSegment(orientation))
	}
	out.Write(data[start:])
	return out.Bytes(), true
}

func orientationSegment(o int) []byte {
	tiff := []byte{
		'M', 'M', 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08,
		0x00, 0x01,
		0x01, 0x12, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, byte(o), 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	payload := append(append([]byte{}, exifHeader...), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[ifd:]))
	for n := 0; n < count; n++ {
		entry := ifd + 2 + n*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:]) == 0x0112 && order.Uint16(tiff[entry+2:]) == 3 {
			if v := int(order.Uint16(tiff[entry+8:])); v >= 1 && v <= 8 {
				return v
			}
		}
	}
	return 0
}

var pngDropped = map[string]bool{"eXIf": true, "tEXt": true, "zTXt": true, "iTXt": true, "tIME": true}

func stripPNG(data []byte) ([]byte, bool) {
	var out bytes.Buffer
	out.Write(pngSignature)
	dropped := false
	i := len(pngSignature)
	for i+12 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[i:]))
		end := i + 12 + length
		if length < 0 || end > len(data) {
			return data, false
		}
		kind := string(data[i+4 : i+8])
		if pngDropped[kind] {
			dropped = true
		} else {
			out.Write(data[i:end])
		}
		i = end
		if kind == "IEND" {
			break
		}
	}
	if !dropped {
		return data, false
	}
	out.Write(data[i:])
	return out.Bytes(), true
}

func stripWebP(data []byte) ([]byte, bool) {
	var body bytes.Buffer
	dropped := false
	flagsAt := -1
	i := 12
	for i+8 <= len(data) {
		kind := string(data[i : i+4])
		size := int(binary.LittleEndian.Uint32(data[i+4:]))
		end := i + 8 + size + size%2
		if size < 0 || end > len(data) {
			return data, false
		}
		switch kind {
		case "EXIF", "XMP ":
			dropped = true
		default:
			if kind == "VP8X" && size >= 1 {
				flagsAt = body.Len() + 8
			}
			body.Write(data[i:end])
		}
		i = end
	}
	if !dropped {
		return data, false
	}
	out := body.Bytes()
	if flagsAt >= 0 {
		out[flagsAt] &^= 0x08 | 0x04
	}
	header := make([]byte, 12)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(4+len(out)))
	copy(header[8:], "WEBP")
	return append(header, out...), true
}
