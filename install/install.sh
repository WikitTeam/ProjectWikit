#!/bin/sh
# Installs a ProjectWikit release on Linux or macOS.
#
#   curl -fsSL https://github.com/WikitTeam/ProjectWikit/releases/latest/download/install.sh | sh
#   curl -fsSL <mirror>/install.sh | sh -s -- --mirror <mirror>
#
# Options:
#   --version <vX.Y.Z>  release to install; defaults to the newest
#   --dir <directory>   where pwikit and all of its data go
#   --mirror <url>      mirror to use when GitHub cannot be reached
#   --user <account>    account that owns the directory when run as root;
#                       defaults to the sudo account, or a new pwikit account
#   --no-path           do not make pwikit runnable by name
set -eu

releases="${PWIKIT_RELEASES_URL:-https://github.com/WikitTeam/ProjectWikit/releases}"
version=""
dir=""
mirror="${PWIKIT_MIRROR:-}"
owner=""
add_path=1

say() { printf '%s\n' "$*"; }
fail() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --version) [ $# -ge 2 ] || fail "--version needs a value"; version="$2"; shift 2 ;;
    --dir) [ $# -ge 2 ] || fail "--dir needs a value"; dir="$2"; shift 2 ;;
    --mirror) [ $# -ge 2 ] || fail "--mirror needs a value"; mirror="$2"; shift 2 ;;
    --user) [ $# -ge 2 ] || fail "--user needs a value"; owner="$2"; shift 2 ;;
    --no-path) add_path=0; shift ;;
    -h|--help)
      say "Usage: install.sh [--version vX.Y.Z] [--dir directory] [--mirror url] [--user account] [--no-path]"
      exit 0 ;;
    *) fail "unknown option $1" ;;
  esac
done
mirror="${mirror%/}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "this script installs on Linux and macOS; on Windows use install.ps1" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) fail "no release is built for $(uname -m)" ;;
esac

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL --connect-timeout 15 --retry 2 -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q -T 15 -O "$2" "$1"; }
else
  fail "curl or wget is needed"
fi
if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  fail "sha256sum or shasum is needed"
fi
command -v tar >/dev/null 2>&1 || fail "tar is needed"

create_owner=0
if [ "$(id -u)" -eq 0 ]; then
  if [ -z "$owner" ]; then
    if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != root ]; then
      owner="$SUDO_USER"
    else
      owner=pwikit
    fi
  fi
  [ "$owner" != root ] || fail "pwikit does not run as root; pass another account to --user"
  [ -n "$dir" ] || dir=/opt/pwikit
  if ! id "$owner" >/dev/null 2>&1; then
    [ "$owner" = pwikit ] || fail "no account named $owner"
    [ "$os" = linux ] || fail "pwikit does not run as root; create an account for it and pass it to --user"
    command -v useradd >/dev/null 2>&1 || command -v adduser >/dev/null 2>&1 ||
      fail "pwikit does not run as root, and there is no useradd to create an account for it; create one and pass it to --user"
    create_owner=1
  fi
else
  [ -z "$owner" ] || fail "--user only applies when running as root"
  [ -n "$dir" ] || dir="$HOME/pwikit"
fi

if [ -e "$dir/pwikit" ]; then
  fail "$dir already holds pwikit; update it with: $dir/pwikit update"
fi
if [ -d "$dir" ] && [ -n "$(ls -A "$dir" 2>/dev/null)" ]; then
  fail "$dir is not empty; pass --dir with a new or empty directory"
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT INT TERM

