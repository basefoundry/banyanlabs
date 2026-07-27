#!/usr/bin/env bats

setup() {
  repo_dir="$(mktemp -d "${TMPDIR:-/tmp}/banyanlabs-go-toolchain.XXXXXX")"
  mkdir -p "$repo_dir/bin"
  helper="$BATS_TEST_DIRNAME/../scripts/go-toolchain.sh"

  printf '[tools]\ngo = "1.25"\n' >"$repo_dir/.mise.toml"
  cat >"$repo_dir/bin/go" <<'EOF'
#!/usr/bin/env bash
case "${1:-} ${2:-}" in
  "env GOVERSION")
    printf '%s\n' "$FAKE_GO_VERSION"
    ;;
  "version ")
    printf 'go version %s darwin/arm64\n' "$FAKE_GO_VERSION"
    ;;
  *)
    exit 0
    ;;
esac
EOF
  chmod +x "$repo_dir/bin/go"
}

teardown() {
  rm -rf "$repo_dir"
}

@test "go toolchain guard accepts matching major and minor version" {
  run env PATH="$repo_dir/bin:$PATH" FAKE_GO_VERSION=go1.25.12 \
    bash -c 'source "$1"; banyanlabs_require_go_toolchain "$2"' _ "$helper" "$repo_dir"

  [ "$status" -eq 0 ]
}

@test "go toolchain guard rejects mismatched major and minor version" {
  run env PATH="$repo_dir/bin:$PATH" FAKE_GO_VERSION=go1.24.13 \
    bash -c 'source "$1"; banyanlabs_require_go_toolchain "$2"' _ "$helper" "$repo_dir"

  [ "$status" -eq 1 ]
  [[ "$output" == *"Active Go toolchain is go1.24.13"* ]]
  [[ "$output" == *".mise.toml requires Go 1.25"* ]]
  [[ "$output" == *"basectl setup banyanlabs"* ]]
}
