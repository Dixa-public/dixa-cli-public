#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$REPO_ROOT/.release-extra}"
VERSION="${VERSION:-}"

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    exit 1
  fi
}

default_version() {
  if git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$REPO_ROOT" describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'
    return
  fi
  echo ""
}

render_skill() {
  local source="$1"
  local destination="$2"
  local version="$3"

  python3 - "$source" "$destination" "$version" <<'PY'
from pathlib import Path
import sys

source = Path(sys.argv[1]).read_text()
version = sys.argv[3]

macos_marker = 'export DIXA_VERSION="${DIXA_VERSION:-}"'
windows_marker = '$env:DIXA_VERSION = if ($env:DIXA_VERSION) { $env:DIXA_VERSION } else { "" }'

if macos_marker not in source:
    raise SystemExit(f"missing macOS version marker in {sys.argv[1]}")
if windows_marker not in source:
    raise SystemExit(f"missing Windows version marker in {sys.argv[1]}")

source = source.replace(
    macos_marker,
    f'export DIXA_VERSION="${{DIXA_VERSION:-{version}}}"',
    1,
)
source = source.replace(
    windows_marker,
    f'$env:DIXA_VERSION = if ($env:DIXA_VERSION) {{ $env:DIXA_VERSION }} else {{ "{version}" }}',
    1,
)

Path(sys.argv[2]).write_text(source)
PY
}

require_tool python3
require_tool zip

case "$OUTPUT_DIR" in
  /*) ;;
  *) OUTPUT_DIR="$REPO_ROOT/$OUTPUT_DIR" ;;
esac

VERSION="${VERSION:-$(default_version)}"
if [[ -z "$VERSION" ]]; then
  echo "VERSION is required when no git tag is available" >&2
  exit 1
fi

SAFE_VERSION="$(printf '%s' "$VERSION" | sed 's/^v//')"
TAGGED_VERSION="v${SAFE_VERSION}"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dixa-skill.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

STAGE_DIR="$TMP_DIR/dixa"
mkdir -p "$STAGE_DIR/scripts" "$OUTPUT_DIR"

render_skill "$REPO_ROOT/SKILL.md" "$STAGE_DIR/SKILL.md" "$SAFE_VERSION"
install -m 0755 "$REPO_ROOT/scripts/install.sh" "$STAGE_DIR/scripts/install.sh"
install -m 0755 "$REPO_ROOT/scripts/install.ps1" "$STAGE_DIR/scripts/install.ps1"

OUTPUT_ZIP="$OUTPUT_DIR/skill-${TAGGED_VERSION}.zip"
rm -f "$OUTPUT_ZIP"

(
  cd "$TMP_DIR"
  zip -rq "$OUTPUT_ZIP" dixa
)

echo
echo "Created Claude skill bundle: $OUTPUT_ZIP"
echo "Bundled CLI version:        $SAFE_VERSION"
echo "Contents:"
unzip -l "$OUTPUT_ZIP"
