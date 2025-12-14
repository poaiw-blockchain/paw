#!/bin/bash
# Metrics Verification Script
# Checks Prometheus endpoints and validates metrics accessibility after node startup
#
# Usage:
#   ./verify-metrics.sh [OPTIONS]
#
# Options:
#   --node-host HOST     Node hostname/IP (default: localhost)
#   --cometbft-port PORT CometBFT metrics port (default: 26660)
#   --api-port PORT      Cosmos SDK API metrics port (default: 1317)
#   --app-port PORT      Application metrics port (default: 26661)
#   --timeout SEC        HTTP timeout in seconds (default: 5)
#   --wait SEC           Wait time before checks (default: 0)
#   --verbose            Show detailed output
#   --json               Output results as JSON
#   --help               Show this help message
#
# Exit codes:
#   0 - All metrics endpoints accessible
#   1 - One or more endpoints failed
#   2 - Invalid arguments

set -euo pipefail

# Default configuration
NODE_HOST="${NODE_HOST:-localhost}"
COMETBFT_PORT="${COMETBFT_PORT:-26660}"
API_PORT="${API_PORT:-1317}"
APP_PORT="${APP_PORT:-26661}"
TIMEOUT="${TIMEOUT:-5}"
WAIT_TIME=0
VERBOSE=false
JSON_OUTPUT=false

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --node-host)
            NODE_HOST="$2"
            shift 2
            ;;
        --cometbft-port)
            COMETBFT_PORT="$2"
            shift 2
            ;;
        --api-port)
            API_PORT="$2"
            shift 2
            ;;
        --app-port)
            APP_PORT="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --wait)
            WAIT_TIME="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --help)
            grep '^#' "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Use --help for usage information" >&2
            exit 2
            ;;
    esac
done

# Logging functions
log_info() {
    if [[ "$JSON_OUTPUT" == "false" ]]; then
        echo -e "${BLUE}[INFO]${NC} $*"
    fi
}

log_success() {
    if [[ "$JSON_OUTPUT" == "false" ]]; then
        echo -e "${GREEN}[✓]${NC} $*"
    fi
}

log_warning() {
    if [[ "$JSON_OUTPUT" == "false" ]]; then
        echo -e "${YELLOW}[⚠]${NC} $*"
    fi
}

log_error() {
    if [[ "$JSON_OUTPUT" == "false" ]]; then
        echo -e "${RED}[✗]${NC} $*" >&2
    fi
}

