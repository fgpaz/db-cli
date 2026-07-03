#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_ROOT="$SKILL_ROOT/bin"

OS="${1:-linux}"
ARCH="${2:-$(uname -m)}"
MODE="${3:-single}"

map_arch() {
  case "$1" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "$1" ;;
  esac
}

build_target() {
  local goos="$1"
  local goarch="$2"
  local target_dir="$BIN_ROOT/${goos}-${goarch}"
  local binary_name="db-cli"
  if [[ "$goos" == "windows" ]]; then
    binary_name="db-cli.exe"
  fi
  mkdir -p "$target_dir"
  echo "  target: ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -buildvcs=false -o "$target_dir/$binary_name" .
  chmod +x "$target_dir/$binary_name"
}

mkdir -p "$BIN_ROOT"
cd "$SCRIPT_DIR"
go mod tidy

if [[ "$MODE" == "--all" || "$OS" == "--all" ]]; then
  build_target linux amd64
  build_target linux arm64
  build_target windows amd64
  build_target windows arm64
else
  build_target "$OS" "$(map_arch "$ARCH")"
  if [[ "$OS" == "linux" ]]; then
    cp "$BIN_ROOT/${OS}-$(map_arch "$ARCH")/db-cli" "$BIN_ROOT/db-cli"
    chmod +x "$BIN_ROOT/db-cli"
  elif [[ "$OS" == "windows" ]]; then
    cp "$BIN_ROOT/${OS}-$(map_arch "$ARCH")/db-cli.exe" "$BIN_ROOT/db-cli.exe"
  fi
fi

find "$BIN_ROOT" -maxdepth 2 -type f -print
