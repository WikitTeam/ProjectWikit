//go:build bundle

package pgbundle

import _ "embed"

var (
	//go:embed archive/postgresql-darwin-arm64.tar.zst
	archive []byte

	//go:embed archive/postgresql-darwin-arm64.sha256
	archiveSum string
)
