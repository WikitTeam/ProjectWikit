//go:build bundle

package pgbundle

import _ "embed"

var (
	//go:embed archive/postgresql-linux-amd64.tar.zst
	archive []byte

	//go:embed archive/postgresql-linux-amd64.sha256
	archiveSum string
)
