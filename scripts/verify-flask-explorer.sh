#!/bin/bash
#
# Verify Flask Explorer Health and Functionality
#
# This script performs comprehensive health checks on the Flask blockchain explorer,
# verifying that all endpoints are responsive and returning valid data.
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXPLORER_URL="${EXPLORER_URL:-http://localhost:11080}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"

    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")

    if [ "$response" = "$expected_status" ]; then
        log_pass "$name: HTTP $response"
        return 0
    else
        log_fail "$name: Expected HTTP $expected_status, got HTTP $response"
        return 1
    fi
}

test_api_endpoint() {
    local name="$1"
    local url="$2"
    local check_field="$3"

    local response
    response=$(curl -s "$url" 2>/dev/null || echo "{}")

    if echo "$response" | grep -q "$check_field"; then
        log_pass "$name: API returned valid data"
        return 0
    else
        log_fail "$name: API did not return expected field '$check_field'"
        echo "Response: $response"
        return 1
    fi
}

echo "======================================"
echo "Flask Explorer Health Check"
echo "======================================"
echo ""
log_info "Testing Flask explorer at: $EXPLORER_URL"
echo ""

# Check if container is running
log_info "Checking Docker container status..."
if docker ps | grep -q paw-flask-explorer; then
    log_pass "Docker container is running"
else
    log_fail "Docker container is not running"
    log_error "Start the explorer with: $SCRIPT_DIR/deploy-flask-explorer.sh"
    exit 1
fi

echo ""
log_info "Testing Web UI Endpoints..."
echo ""

# Test dashboard
test_endpoint "Dashboard" "$EXPLORER_URL/"

# Test validators page
test_endpoint "Validators Page" "$EXPLORER_URL/validators"

# Test search page
test_endpoint "Search Page" "$EXPLORER_URL/search"

# Test search with query
test_endpoint "Search with query" "$EXPLORER_URL/search?q=1"

echo ""
log_info "Testing API Endpoints..."
echo ""

# Test API status
test_api_endpoint "API Status" "$EXPLORER_URL/api/status" "sync_info"

# Test API validators
test_api_endpoint "API Validators" "$EXPLORER_URL/api/validators" "validators"

# Test API block (block 1)
test_api_endpoint "API Block" "$EXPLORER_URL/api/block/1" "block"

echo ""
log_info "Testing RPC Integration..."
echo ""

# Get latest block from status
LATEST_BLOCK=$(curl -s "$EXPLORER_URL/api/status" 2>/dev/null | grep -o '"latest_block_height":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -n "$LATEST_BLOCK" ] && [ "$LATEST_BLOCK" != "0" ]; then
    log_pass "RPC integration working (latest block: $LATEST_BLOCK)"

    # Test fetching the latest block
    test_api_endpoint "Fetch Latest Block" "$EXPLORER_URL/api/block/$LATEST_BLOCK" "block"
else
    log_fail "RPC integration: Unable to fetch latest block height"
fi

echo ""
log_info "Checking Docker Logs for Errors..."
echo ""

ERROR_COUNT=$(docker logs paw-flask-explorer 2>&1 | grep -i "error" | wc -l || echo "0")
if [ "$ERROR_COUNT" -eq 0 ]; then
    log_pass "No errors found in logs"
else
    log_warn "Found $ERROR_COUNT error messages in logs"
    log_info "View logs with: docker logs paw-flask-explorer"
fi

echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    log_info "All tests passed! Flask explorer is healthy."
    echo ""
    log_info "Access the explorer at: $EXPLORER_URL"
    exit 0
else
    log_error "Some tests failed. Please check the output above."
    log_info "View container logs: docker logs paw-flask-explorer"
    exit 1
fi
