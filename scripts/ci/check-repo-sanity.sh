#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$REPO_ROOT"

if command -v rg >/dev/null 2>&1; then
  SEARCH_CMD=(rg)
  FILTER_CMD=(rg -v)
else
  SEARCH_CMD=(grep -E)
  FILTER_CMD=(grep -vE)
fi

FORBIDDEN_RE='\.(exe|dll|so|dylib|a|o|bin|test|apk|ipa|aab|deb|rpm|AppImage)$'
SENSITIVE_RE='\.(pem|key|p12|pfx|keystore|jks|mnemonic)$'
ENV_RE='(^|/)\.env(\..*)?$'
ALLOW_ENV_RE='\.env(\.[^/]+)?\.(example|template)$'

fail=0

check_pattern() {
  local name="$1"
  local pattern="$2"
  local matches

  matches=$(git ls-files | "${SEARCH_CMD[@]}" "$pattern" || true)
  if [ -n "$matches" ]; then
    echo "ERROR: $name files are tracked:"
    echo "$matches"
    fail=1
  fi
}

check_env_files() {
  local matches filtered
  matches=$(git ls-files | "${SEARCH_CMD[@]}" "$ENV_RE" || true)
  if [ -n "$matches" ]; then
    filtered=$(echo "$matches" | "${FILTER_CMD[@]}" "$ALLOW_ENV_RE" || true)
    if [ -n "$filtered" ]; then
      echo "ERROR: .env files are tracked:"
      echo "$filtered"
      fail=1
    fi
  fi
}

check_pattern "Forbidden binary" "$FORBIDDEN_RE"
check_pattern "Sensitive" "$SENSITIVE_RE"
check_env_files

SKIP_SIZE_CHECK="${SKIP_SIZE_CHECK:-0}"
SIZE_LIMIT_MB="${SIZE_LIMIT_MB:-10}"
SIZE_LIMIT_BYTES=$((SIZE_LIMIT_MB * 1024 * 1024))

if [ "$SKIP_SIZE_CHECK" != "1" ]; then
  oversized=""
  while IFS= read -r -d '' file; do
    [ -f "$file" ] || continue
    size=$(stat -c %s "$file")
    if [ "$size" -gt "$SIZE_LIMIT_BYTES" ]; then
      oversized+="${file}\t${size}\n"
    fi
  done < <(git ls-files -z)

  if [ -n "$oversized" ]; then
    echo "ERROR: Tracked files larger than ${SIZE_LIMIT_MB}MB:"
    printf '%b' "$oversized"
    fail=1
  fi
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi

echo "Repo sanity checks passed."
