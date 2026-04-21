#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    exit 1
  fi
}

default_version_label() {
  if git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$REPO_ROOT" describe --tags --always --dirty 2>/dev/null || echo "dev"
    return
  fi
  echo "dev"
}

require_tool go
require_tool lipo
require_tool pkgbuild
require_tool xattr

VERSION_LABEL="${VERSION:-$(default_version_label)}"
PKG_VERSION="${PKG_VERSION:-$(printf '%s' "$VERSION_LABEL" | sed -E 's/^v//; s/[^0-9.].*$//')}"
PKG_VERSION="${PKG_VERSION:-0.0.0}"
PKG_IDENTIFIER="${PKG_IDENTIFIER:-io.dixa.cli}"
OUTPUT_DIR="${OUTPUT_DIR:-$REPO_ROOT/dist}"
APP_SIGNING_IDENTITY="${APP_SIGNING_IDENTITY:-}"
PKG_SIGNING_IDENTITY="${PKG_SIGNING_IDENTITY:-}"
SAFE_VERSION_LABEL="$(printf '%s' "$VERSION_LABEL" | tr '/[:space:]' '-')"
OUTPUT_PKG="$OUTPUT_DIR/dixa-${SAFE_VERSION_LABEL}-macos-universal.pkg"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dixa-pkg.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

build_binary() {
  local arch="$1"
  local output="$2"

  GOOS=darwin GOARCH="$arch" go build \
    -trimpath \
    -ldflags="-s -w -X github.com/Dixa-public/dixa-cli-public/internal/cli.version=$VERSION_LABEL" \
    -o "$output" \
    ./cmd/dixa
}

ARM64_BINARY="$TMP_DIR/dixa-arm64"
AMD64_BINARY="$TMP_DIR/dixa-amd64"
UNIVERSAL_BINARY="$TMP_DIR/dixa"
PKG_ROOT="$TMP_DIR/pkgroot"
PKG_SCRIPTS="$TMP_DIR/pkgscripts"

build_binary arm64 "$ARM64_BINARY"
build_binary amd64 "$AMD64_BINARY"
lipo -create -output "$UNIVERSAL_BINARY" "$ARM64_BINARY" "$AMD64_BINARY"
chmod 0755 "$UNIVERSAL_BINARY"
xattr -cr "$UNIVERSAL_BINARY"

if [[ -n "$APP_SIGNING_IDENTITY" ]]; then
  require_tool codesign
  codesign --force --sign "$APP_SIGNING_IDENTITY" "$UNIVERSAL_BINARY"
fi

mkdir -p "$PKG_ROOT/usr/local/bin"
install -m 0755 "$UNIVERSAL_BINARY" "$PKG_ROOT/usr/local/bin/dixa"
xattr -cr "$PKG_ROOT"
mkdir -p "$OUTPUT_DIR"
mkdir -p "$PKG_SCRIPTS"

cat >"$PKG_SCRIPTS/postinstall" <<'EOF'
#!/bin/sh
set -e

target_volume="${3:-/}"

rm -f \
  "$target_volume/usr/local/bin/._dixa" \
  "$target_volume/usr/local/._bin" \
  "$target_volume/usr/._local" \
  "$target_volume/._usr"
EOF
chmod 0755 "$PKG_SCRIPTS/postinstall"

PKGBUILD_ARGS=(
  --root "$PKG_ROOT"
  --scripts "$PKG_SCRIPTS"
  --identifier "$PKG_IDENTIFIER"
  --version "$PKG_VERSION"
  --install-location /
)

if [[ -n "$PKG_SIGNING_IDENTITY" ]]; then
  PKGBUILD_ARGS+=(--sign "$PKG_SIGNING_IDENTITY")
fi

PKGBUILD_ARGS+=("$OUTPUT_PKG")
COPYFILE_DISABLE=1 pkgbuild "${PKGBUILD_ARGS[@]}"

echo
echo "Created installer: $OUTPUT_PKG"
echo "Package version:   $PKG_VERSION"
echo "Install target:    /usr/local/bin/dixa"
echo "SHA256:"
shasum -a 256 "$OUTPUT_PKG"

if [[ -z "$APP_SIGNING_IDENTITY" || -z "$PKG_SIGNING_IDENTITY" ]]; then
  echo
  echo "Note: the installer is not fully signed."
  echo "For the smoothest internal rollout, sign the binary and package before distribution."
fi