download() {
  if fetch "$releases/$1" "$2" 2>/dev/null; then
    return 0
  fi
  if [ -n "$mirror" ]; then
    case "$1" in
      latest/download/*) mirrored="$mirror/${1#latest/download/}" ;;
      download/*) mirrored="$mirror/${1#download/}" ;;
    esac
    say "GitHub could not be reached, trying $mirror"
    fetch "$mirrored" "$2" && return 0
  fi
  return 1
}

package_line() {
  awk -v want="\"$os-$arch\"" -v key="\"$1\"" '
    index($0, want) { inside = 1; next }
    inside && index($0, key) {
      sub(/^[^:]*:[ \t]*"?/, ""); sub(/"?,?[ \t\r]*$/, ""); print; exit
    }
    inside && /}/ { exit }
  ' "$work/latest.json"
}

if [ -z "$version" ]; then
  download latest/download/latest.json "$work/latest.json" || fail "could not fetch latest.json; check the network, or pass --mirror"
  version="$(awk -F'"' '/"version"/ { print $4; exit }' "$work/latest.json")"
  file="$(package_line file)"
  want="$(package_line sha256)"
  [ -n "$version" ] && [ -n "$file" ] && [ -n "$want" ] || fail "latest.json has no package for $os-$arch"
else
  case "$version" in v*) ;; *) version="v$version" ;; esac
  file="pwikit-$version-$os-$arch.tar.gz"
  download "download/$version/SHA256SUMS" "$work/SHA256SUMS" || fail "could not fetch the checksums of $version; does that release exist?"
  want="$(awk -v f="$file" '$2 == f { print $1; exit }' "$work/SHA256SUMS")"
  [ -n "$want" ] || fail "release $version has no package for $os-$arch"
fi

say "Downloading pwikit $version for $os-$arch"
download "download/$version/$file" "$work/$file" || fail "could not download $file"
got="$(sha256 "$work/$file")"
[ "$got" = "$want" ] || fail "$file has sha256 $got, want $want; the download is damaged or was altered"

mkdir -p "$work/unpacked"
tar -xzf "$work/$file" -C "$work/unpacked"
top="$work/unpacked/pwikit-$version-$os-$arch"
[ -x "$top/pwikit" ] || fail "$file does not hold pwikit"

if [ "$create_owner" -eq 1 ]; then
  shell=/bin/false
  for candidate in /usr/sbin/nologin /sbin/nologin; do
    if [ -x "$candidate" ]; then
      shell="$candidate"
      break
    fi
  done
  if command -v useradd >/dev/null 2>&1; then
    useradd --system --user-group --home-dir "$dir" --no-create-home --shell "$shell" "$owner" ||
      fail "could not create the account $owner; create one and pass it to --user"
  else
    adduser -S -D -H -h "$dir" -s "$shell" "$owner" ||
      fail "could not create the account $owner; create one and pass it to --user"
  fi
  say "Created the account $owner, which pwikit runs as"
fi

mkdir -p "$dir"
cp "$top/pwikit" "$dir/pwikit"
[ ! -f "$top/LICENSE" ] || cp "$top/LICENSE" "$dir/LICENSE"
chmod 755 "$dir/pwikit"
if [ -n "$owner" ]; then
  chown -R "$owner" "$dir"
fi
say "Installed pwikit $version into $dir"

if [ "$add_path" -eq 1 ]; then
  "$dir/pwikit" path install || say "pwikit is installed, but could not be made runnable by name; run $dir/pwikit path install later"
fi

say ""
say "Next, create the site from that directory:"
say "  cd $dir"
if [ -n "$owner" ]; then
  create='./pwikit createsite -slug main -title "My Wiki" -headline "A wiki" -domain wiki.example.org -media-domain files.example.org'
  if command -v sudo >/dev/null 2>&1; then
    say "  sudo -u $owner $create"
  else
    say "  su -s /bin/sh $owner -c '$create'"
  fi
  say "To start it at boot, run as root from that directory:"
  say "  ./pwikit service install -user $owner"
else
  say "  ./pwikit createsite -slug main -title \"My Wiki\" -headline \"A wiki\" -domain wiki.example.org -media-domain files.example.org"
fi
say "Then follow the quick start: https://github.com/WikitTeam/ProjectWikit/blob/main/docs/en/quickstart.md"
