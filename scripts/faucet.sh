#!/bin/bash
#
# PAW Testnet Faucet Script
# Usage: ./faucet.sh <recipient_address> [amount]
#        ./faucet.sh --check        # Run preflight without sending
#
# Default amount: 100000000 upaw (100 PAW)
# Rate limit: 1 request per address per hour (simple file-based tracking)
#
# This version includes full preflight validation so the faucet can be safely
# exposed for public testnet use (node health, IAVL fast node config, faucet
# key, and balance checks).

set -euo pipefail

# Configuration
CHAIN_ID="paw-testnet-1"
PAW_HOME="${PAW_HOME:-$HOME/.paw}"
PAWD="${PAWD:-$(dirname "$0")/../build/pawd}"
FAUCET_KEY="${FAUCET_KEY:-faucet}"
DEFAULT_AMOUNT="100000000"
DENOM="upaw"
GAS="auto"
GAS_ADJUSTMENT="1.3"
GAS_PRICES="0.001upaw"
KEYRING_BACKEND="test"
RPC_ADDR="${RPC_ADDR:-tcp://localhost:26657}"

# Rate limiting
RATE_LIMIT_DIR="${PAW_HOME}/faucet_rate_limit"
RATE_LIMIT_SECONDS=3600  # 1 hour

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    echo "PAW Testnet Faucet"
    echo ""
    echo "Usage: $0 <recipient_address> [amount]"
    echo "       $0 --check"
    echo ""
    echo "Arguments:"
    echo "  recipient_address  PAW address to send tokens to (paw1...)"
    echo "  amount            Amount in upaw (default: $DEFAULT_AMOUNT = 100 PAW)"
    echo ""
    echo "Environment Variables:"
    echo "  PAW_HOME          Node home directory (default: ~/.paw)"
    echo "  PAWD              Path to pawd binary (default: ./build/pawd)"
    echo "  RPC_ADDR          Node RPC endpoint (default: $RPC_ADDR)"
    echo ""
    echo "Examples:"
    echo "  $0 paw1abc123def456..."
    echo "  $0 paw1abc123def456... 50000000"
    echo "  $0 --check"
    exit 1
}

ensure_dependencies() {
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed."
        exit 1
    fi
    if ! command -v bc >/dev/null 2>&1; then
        log_error "bc is required but not installed."
        exit 1
    fi
}

validate_address() {
    local addr="$1"
    if [[ ! "$addr" =~ ^paw1[a-z0-9]{38}$ ]]; then
        log_error "Invalid PAW address format: $addr"
        log_error "Address must start with 'paw1' and be 42 characters total"
        exit 1
    fi
}

check_rate_limit() {
    local addr="$1"
    local addr_hash=$(echo -n "$addr" | md5sum | cut -d' ' -f1)
    local rate_file="${RATE_LIMIT_DIR}/${addr_hash}"

    mkdir -p "$RATE_LIMIT_DIR"

    if [[ -f "$rate_file" ]]; then
        local last_request=$(cat "$rate_file")
        local current_time=$(date +%s)
        local time_diff=$((current_time - last_request))

        if [[ $time_diff -lt $RATE_LIMIT_SECONDS ]]; then
            local wait_time=$((RATE_LIMIT_SECONDS - time_diff))
            local wait_mins=$((wait_time / 60))
            log_error "Rate limit exceeded for address: $addr"
            log_error "Please wait $wait_mins minutes before requesting again"
            exit 1
        fi
    fi

    # Update rate limit timestamp
    echo "$(date +%s)" > "$rate_file"
}

ensure_pawd_binary() {
    if [[ ! -x "$PAWD" ]]; then
        log_error "pawd binary not found at: $PAWD"
        log_error "Build it with: go build -o build/pawd ./cmd/..."
        exit 1
    fi
}

ensure_faucet_key() {
    if ! "$PAWD" keys show "$FAUCET_KEY" --home "$PAW_HOME" --keyring-backend "$KEYRING_BACKEND" >/dev/null 2>&1; then
        log_error "Faucet key \"$FAUCET_KEY\" not found in keyring (backend: $KEYRING_BACKEND)."
        log_error "Import or create it first (see scripts/devnet/.state/ for mnemonics)."
        exit 1
    fi

    "$PAWD" keys show "$FAUCET_KEY" -a --home "$PAW_HOME" --keyring-backend "$KEYRING_BACKEND"
}

