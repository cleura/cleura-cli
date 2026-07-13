#!/bin/sh
# Install the cleura CLI from GitHub Releases (Linux and macOS).
#
#   curl -fsSL https://raw.githubusercontent.com/cleura/cleura-cli/main/install.sh | sh
#
# Environment overrides:
#   CLEURA_VERSION   version to install, e.g. v0.7.0 (default: latest release)
#   BINDIR           install directory (default: /usr/local/bin)
#
# Downloads the release archive for your OS/architecture, verifies it against the
# release checksums, and installs the `cleura` binary. Windows users: download the
# .zip from the Releases page instead.
set -eu

REPO="cleura/cleura-cli"
BINDIR="${BINDIR:-/usr/local/bin}"
VERSION="${CLEURA_VERSION:-latest}"

err() { echo "install: $*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

have curl || err "curl is required"
have tar || err "tar is required"

# --- detect platform ---
os=$(uname -s)
case "$os" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) err "unsupported OS '$os'; download a binary from https://github.com/$REPO/releases" ;;
esac
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) err "unsupported architecture '$arch'" ;;
esac

# --- resolve the version to install ---
if [ "$VERSION" = latest ]; then
	VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
		grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name" *: *"([^"]+)".*/\1/')
	[ -n "$VERSION" ] || err "could not determine the latest version (no releases yet?); set CLEURA_VERSION"
fi
num=${VERSION#v} # goreleaser archive names drop the leading v

archive="cleura_${num}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading cleura $VERSION ($os/$arch)..."
curl -fsSL "$base/$archive" -o "$tmp/$archive" || err "download failed: $base/$archive"

# --- verify checksum ---
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || err "could not download checksums.txt"
want=$(grep " ${archive}\$" "$tmp/checksums.txt" | awk '{print $1}')
[ -n "$want" ] || err "no checksum listed for $archive"
if have sha256sum; then
	got=$(sha256sum "$tmp/$archive" | awk '{print $1}')
else
	got=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
fi
[ "$want" = "$got" ] || err "checksum mismatch for $archive (expected $want, got $got)"
echo "Checksum verified."

tar -xzf "$tmp/$archive" -C "$tmp" cleura || err "failed to extract cleura from $archive"

# --- install ---
if [ -w "$BINDIR" ]; then
	install -m 0755 "$tmp/cleura" "$BINDIR/cleura"
elif have sudo; then
	echo "Installing to $BINDIR (requires sudo)..."
	sudo install -m 0755 "$tmp/cleura" "$BINDIR/cleura"
else
	err "$BINDIR is not writable; re-run with sudo, or set BINDIR (e.g. BINDIR=\$HOME/.local/bin)"
fi

echo "Installed $("$BINDIR/cleura" version 2>/dev/null || echo "cleura to $BINDIR")."
