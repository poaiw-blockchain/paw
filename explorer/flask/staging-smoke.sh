#!/bin/bash
# Lightweight smoke check for the staging explorer stack

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

check() {
  local name="$1"
  local url="$2"
  if curl -sf --max-time 5 "$url" >/dev/null; then
    echo "[ok] $name: $url"
  else
    echo "[fail] $name: $url"
    return 1
  fi
}

main() {
  check "nginx (explorer)" "http://localhost:11083/health"
  # Direct Flask port is only exposed when run without nginx; keep as best-effort
  check "flask (direct)" "http://localhost:5000/health" || true
  check "indexer (stub)" "http://localhost:11081/health"
  # Postgres isn't HTTP; rely on compose health or pg_isready manually
  check "prometheus" "http://localhost:11091/-/healthy"
}

main "$@"
