#!/usr/bin/env bash

banyanlabs_configured_go_version() {
  local repo_root="$1"
  local mise_file="$repo_root/.mise.toml"

  [[ -f "$mise_file" ]] || return 1

  awk -F= '
    /^[[:space:]]*\[tools\][[:space:]]*$/ { in_tools = 1; next }
    /^[[:space:]]*\[/ { in_tools = 0 }
    in_tools && $1 ~ /^[[:space:]]*go[[:space:]]*$/ {
      value = $2
      sub(/#.*/, "", value)
      gsub(/[[:space:]"]/, "", value)
      print value
      exit
    }
  ' "$mise_file"
}

banyanlabs_go_major_minor() {
  local version="${1#go}"

  if [[ "$version" =~ ^([0-9]+)\.([0-9]+) ]]; then
    printf '%s.%s\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
    return 0
  fi

  return 1
}

banyanlabs_active_go_version() {
  go env GOVERSION 2>/dev/null || go version | awk '{print $3}'
}

banyanlabs_require_go_toolchain() {
  local repo_root="$1"
  local configured_version
  local expected_major_minor
  local active_version
  local active_major_minor

  if ! command -v go >/dev/null 2>&1; then
    printf 'Missing required command: go\n' >&2
    printf 'Run `basectl setup banyanlabs` to install the mise-managed Go toolchain.\n' >&2
    return 1
  fi

  configured_version="$(banyanlabs_configured_go_version "$repo_root")" || {
    printf 'Unable to read Go version from %s/.mise.toml.\n' "$repo_root" >&2
    return 1
  }
  if [[ -z "$configured_version" ]]; then
    printf 'No Go version is configured in %s/.mise.toml.\n' "$repo_root" >&2
    return 1
  fi

  expected_major_minor="$(banyanlabs_go_major_minor "$configured_version")" || {
    printf 'Unsupported Go version in %s/.mise.toml: %s\n' "$repo_root" "$configured_version" >&2
    return 1
  }

  active_version="$(banyanlabs_active_go_version)" || {
    printf 'Unable to determine active Go toolchain version.\n' >&2
    return 1
  }
  active_major_minor="$(banyanlabs_go_major_minor "$active_version")" || {
    printf 'Unable to parse active Go toolchain version: %s\n' "$active_version" >&2
    return 1
  }

  if [[ "$active_major_minor" != "$expected_major_minor" ]]; then
    printf 'Active Go toolchain is %s, but .mise.toml requires Go %s.\n' \
      "$active_version" "$configured_version" >&2
    printf 'Run `basectl setup banyanlabs` or `mise install` from the repo root, then retry.\n' >&2
    return 1
  fi
}
