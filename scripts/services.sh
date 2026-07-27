#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
service_dir="$repo_root/services/url-shortener"

# shellcheck source=scripts/go-toolchain.sh
source "$repo_root/scripts/go-toolchain.sh"

runtime_root="${BANYANLABS_RUNTIME_DIR:-$repo_root/var}"
run_dir="$runtime_root/run"
log_dir="$runtime_root/log"
bin_dir="$runtime_root/bin"
cache_dir="$runtime_root/cache"

service_name="url-shortener"
addr="${BANYAN_URL_SHORTENER_ADDR:-127.0.0.1:8080}"
database_path="${BANYAN_URL_SHORTENER_DATABASE:-$runtime_root/url-shortener/url-shortener.sqlite3}"
log_level="${BANYAN_LOG_LEVEL:-info}"
health_url="http://${addr}/healthz"

pid_file="$run_dir/${service_name}.pid"
log_file="$log_dir/${service_name}.log"
binary_path="$bin_dir/${service_name}"

usage() {
  cat <<'EOF'
Usage:
  scripts/services.sh build
  scripts/services.sh dev [--foreground]
  scripts/services.sh status
  scripts/services.sh stop

Commands:
  build         Build local service binaries.
  dev           Start local services in the background by default.
  status        List local service status.
  stop          Stop background local services started by this script.

Options:
  --foreground  Run the URL shortener in the foreground.
EOF
}

require_command() {
  local command_name="$1"

  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$command_name" >&2
    exit 1
  fi
}

read_pid() {
  [[ -f "$pid_file" ]] || return 1

  local pid
  pid="$(<"$pid_file")"
  [[ "$pid" =~ ^[0-9]+$ ]] || return 1
  printf '%s\n' "$pid"
}

pid_is_running() {
  local pid="$1"

  kill -0 "$pid" >/dev/null 2>&1
}

health_is_ok() {
  curl --fail --silent --max-time 2 "$health_url" >/dev/null 2>&1
}

build_url_shortener() {
  local go_cache_dir="$cache_dir/go-build"
  local go_tmp_dir="$cache_dir/go-tmp"

  banyanlabs_require_go_toolchain "$repo_root"
  mkdir -p "$bin_dir" "$go_cache_dir" "$go_tmp_dir" "$(dirname "$database_path")"

  printf 'Building %s.\n' "$service_name"
  (
    cd "$service_dir"
    CGO_ENABLED="${CGO_ENABLED:-0}" \
      GOCACHE="${GOCACHE:-$go_cache_dir}" \
      GOTMPDIR="${GOTMPDIR:-$go_tmp_dir}" \
      go build -o "$binary_path" ./cmd/url-shortener
  )
}

wait_for_health() {
  local pid="$1"

  for _ in 1 2 3 4 5 6 7 8 9 10 \
    11 12 13 14 15 16 17 18 19 20 \
    21 22 23 24 25 26 27 28 29 30 \
    31 32 33 34 35 36 37 38 39 40 \
    41 42 43 44 45 46 47 48 49 50; do
    health_is_ok && return 0

    if ! pid_is_running "$pid"; then
      return 1
    fi

    sleep 0.2
  done

  return 1
}

print_status_header() {
  printf '%-18s %-12s %-8s %-22s %s\n' "SERVICE" "STATUS" "PID" "ADDRESS" "DETAIL"
}

print_url_shortener_status() {
  local pid="-"
  local found_pid=""
  local status="stopped"
  local detail="not responding"

  if found_pid="$(read_pid)"; then
    pid="$found_pid"
    if pid_is_running "$pid"; then
      if health_is_ok; then
        status="running"
        detail="healthy"
      else
        status="unhealthy"
        detail="process is running but /healthz failed"
      fi
    else
      status="stale"
      detail="pid file exists but process is not running"
    fi
  elif health_is_ok; then
    status="external"
    detail="healthy but not managed by this script"
  fi

  printf '%-18s %-12s %-8s %-22s %s\n' "$service_name" "$status" "$pid" "$addr" "$detail"
}

status_services() {
  require_command curl

  print_status_header
  print_url_shortener_status
}

