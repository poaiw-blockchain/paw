#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VENV_PATH="${VENV_PATH:-${ROOT}/.venv-pytest}"

if [[ ! -d "${VENV_PATH}" ]]; then
  python3 -m venv "${VENV_PATH}"
fi

source "${VENV_PATH}/bin/activate"
pip install --upgrade pip

# Install cosmospy-protobuf first so paw-sdk picks up the local wheel.
pip install -e "${ROOT}/sdk/python/cosmospy-protobuf"
pip install -e "${ROOT}/archive/sdk/python[dev]"
