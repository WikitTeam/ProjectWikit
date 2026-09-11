//go:build !bundle

package static

import "embed"

var Files embed.FS

const Embedded = false