log_verbose() {
    if [[ "$VERBOSE" == "true" && "$JSON_OUTPUT" == "false" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $*"
    fi
}

# Wait before starting checks
if [[ $WAIT_TIME -gt 0 ]]; then
    log_info "Waiting ${WAIT_TIME} seconds before checks..."
    sleep "$WAIT_TIME"
fi

# Check if curl is available
if ! command -v curl &> /dev/null; then
    log_error "curl is required but not installed"
    exit 2
fi

# Results storage
declare -A RESULTS
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# Function to check a metrics endpoint
check_endpoint() {
    local endpoint_name="$1"
    local url="$2"
    local expected_metrics="$3"  # Comma-separated list of expected metric prefixes

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    log_verbose "Checking ${endpoint_name}: ${url}"

    # Attempt to fetch metrics
    local http_code
    local response
    local curl_exit_code

    response=$(curl -s -w "\n%{http_code}" --connect-timeout "$TIMEOUT" --max-time "$((TIMEOUT + 5))" "$url" 2>&1) || curl_exit_code=$?

    if [[ ${curl_exit_code:-0} -ne 0 ]]; then
        log_error "${endpoint_name} - Connection failed (curl exit code: ${curl_exit_code})"
        RESULTS["$endpoint_name"]="FAILED:Connection error"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi

    http_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')

    if [[ "$http_code" != "200" ]]; then
        log_error "${endpoint_name} - HTTP ${http_code}"
        RESULTS["$endpoint_name"]="FAILED:HTTP ${http_code}"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi

    # Check if response contains Prometheus-format metrics
    if ! echo "$response" | grep -q "^# HELP"; then
        log_warning "${endpoint_name} - Response doesn't appear to be Prometheus format"
        RESULTS["$endpoint_name"]="WARNING:Invalid format"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi

    # Check for expected metrics
    local missing_metrics=()
    IFS=',' read -ra METRICS <<< "$expected_metrics"
    for metric in "${METRICS[@]}"; do
        if ! echo "$response" | grep -q "^${metric}"; then
            missing_metrics+=("$metric")
            log_verbose "Expected metric '${metric}' not found"
        else
            log_verbose "Found expected metric: ${metric}"
        fi
    done

    # Count total metrics
    local metric_count
    metric_count=$(echo "$response" | grep -c "^[a-z]" || true)

    if [[ ${#missing_metrics[@]} -gt 0 ]]; then
        log_warning "${endpoint_name} - OK (${metric_count} metrics, missing: ${missing_metrics[*]})"
        RESULTS["$endpoint_name"]="WARNING:Missing metrics ${missing_metrics[*]}"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))  # Still count as passed since endpoint is accessible
    else
        log_success "${endpoint_name} - OK (${metric_count} metrics)"
        RESULTS["$endpoint_name"]="SUCCESS:${metric_count} metrics"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    fi

    return 0
}

# Function to check if port is listening
check_port_listening() {
    local port="$1"
    local endpoint_name="$2"

    log_verbose "Checking if port ${port} is listening..."

    if command -v netstat &> /dev/null; then
        if netstat -tuln 2>/dev/null | grep -q ":${port}"; then
            log_verbose "Port ${port} is listening"
            return 0
        fi
    elif command -v ss &> /dev/null; then
        if ss -tuln 2>/dev/null | grep -q ":${port}"; then
            log_verbose "Port ${port} is listening"
            return 0
        fi
    fi

    # Try TCP connection as fallback
    if timeout 2 bash -c "echo >/dev/tcp/${NODE_HOST}/${port}" 2>/dev/null; then
        log_verbose "Port ${port} is reachable"
        return 0
    fi

    log_warning "${endpoint_name} - Port ${port} not listening or not reachable"
    return 1
}

# Header
if [[ "$JSON_OUTPUT" == "false" ]]; then
    echo ""
    echo "======================================"
    echo "  PAW Metrics Verification"
    echo "======================================"
    echo "Host: ${NODE_HOST}"
    echo "Timeout: ${TIMEOUT}s"
    echo ""
fi

# Check CometBFT metrics endpoint
log_info "Checking CometBFT consensus metrics..."
check_port_listening "$COMETBFT_PORT" "CometBFT"
check_endpoint "CometBFT" \
    "http://${NODE_HOST}:${COMETBFT_PORT}/metrics" \
    "tendermint_consensus_height,tendermint_p2p_peers,tendermint_mempool_size"

# Check Cosmos SDK API metrics endpoint
log_info "Checking Cosmos SDK API metrics..."
check_port_listening "$API_PORT" "Cosmos SDK API"
check_endpoint "Cosmos SDK API" \
    "http://${NODE_HOST}:${API_PORT}/metrics" \
    "cosmos_tx_total,cosmos_block_height"

# Check custom application metrics endpoint
log_info "Checking custom application metrics..."
check_port_listening "$APP_PORT" "Application"
check_endpoint "Application" \
    "http://${NODE_HOST}:${APP_PORT}/metrics" \
    "paw_dex_,paw_oracle_,paw_compute_"

# Output results
if [[ "$JSON_OUTPUT" == "true" ]]; then
    # JSON output
    echo "{"
    echo "  \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
    echo "  \"node_host\": \"${NODE_HOST}\","
    echo "  \"total_checks\": ${TOTAL_CHECKS},"
    echo "  \"passed\": ${PASSED_CHECKS},"
    echo "  \"failed\": ${FAILED_CHECKS},"
    echo "  \"endpoints\": {"

    first=true
    for endpoint in "${!RESULTS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi

        status="${RESULTS[$endpoint]%%:*}"
        details="${RESULTS[$endpoint]#*:}"

        echo -n "    \"${endpoint}\": {"
        echo -n "\"status\": \"${status}\", "
        echo -n "\"details\": \"${details}\""
        echo -n "}"
    done

    echo ""
    echo "  }"
    echo "}"
else
    # Human-readable summary
    echo ""
    echo "======================================"
    echo "  Summary"
    echo "======================================"
    echo "Total checks: ${TOTAL_CHECKS}"
    echo -e "${GREEN}Passed: ${PASSED_CHECKS}${NC}"
    echo -e "${RED}Failed: ${FAILED_CHECKS}${NC}"
    echo ""

    if [[ $FAILED_CHECKS -eq 0 ]]; then
        echo -e "${GREEN}✓ All metrics endpoints are accessible${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Configure Prometheus to scrape these endpoints"
        echo "  2. Import Grafana dashboards from docs/DASHBOARDS_GUIDE.md"
        echo "  3. Set up alerts using examples in docs/METRICS.md"
        echo ""
    else
        echo -e "${RED}✗ Some metrics endpoints failed${NC}"
        echo ""
        echo "Troubleshooting:"
        echo "  1. Check if the node is running: systemctl status pawd"
        echo "  2. Verify metrics are enabled in config:"
        echo "     - config.toml: [instrumentation] prometheus = true"
        echo "     - app.toml: [telemetry] enabled = true"
        echo "  3. Check firewall rules: sudo iptables -L -n | grep 2666"
        echo "  4. Review logs: journalctl -u pawd -n 100"
        echo "  5. See docs/METRICS.md for detailed troubleshooting"
        echo ""
    fi
fi

# Exit with appropriate code
if [[ $FAILED_CHECKS -eq 0 ]]; then
    exit 0
else
    exit 1
fi
