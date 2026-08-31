package media

import "strings"

// An empty type means unknown, and the caller supplies the fallback.
func guessType(name string) (mimeType, encoding string) {
	base, ext := splitExt(name)
	for {
		mapped, ok := mimeSuffixes[strings.ToLower(ext)]
		if !ok {
			break
		}
		base, ext = splitExt(base + mapped)
	}
	if enc, ok := mimeEncodings[strings.ToLower(ext)]; ok {
		encoding = enc
		_, ext = splitExt(base)
	}
	return mimeTypes[strings.ToLower(ext)], encoding
}

// splitExt is os.path.splitext: ".gitkeep" splits to (".gitkeep", "").
func splitExt(p string) (base, ext string) {
	sep := strings.LastIndexByte(p, '/')
	dot := strings.LastIndexByte(p, '.')
	if dot > sep {
		for i := sep + 1; i < dot; i++ {
			if p[i] != '.' {
				return p[:dot], p[dot:]
			}
		}
	}
	return p, ""
}

// matchMime splits on the first slash only, so a type carrying parameters
// keeps them in the subtype and matches nothing.
func matchMime(mime1, mime2 string) bool {
	type1, subtype1, ok1 := strings.Cut(mime1, "/")
	type2, subtype2, ok2 := strings.Cut(mime2, "/")
	if !ok1 || !ok2 {
		return false
	}
	if type1 == "*" && subtype1 == "*" {
		return true
	}
	if type1 != type2 {
		return false
	}
	return subtype1 == "*" || subtype1 == subtype2 || subtype2 == "*"
}

// Order is load-bearing: the first match wins.
var rangedMime = []struct {
	mime  string
	chunk int64
}{
	{"audio/*", 2097152},
	{"video/*", 4194304},
	{"application/octet-stream", 4194304},
	{"application/zip", 8388608},
	{"application/gzip", 8388608},
	{"application/x-tar", 8388608},
	{"application/pdf", 1048576},
}

func chunkSizeFor(mimeType string) int64 {
	for _, r := range rangedMime {
		if matchMime(mimeType, r.mime) {
			return r.chunk
		}
	}
	return 0
}