start_foreground() {
  build_url_shortener
  printf 'Starting %s in the foreground at %s.\n' "$service_name" "$health_url"
  exec env \
    BANYAN_URL_SHORTENER_ADDR="$addr" \
    BANYAN_URL_SHORTENER_DATABASE="$database_path" \
    BANYAN_LOG_LEVEL="$log_level" \
    "$binary_path"
}

start_background() {
  local pid=""

  require_command curl
  mkdir -p "$run_dir" "$log_dir"

  if pid="$(read_pid)" && pid_is_running "$pid"; then
    if health_is_ok; then
      printf '%s is already running.\n' "$service_name"
      status_services
      return 0
    fi

    printf '%s has pid %s but failed health checks.\n' "$service_name" "$pid" >&2
    status_services
    return 1
  fi

  if [[ -f "$pid_file" ]]; then
    rm -f "$pid_file"
  fi

  if health_is_ok; then
    printf '%s is already responding but is not managed by this script.\n' "$service_name" >&2
    status_services
    return 0
  fi

  build_url_shortener

  printf 'Starting %s in the background at %s.\n' "$service_name" "$health_url"
  nohup env \
    BANYAN_URL_SHORTENER_ADDR="$addr" \
    BANYAN_URL_SHORTENER_DATABASE="$database_path" \
    BANYAN_LOG_LEVEL="$log_level" \
    "$binary_path" >"$log_file" 2>&1 </dev/null &
  pid="$!"
  printf '%s\n' "$pid" >"$pid_file"

  if wait_for_health "$pid"; then
    printf '%s started. pid=%s log=%s\n' "$service_name" "$pid" "$log_file"
    status_services
    return 0
  fi

  printf '%s did not become healthy. Recent log output:\n' "$service_name" >&2
  sed 's/^/  /' "$log_file" >&2 || true
  return 1
}

stop_services() {
  local pid=""

  if ! pid="$(read_pid)"; then
    printf '%s is not managed by this script.\n' "$service_name"
    status_services
    return 0
  fi

  if ! pid_is_running "$pid"; then
    rm -f "$pid_file"
    printf '%s was not running; removed stale pid file.\n' "$service_name"
    status_services
    return 0
  fi

  printf 'Stopping %s. pid=%s\n' "$service_name" "$pid"
  kill "$pid"

  for _ in 1 2 3 4 5 6 7 8 9 10 \
    11 12 13 14 15 16 17 18 19 20 \
    21 22 23 24 25; do
    if ! pid_is_running "$pid"; then
      rm -f "$pid_file"
      printf '%s stopped.\n' "$service_name"
      status_services
      return 0
    fi
    sleep 0.2
  done

  printf '%s did not stop within timeout. pid=%s\n' "$service_name" "$pid" >&2
  return 1
}

run_dev() {
  local foreground=0

  while (($#)); do
    case "$1" in
      --foreground)
        foreground=1
        shift
        ;;
      -h|--help|help)
        usage
        return 0
        ;;
      *)
        usage >&2
        printf 'ERROR: Unknown dev option: %s\n' "$1" >&2
        return 2
        ;;
    esac
  done

  if ((foreground == 1)); then
    start_foreground
  else
    start_background
  fi
}

main() {
  local command="${1:-}"

  case "$command" in
    dev)
      shift
      run_dev "$@"
      ;;
    build)
      shift
      (($# == 0)) || {
        usage >&2
        printf 'ERROR: build does not accept extra arguments.\n' >&2
        return 2
      }
      build_url_shortener
      ;;
    status)
      shift
      (($# == 0)) || {
        usage >&2
        printf 'ERROR: status does not accept extra arguments.\n' >&2
        return 2
      }
      status_services
      ;;
    stop)
      shift
      (($# == 0)) || {
        usage >&2
        printf 'ERROR: stop does not accept extra arguments.\n' >&2
        return 2
      }
      stop_services
      ;;
    -h|--help|help|"")
      usage
      ;;
    *)
      usage >&2
      printf 'ERROR: Unknown command: %s\n' "$command" >&2
      return 2
      ;;
  esac
}

main "$@"
