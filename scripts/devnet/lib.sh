#!/usr/bin/env bash
# Shared helpers for devnet automation scripts.

DEVNET_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

ensure_pawd_binary() {
  local scope="${1:-devnet}"
  local binary="${PAWD_BINARY:-${DEVNET_ROOT}/pawd}"
  local go_bin="${GO_BIN:-$(command -v go || echo /usr/local/go/bin/go)}"
  local rebuild=0

  if [[ ! -x "${go_bin}" ]]; then
    echo "[${scope}] go toolchain not found at ${go_bin}" >&2
    exit 1
  fi

  if [[ ! -x "${binary}" ]]; then
    rebuild=1
  elif find "${DEVNET_ROOT}/cmd/pawd" "${DEVNET_ROOT}/app" -type f -name '*.go' -newer "${binary}" | read -r; then
    rebuild=1
  fi

  if (( rebuild == 1 )); then
    echo "[${scope}] building pawd binary at ${binary}"
    mkdir -p "${DEVNET_ROOT}/.cache/go-build" "${DEVNET_ROOT}/.cache/go/pkg/mod"
    GOCACHE="${DEVNET_ROOT}/.cache/go-build" \
    GOMODCACHE="${DEVNET_ROOT}/.cache/go/pkg/mod" \
      "${go_bin}" build -o "${binary}" ./cmd/pawd
  else
    echo "[${scope}] reusing existing pawd binary at ${binary}"
  fi

  local sha
  sha=$(sha256sum "${binary}" | awk '{print $1}')
  echo "[${scope}] pawd sha256=${sha}"
}

# Extract the bech32 address from `pawd keys show` output (custom PAW CLI prints YAML-like output).
extract_key_address() {
  awk '/^[[:space:]]*address:/{print $2; exit}' <<<"$1"
}

# Show a key and return its address. Returns empty string if the key is missing.
show_key_address() {
  local binary=${1:?pawd binary required}
  local name=${2:?key name required}
  shift 2
  local output
  output=$("${binary}" keys show "${name}" "$@" 2>/dev/null || true)
  if [[ -z "${output}" ]]; then
    return 1
  fi
  local addr
  addr=$(extract_key_address "${output}")
  if [[ -n "${addr}" ]]; then
    printf '%s\n' "${addr}"
    return 0
  fi
  return 1
}
