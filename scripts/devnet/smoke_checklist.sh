#!/usr/bin/env bash
set -euo pipefail

# Parse smoke log output to confirm key milestones.
# Usage: ./scripts/devnet/smoke.sh | tee /tmp/smoke.log; ./scripts/devnet/smoke_checklist.sh /tmp/smoke.log

LOGFILE=${1:-}
if [[ -z "$LOGFILE" || ! -f "$LOGFILE" ]]; then
  echo "usage: $0 <smoke-log-file>" >&2
  exit 1
fi

check() {
  local needle=$1; local label=$2
  if grep -q "$needle" "$LOGFILE"; then
    printf "[ok] %s\n" "$label"
  else
    printf "[missing] %s\n" "$label"
  fi
}

check "[smoke] bank send" "bank send submitted"
check "[smoke] ensure validator" "validator ensured"
check "[smoke] delegate" "delegation tx"
check "[smoke] create pool" "dex pool created"
check "[smoke] add liquidity" "liquidity added"
check "[smoke] swap" "swap executed"
check "[smoke] gov submit" "gov proposal submitted"
check "[smoke] gov deposit" "gov deposit"
check "[smoke] gov vote" "gov vote"
check "[smoke] balances after" "post-state balances shown"
