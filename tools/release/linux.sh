#!/bin/sh
set -eu

if [ -z "${VERSION:-}" ]; then
  cat >&2 <<'USAGE'
Usage: VERSION=v1.0.0 sh tools/release/linux.sh

Builds the Linux release package into dist/. Run it inside a manylinux_2_28
container, whose old C library lets the package run on RHEL 8, Debian 10,
Ubuntu 20.04 and anything newer:

  docker run --rm -v "$PWD:/src" -w /src -e VERSION=v1.0.0     quay.io/pypa/manylinux_2_28_x86_64 sh tools/release/linux.sh

Build the page assets first, outside the container, with yarn build.
USAGE
  exit 2
fi
GO_VERSION="${GO_VERSION:-$(sed -n 's/^go //p' go.mod)}"

case "$(uname -m)" in
  x86_64) goarch=amd64 ;;
  aarch64) goarch=arm64 ;;
  *) echo "linux.sh: no Go build for $(uname -m)" >&2; exit 1 ;;
esac

if ! command -v go >/dev/null 2>&1; then
  full="$(curl -fsSL 'https://go.dev/dl/?mode=json&include=all' | python3 -c '
import json, sys
want = sys.argv[1]
for release in json.load(sys.stdin):
    v = release["version"]
    if release["stable"] and (v == "go" + want or v.startswith("go" + want + ".")):
        print(v)
        break
' "$GO_VERSION")"
  [ -n "$full" ] || { echo "linux.sh: no stable Go release matches $GO_VERSION" >&2; exit 1; }
  curl -fsSL "https://go.dev/dl/$full.linux-$goarch.tar.gz" | tar -xz -C /usr/local
  export PATH="/usr/local/go/bin:$PATH"
fi
if ! command -v cargo >/dev/null 2>&1; then
  curl -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal -q
  export PATH="$HOME/.cargo/bin:$PATH"
fi

git config --global --add safe.directory "$(pwd)" 2>/dev/null || true
go run ./tools/release build -version "$VERSION" -out "${OUT:-dist}" -skip-frontend
