#!/usr/bin/env bash
set -euo pipefail

REPO="${DIXA_REPO:-Dixa-public/dixa-cli-public}"
INSTALL_DIR="${INSTALL_DIR:-}"

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    exit 1
  fi
}

latest_tag() {
  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

require_tool curl
require_tool tar

case "$(uname -s)" in
  Darwin) DIXA_OS="darwin" ;;
  *)
    echo "unsupported OS: $(uname -s)" >&2
    echo "use scripts/install.ps1 on Windows, or install manually from GitHub Releases" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  arm64) DIXA_ARCH="arm64" ;;
  x86_64|amd64) DIXA_ARCH="amd64" ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

TAG="${DIXA_VERSION:-}"
if [[ -z "$TAG" ]]; then
  TAG="$(latest_tag)"
fi
if [[ -z "$TAG" ]]; then
  echo "failed to resolve the latest release tag" >&2
  exit 1
fi

case "$TAG" in
  v*) VERSION="${TAG#v}" ;;
  *)
    VERSION="$TAG"
    TAG="v$TAG"
    ;;
esac

if [[ -z "$INSTALL_DIR" ]]; then
  if [[ -w /usr/local/bin ]] || [[ ! -e /usr/local/bin && -w /usr/local ]]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dixa-install.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE_NAME="dixa_${VERSION}_${DIXA_OS}_${DIXA_ARCH}.tar.gz"
ARCHIVE_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"

mkdir -p "$INSTALL_DIR"
curl -fsSL -o "$TMP_DIR/$ARCHIVE_NAME" "$ARCHIVE_URL"
tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"
install -m 0755 "$TMP_DIR/dixa" "$INSTALL_DIR/dixa"

echo
echo "Installed dixa ${VERSION} to ${INSTALL_DIR}/dixa"
"$INSTALL_DIR/dixa" --version

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo
    echo "Note: ${INSTALL_DIR} is not currently on PATH."
    echo "Add it to your shell profile, for example:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
