#!/bin/bash
#
# PAW Testnet Faucet Script
# Usage: ./faucet.sh <recipient_address> [amount]
#
# Default amount: 100000000 upaw (100 PAW)
# Rate limit: 1 request per address per hour (simple file-based tracking)
#
# KNOWN ISSUE: The faucet currently fails due to an IAVL state query bug in the
# PAW chain. All state queries return "version does not exist" errors, which
# prevents the SDK client from querying account info needed to broadcast
# transactions. This needs to be fixed in the app module service registration
# before the faucet will work. See: ROADMAP_PRODUCTION.md
#

set -e

# Configuration
CHAIN_ID="paw-testnet-1"
PAW_HOME="${PAW_HOME:-$HOME/.paw}"
PAWD="${PAWD:-$(dirname "$0")/../build/pawd}"
FAUCET_KEY="validator"
DEFAULT_AMOUNT="100000000"
DENOM="upaw"
GAS="auto"
GAS_ADJUSTMENT="1.3"
GAS_PRICES="0.001upaw"
KEYRING_BACKEND="test"

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
    echo ""
    echo "Arguments:"
    echo "  recipient_address  PAW address to send tokens to (paw1...)"
    echo "  amount            Amount in upaw (default: $DEFAULT_AMOUNT = 100 PAW)"
    echo ""
    echo "Environment Variables:"
    echo "  PAW_HOME          Node home directory (default: ~/.paw)"
    echo "  PAWD              Path to pawd binary (default: ./build/pawd)"
    echo ""
    echo "Examples:"
    echo "  $0 paw1abc123def456..."
    echo "  $0 paw1abc123def456... 50000000"
    exit 1
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

get_faucet_balance() {
    # Note: query bank is not registered in this chain version
    # Skip balance checking - transaction will fail if insufficient funds
    echo "unknown"
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
        --gas "$GAS" \
        --gas-adjustment "$GAS_ADJUSTMENT" \
        --gas-prices "$GAS_PRICES" \
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

    local recipient="$1"
    local amount="${2:-$DEFAULT_AMOUNT}"

    # Validate inputs
    validate_address "$recipient"

    if ! [[ "$amount" =~ ^[0-9]+$ ]]; then
        log_error "Invalid amount: $amount (must be a positive integer)"
        exit 1
    fi

    # Check pawd binary exists
    if [[ ! -x "$PAWD" ]]; then
        log_error "pawd binary not found at: $PAWD"
        log_error "Build it with: go build -o build/pawd ./cmd/..."
        exit 1
    fi

    # Check rate limit
    check_rate_limit "$recipient"

    # Check faucet balance (skip if query not available)
    local faucet_balance=$(get_faucet_balance)
    if [[ "$faucet_balance" != "unknown" ]]; then
        log_info "Faucet balance: $faucet_balance $DENOM"
        if [[ "$faucet_balance" -lt "$amount" ]]; then
            log_error "Insufficient faucet balance!"
            log_error "Available: $faucet_balance $DENOM"
            log_error "Requested: $amount $DENOM"
            exit 1
        fi
    else
        log_warn "Balance check skipped (query not available)"
    fi

    # Send tokens
    send_tokens "$recipient" "$amount"
}

main "$@"
