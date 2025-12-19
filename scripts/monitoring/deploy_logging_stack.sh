#!/usr/bin/env bash
# Deploy or manage the Loki + Promtail logging stack used for centralized log aggregation.
set -euo pipefail

ACTION="${1:-up}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_DIR="${ROOT_DIR}/compose"
COMPOSE_FILE="${COMPOSE_DIR}/docker-compose.logging.yml"
NETWORK_NAME="compose_monitoring"

function usage() {
  cat <<EOF
Usage: $(basename "$0") [up|down|status]

Actions:
  up      Start or redeploy the Loki + Promtail stack (default)
  down    Stop the logging stack
  status  Show container status + health
EOF
}

function require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: Missing required command: $1" >&2
    exit 1
  fi
}

function docker_compose() {
  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    docker compose "$@"
  fi
}

function ensure_network() {
  if ! docker network inspect "${NETWORK_NAME}" >/dev/null 2>&1; then
    echo "Creating monitoring network ${NETWORK_NAME}..."
    docker network create "${NETWORK_NAME}" >/dev/null
  fi
}

function verify_stack() {
  echo "Waiting for Loki readiness endpoint..."
  curl -sf --retry 20 --retry-delay 3 "http://localhost:${LOKI_PORT:-11025}/ready" >/dev/null
  echo "Loki is ready on port ${LOKI_PORT:-11025}"
  echo "Fetching last 5 log lines via API check..."
  curl -G -s "http://localhost:${LOKI_PORT:-11025}/loki/api/v1/query" \
    --data-urlencode 'query={job="paw-docker"}' \
    --data-urlencode 'limit=5' | jq '.status' 2>/dev/null || true
}

function status() {
  docker ps --filter "name=paw-loki" --filter "name=paw-promtail"
  echo
  echo "Recent Loki logs:"
  docker logs --tail 20 paw-loki 2>/dev/null || true
  echo
  echo "Recent Promtail logs:"
  docker logs --tail 20 paw-promtail 2>/dev/null || true
}

require_command docker
require_command jq

case "${ACTION}" in
  up)
    ensure_network
    echo "Starting Loki + Promtail using ${COMPOSE_FILE}"
    docker_compose -f "${COMPOSE_FILE}" up -d
    verify_stack
    ;;
  down)
    echo "Stopping Loki + Promtail"
    docker_compose -f "${COMPOSE_FILE}" down
    ;;
  status)
    status
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "ERROR: Unknown action '${ACTION}'"
    usage
    exit 1
    ;;
esac