preflight_node() {
    local app_toml="${PAW_HOME}/config/app.toml"
    if [[ -f "$app_toml" ]] && grep -q "iavl-disable-fastnode = true" "$app_toml"; then
        log_error "iavl-disable-fastnode is set to true. Set it to false to avoid IAVL query failures."
        log_error "Fix: sed -i 's/iavl-disable-fastnode = true/iavl-disable-fastnode = false/' \"$app_toml\" && restart the node."
        exit 1
    fi

    local status_json
    if ! status_json=$("$PAWD" status --node "$RPC_ADDR" --home "$PAW_HOME" 2>/dev/null); then
        log_error "Unable to reach node at $RPC_ADDR. Is pawd running?"
        exit 1
    fi

    local catching_up
    catching_up=$(echo "$status_json" | jq -r '.SyncInfo.catching_up // .sync_info.catching_up // empty')
    local latest_height
    latest_height=$(echo "$status_json" | jq -r '.SyncInfo.latest_block_height // .sync_info.latest_block_height // "unknown"')

    if [[ "$catching_up" == "true" ]]; then
        log_error "Node at $RPC_ADDR is still catching up (latest height: $latest_height)."
        exit 1
    fi

    log_info "Node healthy at height $latest_height (catching_up=false)."
}

get_faucet_balance() {
    local faucet_addr="$1"
    local balances_json

    if ! balances_json=$("$PAWD" q bank balances "$faucet_addr" --node "$RPC_ADDR" --home "$PAW_HOME" --output json 2>/dev/null); then
        log_warn "Balance query failed; proceeding without a preflight balance guard."
        echo "unknown"
        return
    fi

    local amount
    amount=$(echo "$balances_json" | jq -r --arg denom "$DENOM" '.balances[]? | select(.denom==$denom) | .amount' | head -n1)
    echo "${amount:-0}"
}

send_tokens() {
    local recipient="$1"
    local amount="$2"

    log_info "Sending $amount $DENOM to $recipient..."

    # Execute the transfer
    local result=$("$PAWD" tx bank send "$FAUCET_KEY" "$recipient" "${amount}${DENOM}" \
        --home "$PAW_HOME" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING_BACKEND" \
        --node "$RPC_ADDR" \
        --gas "$GAS" \
        --gas-adjustment "$GAS_ADJUSTMENT" \
        --gas-prices "$GAS_PRICES" \
        --broadcast-mode block \
        --yes \
        --output json 2>&1)

    # Check result
    local code=$(echo "$result" | jq -r '.code // empty' 2>/dev/null)
    local txhash=$(echo "$result" | jq -r '.txhash // empty' 2>/dev/null)

    if [[ "$code" == "0" ]] || [[ -n "$txhash" && "$code" == "" ]]; then
        log_info "Transaction submitted successfully!"
        log_info "TxHash: $txhash"
        echo ""
        echo "Tokens sent:"
        echo "  Amount: $amount $DENOM ($(echo "scale=6; $amount / 1000000" | bc) PAW)"
        echo "  To: $recipient"
        echo "  TxHash: $txhash"
        return 0
    else
        local raw_log=$(echo "$result" | jq -r '.raw_log // empty' 2>/dev/null)
        log_error "Transaction failed!"
        log_error "Code: $code"
        log_error "Log: ${raw_log:-$result}"
        return 1
    fi
}

# Main
main() {
    if [[ $# -lt 1 ]]; then
        usage
    fi

    if [[ "$1" == "--check" ]]; then
        ensure_dependencies
        ensure_pawd_binary
        local faucet_addr
        faucet_addr=$(ensure_faucet_key)
        preflight_node
        local faucet_balance
        faucet_balance=$(get_faucet_balance "$faucet_addr")
        log_info "Faucet address: $faucet_addr"
        log_info "Preflight balance: ${faucet_balance} ${DENOM}"
        log_info "All preflight checks passed."
        exit 0
    fi

    local recipient="$1"
    local amount="${2:-$DEFAULT_AMOUNT}"

    # Validate inputs
    validate_address "$recipient"

    if ! [[ "$amount" =~ ^[0-9]+$ ]]; then
        log_error "Invalid amount: $amount (must be a positive integer)"
        exit 1
    fi

    ensure_dependencies
    ensure_pawd_binary
    local faucet_addr
    faucet_addr=$(ensure_faucet_key)
    preflight_node

    # Check rate limit
    check_rate_limit "$recipient"

    # Check faucet balance (skip if query not available)
    local faucet_balance
    faucet_balance=$(get_faucet_balance "$faucet_addr")
    if [[ "$faucet_balance" != "unknown" ]]; then
        log_info "Faucet balance: $faucet_balance $DENOM"
        if [[ "$faucet_balance" -lt "$amount" ]]; then
            log_error "Insufficient faucet balance!"
            log_error "Available: $faucet_balance $DENOM"
            log_error "Requested: $amount $DENOM"
            exit 1
        fi
    fi

    # Send tokens
    send_tokens "$recipient" "$amount"
}

main "$@"
