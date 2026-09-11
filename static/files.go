//go:build bundle

// Package static carries the frontend asset bundle inside the binary.
package static

import "embed"

//go:embed *.css *.js *.js.map fonts fontawesome images
var Files embed.FS

const Embedded = true
