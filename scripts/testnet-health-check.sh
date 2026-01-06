#!/usr/bin/env bash
# Aggregated health check for PAW testnet (validators + public endpoints).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VALIDATOR_CHECK="${SCRIPT_DIR}/testnet/health-validators.sh"
PUBLIC_CHECK="${SCRIPT_DIR}/testnet/health-public.sh"

JSON_OUTPUT=false
QUIET=false

usage() {
  cat <<'USAGE'
Usage: testnet-health-check.sh [options]

Options:
  --json        JSON output (aggregates validator + public checks)
  --quiet       Suppress non-JSON output
  -h, --help    Show this help

For advanced options, run the underlying scripts directly:
  scripts/testnet/health-validators.sh
  scripts/testnet/health-public.sh
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --json) JSON_OUTPUT=true; shift;;
    --quiet|-q) QUIET=true; shift;;
    -h|--help) usage; exit 0;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1;;
  esac
done

if [[ ! -x "$VALIDATOR_CHECK" ]]; then
  echo "error: validator check script not found: $VALIDATOR_CHECK" >&2
  exit 1
fi
if [[ ! -x "$PUBLIC_CHECK" ]]; then
  echo "error: public check script not found: $PUBLIC_CHECK" >&2
  exit 1
fi

overall_state="HEALTHY"

if [[ "$JSON_OUTPUT" == true ]]; then
  set +e
  validator_json=$($VALIDATOR_CHECK --json)
  validator_rc=$?
  public_json=$($PUBLIC_CHECK --json)
  public_rc=$?
  set -e

  if [[ $validator_rc -eq 1 || $public_rc -eq 1 ]]; then
    overall_state="UNHEALTHY"
  elif [[ $validator_rc -eq 2 || $public_rc -eq 2 ]]; then
    overall_state="DEGRADED"
  fi

  jq -n \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg state "$overall_state" \
    --argjson validators "$validator_json" \
    --argjson public "$public_json" \
    '{timestamp: $timestamp, state: $state, validators: $validators, public: $public}'
else
  set +e
  if [[ "$QUIET" == true ]]; then
    $VALIDATOR_CHECK --quiet
    validator_rc=$?
    $PUBLIC_CHECK --quiet
    public_rc=$?
  else
    echo "PAW Testnet Health Check"
    echo "=========================="
    echo ""
    $VALIDATOR_CHECK
    validator_rc=$?
    $PUBLIC_CHECK
    public_rc=$?
  fi
  set -e

  if [[ $validator_rc -eq 1 || $public_rc -eq 1 ]]; then
    overall_state="UNHEALTHY"
  elif [[ $validator_rc -eq 2 || $public_rc -eq 2 ]]; then
    overall_state="DEGRADED"
  fi

  if [[ "$QUIET" == false ]]; then
    echo "Overall: $overall_state"
  fi
fi

if [[ "$overall_state" == "UNHEALTHY" ]]; then
  exit 1
fi
if [[ "$overall_state" == "DEGRADED" ]]; then
  exit 2
fi

exit 0
