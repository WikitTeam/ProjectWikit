//go:build !bundle && !assets

package static

import "embed"

var Files embed.FS

const Embedded = false
