#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

find_env_file() {
  local cursor="$PWD"
  while true; do
    local candidate="$cursor/infra/.env"
    if [[ -f "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
    if [[ "$cursor" == "/" ]]; then
      return 1
    fi
    cursor="$(dirname "$cursor")"
  done
}

load_env_file() {
  local file="$1"
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "${line:0:1}" == "#" ]] && continue
    line="${line#export }"
    [[ "$line" != *"="* ]] && continue
    local key="${line%%=*}"
    local value="${line#*=}"
    key="${key//[[:space:]]/}"
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    if [[ -z "${!key:-}" ]]; then
      if [[ "$value" == \"*\" && "$value" == *\" ]]; then
        value="${value:1:${#value}-2}"
      elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
        value="${value:1:${#value}-2}"
      fi
      printf -v "$key" '%s' "$value"
      export "$key"
    fi
  done < "$file"
}

if env_file="$(find_env_file)"; then
  load_env_file "$env_file"
fi

case "$(uname -s)" in
  Linux) OS="linux" ;;
  *) echo "Unsupported OS for db-cli.sh: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture for db-cli.sh: $(uname -m)" >&2; exit 1 ;;
esac

BINARY_PATH="$SKILL_ROOT/bin/${OS}-${ARCH}/db-cli"
FALLBACK_PATH="$SKILL_ROOT/bin/db-cli"

if [[ ! -x "$BINARY_PATH" && -x "$FALLBACK_PATH" ]]; then
  BINARY_PATH="$FALLBACK_PATH"
fi

if [[ ! -x "$BINARY_PATH" ]]; then
  echo "No se encontro db-cli en '$BINARY_PATH'. Compilalo con: '$SCRIPT_DIR/build.sh' --all" >&2
  exit 1
fi

exec "$BINARY_PATH" "$@"
