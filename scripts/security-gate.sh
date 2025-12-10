#!/usr/bin/env bash
# Security gate: gosec + staticcheck + govulncheck + go mod verify

set -euo pipefail

TOOLS=(
  "gosec::github.com/securego/gosec/v2/cmd/gosec@latest"
  "staticcheck::honnef.co/go/tools/cmd/staticcheck@latest"
  "govulncheck::golang.org/x/vuln/cmd/govulncheck@latest"
)

missing=()
for entry in "${TOOLS[@]}"; do
  tool="${entry%%::*}"
  if ! command -v "${tool}" >/dev/null 2>&1; then
    missing+=("${entry}")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "Missing required tools:"
  for entry in "${missing[@]}"; do
    tool="${entry%%::*}"
    pkg="${entry##*::}"
    echo "  ${tool}: go install ${pkg}"
  done
  exit 1
fi

GOSEC_ARGS=${GOSEC_ARGS:--exclude-dir=archive -exclude-dir=tests -exclude-dir=script/coverage_tools}
echo "Running gosec... (args: ${GOSEC_ARGS})"
gosec ${GOSEC_ARGS} ./...

echo "Running staticcheck..."
staticcheck ./...

echo "Running govulncheck..."
govulncheck ./...

echo "Verifying go modules..."
go mod verify

echo "Security gate passed."
