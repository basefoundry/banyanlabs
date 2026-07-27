#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
service_dir="$repo_root/services/url-shortener"
tests_dir="$repo_root/tests/api/url-shortener"

# shellcheck source=../../scripts/go-toolchain.sh
source "$repo_root/scripts/go-toolchain.sh"

addr="${BANYAN_URL_SHORTENER_ADDR:-127.0.0.1:18080}"
base_url="${BANYAN_URL_SHORTENER_BASE_URL:-http://${addr}}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/banyanlabs-url-shortener-api.XXXXXX")"
go_cache_dir="$tmp_dir/go-cache"
go_tmp_dir="$tmp_dir/go-tmp"
binary_path="$tmp_dir/url-shortener"
server_pid=""

require_command() {
  local command_name="$1"

  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$command_name" >&2
    exit 1
  fi
}

cleanup() {
  local status=$?

  if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi

  rm -rf "$tmp_dir"
  exit "$status"
}

trap cleanup EXIT

require_command curl
banyanlabs_require_go_toolchain "$repo_root"
require_command hurl

mkdir -p "$go_cache_dir" "$go_tmp_dir"

printf 'Building URL shortener API binary.\n'
(
  cd "$service_dir"
  CGO_ENABLED="${CGO_ENABLED:-0}" \
    GOCACHE="$go_cache_dir" \
    GOTMPDIR="$go_tmp_dir" \
    go build -o "$binary_path" ./cmd/url-shortener
)

printf 'Starting URL shortener API at %s\n' "$base_url"
BANYAN_URL_SHORTENER_ADDR="$addr" \
  BANYAN_URL_SHORTENER_DATABASE="$tmp_dir/url-shortener.sqlite3" \
  BANYAN_LOG_LEVEL="${BANYAN_LOG_LEVEL:-warn}" \
  "$binary_path" >"$tmp_dir/url-shortener.log" 2>&1 &
server_pid=$!

ready=false
for _ in 1 2 3 4 5 6 7 8 9 10 \
  11 12 13 14 15 16 17 18 19 20 \
  21 22 23 24 25 26 27 28 29 30 \
  31 32 33 34 35 36 37 38 39 40 \
  41 42 43 44 45 46 47 48 49 50; do
  if curl --fail --silent "${base_url}/healthz" >/dev/null; then
    ready=true
    break
  fi

  if ! kill -0 "$server_pid" 2>/dev/null; then
    printf 'URL shortener exited before it became ready.\n' >&2
    sed 's/^/  /' "$tmp_dir/url-shortener.log" >&2
    exit 1
  fi

  sleep 0.2
done

if [[ "$ready" != true ]]; then
  printf 'Timed out waiting for URL shortener at %s.\n' "$base_url" >&2
  sed 's/^/  /' "$tmp_dir/url-shortener.log" >&2
  exit 1
fi

hurl --test --variable "base_url=${base_url}" "$tests_dir"/*.hurl

printf 'API smoke tests passed.\n'
