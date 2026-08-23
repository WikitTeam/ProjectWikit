package media

import "testing"

func TestSplitExt(t *testing.T) {
	tests := []struct{ in, base, ext string }{
		{"a.pdf", "a", ".pdf"},
		{"a.txt.gz", "a.txt", ".gz"},
		{"archive", "archive", ""},
		{".gitkeep", ".gitkeep", ""},
		{"..hidden", "..hidden", ""},
		{"dir.d/file", "dir.d/file", ""},
		{"dir.d/file.txt", "dir.d/file", ".txt"},
		{"a.", "a", "."},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			base, ext := splitExt(tt.in)
			if base != tt.base || ext != tt.ext {
				t.Errorf("splitExt(%q) = %q, %q, want %q, %q", tt.in, base, ext, tt.base, tt.ext)
			}
		})
	}
}

func TestGuessType(t *testing.T) {
	tests := []struct{ name, mime, encoding string }{
		{"a.pdf", "application/pdf", ""},
		{"a.PDF", "application/pdf", ""},
		{"a.txt", "text/plain", ""},
		{"a.txt.gz", "text/plain", "gzip"},
		{"a.tgz", "application/x-tar", "gzip"},
		{"a.svgz", "image/svg+xml", "gzip"},
		{"manage.py", "text/x-python", ""},
		{"a.mp4", "video/mp4", ""},
		{"a.bin", "application/octet-stream", ""},
		{"a.unknownext", "", ""},
		{"noextension", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime, encoding := guessType(tt.name)
			if mime != tt.mime || encoding != tt.encoding {
				t.Errorf("guessType(%q) = %q, %q, want %q, %q", tt.name, mime, encoding, tt.mime, tt.encoding)
			}
		})
	}
}

func TestMatchMime(t *testing.T) {
	tests := []struct {
		mime1, mime2 string
		want         bool
	}{
		{"audio/mpeg", "audio/*", true},
		{"video/mp4", "video/*", true},
		{"application/pdf", "application/pdf", true},
		{"application/zip", "application/octet-stream", false},
		{"audio/mpeg", "video/*", false},
		{"text/html; charset=utf-8", "text/html", false},
		{"anything", "audio/*", false},
	}
	for _, tt := range tests {
		t.Run(tt.mime1+" vs "+tt.mime2, func(t *testing.T) {
			if got := matchMime(tt.mime1, tt.mime2); got != tt.want {
				t.Errorf("matchMime(%q, %q) = %v, want %v", tt.mime1, tt.mime2, got, tt.want)
			}
		})
	}
}

func TestChunkSizeFor(t *testing.T) {
	tests := []struct {
		mime string
		want int64
	}{
		{"audio/mpeg", 2097152},
		{"video/mp4", 4194304},
		{"application/octet-stream", 4194304},
		{"application/zip", 8388608},
		{"application/gzip", 8388608},
		{"application/x-tar", 8388608},
		{"application/pdf", 1048576},
		// The image entry is disabled upstream.
		{"image/png", 0},
		{"text/plain", 0},
		{defaultHTMLMime, 0},
	}
	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			if got := chunkSizeFor(tt.mime); got != tt.want {
				t.Errorf("chunkSizeFor(%q) = %d, want %d", tt.mime, got, tt.want)
			}
		})
	}
}

func TestDisposition(t *testing.T) {
	tests := []struct{ name, want string }{
		{"a.txt", `inline; filename="a.txt"`},
		{`quote".txt`, `inline; filename="quote\".txt"`},
		{`back\slash.txt`, `inline; filename="back\\slash.txt"`},
		{"报告.pdf", "inline; filename*=utf-8''%E6%8A%A5%E5%91%8A.pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := disposition(tt.name); got != tt.want {
				t.Errorf("disposition(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
