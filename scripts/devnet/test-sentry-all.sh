#!/bin/bash
# PAW Sentry Architecture - Complete Test Suite Runner
# Runs all sentry testing scenarios and generates comprehensive report

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
REPORT_FILE="/tmp/paw-sentry-test-report-${TIMESTAMP}.txt"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    echo "[INFO] $1" >> "$REPORT_FILE"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    echo "[SUCCESS] $1" >> "$REPORT_FILE"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    echo "[ERROR] $1" >> "$REPORT_FILE"
}

log_section() {
    echo ""
    echo -e "${BOLD}==============================================  ${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${BOLD}=============================================="
    echo ""
    echo "" >> "$REPORT_FILE"
    echo "==============================================" >> "$REPORT_FILE"
    echo "  $1" >> "$REPORT_FILE"
    echo "==============================================" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
}

# Initialize report
init_report() {
    cat > "$REPORT_FILE" <<EOF
PAW Sentry Architecture - Test Report
Generated: $(date)
=============================================="

Test Configuration:
- 4 Validator Nodes (node1-4)
- 2 Sentry Nodes (sentry1-2)
- Network: pawnet (172.22.0.0/24)

=============================================="

EOF
}

# Run a test suite and capture results
run_test_suite() {
    local script=$1
    local name=$2
    local suite_start=$(date +%s)

    log_section "$name"

    if [ ! -x "$script" ]; then
        log_error "Test script not found or not executable: $script"
        return 1
    fi

    # Run the test suite
    if "$script" >> "$REPORT_FILE" 2>&1; then
        local suite_end=$(date +%s)
        local suite_duration=$((suite_end - suite_start))
        log_success "$name completed in ${suite_duration}s"
        return 0
    else
        local suite_end=$(date +%s)
        local suite_duration=$((suite_end - suite_start))
        log_error "$name failed after ${suite_duration}s"
        return 1
    fi
}

# Verify network state
verify_network() {
    log_section "Network Pre-Check"

    # Check if network is running
    if ! docker ps | grep -q "paw-node1"; then
        log_error "Network not running!"
        log_info "Start network with: docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d"
        exit 1
    fi
    log_success "Network containers are running"

    # Check validator count
    local validator_count=$(curl -s http://localhost:26657/validators 2>/dev/null | jq -r '.result.total' 2>/dev/null)
    if [ "$validator_count" = "4" ]; then
        log_success "All 4 validators active"
    else
        log_error "Expected 4 validators, found: $validator_count"
        exit 1
    fi

    # Check block production
    local height1=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    sleep 5
    local height2=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ "$height2" -gt "$height1" ]; then
        log_success "Blockchain producing blocks (height: $height1 → $height2)"
    else
        log_error "Blockchain not producing blocks (height stuck at $height1)"
        exit 1
    fi

    # Check sentry status
    local sentry1_status=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
    local sentry2_status=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)

    if [ "$sentry1_status" = "paw-devnet" ] && [ "$sentry2_status" = "paw-devnet" ]; then
        log_success "Both sentries responding"
    else
        log_error "Sentries not responding correctly"
        exit 1
    fi

    # Check sentry peer counts
    local sentry1_peers=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)
    local sentry2_peers=$(curl -s http://localhost:30668/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)

    if [ "$sentry1_peers" = "5" ] && [ "$sentry2_peers" = "5" ]; then
        log_success "Sentries have correct peer counts (5 each: 4 validators + 1 sentry)"
    else
        log_error "Incorrect peer counts (sentry1: $sentry1_peers, sentry2: $sentry2_peers, expected 5)"
        exit 1
    fi

    log_success "Network pre-check passed"
}

# Generate summary
generate_summary() {
    log_section "Test Summary"

    local total_suites=$1
    local passed_suites=$2
    local failed_suites=$3

    cat >> "$REPORT_FILE" <<EOF

Test Suites Summary:
- Total suites: $total_suites
- Passed: $passed_suites
- Failed: $failed_suites

EOF

    if [ $failed_suites -eq 0 ]; then
        log_success "All test suites passed!"
        echo -e "${GREEN}${BOLD}"
        cat <<'EOF'
   _____ _    _  _____ _____ ______  _____ _____
  / ____| |  | |/ ____/ ____|  ____|/ ____/ ____|
 | (___ | |  | | |   | |    | |__  | (___| (___
  \___ \| |  | | |   | |    |  __|  \___ \\___ \
  ____) | |__| | |___| |____| |____ ____) |___) |
 |_____/ \____/ \_____\_____|______|_____/_____/

EOF
        echo -e "${NC}"
        log_success "All sentry architecture tests passed"
    else
        log_error "Some test suites failed"
        echo -e "${YELLOW}Please review the test report: ${REPORT_FILE}${NC}"
    fi

    echo ""
    echo "Full report saved to: $REPORT_FILE"
    echo ""
}

# Main execution
main() {
    local total_suites=0
    local passed_suites=0
    local failed_suites=0

    clear
    echo ""
    echo -e "${BOLD}${BLUE}"
    cat <<'EOF'
 ____   __        __
|  _ \ / /\\      \ \    ___  __ _ __   __
| |_) / /__\\ _____\ \  / __|/ _\` |\ \ / /
|  __/ \_  _//_____/ / \__ \ (_| | \ V /
|_|     /_/        /_/  |___/\__,_|  \_/

   Sentry Architecture Test Suite
=============================================="
EOF
    echo -e "${NC}"
    echo "Comprehensive testing of production-like"
    echo "sentry node architecture with 4 validators"
    echo "and 2 sentry nodes."
    echo ""
    echo "Report will be saved to: $REPORT_FILE"
    echo ""

    # Initialize report
    init_report

    # Verify network state
    verify_network

    # Test Suite 1: Basic Scenarios
    total_suites=$((total_suites + 1))
    if run_test_suite "$SCRIPT_DIR/test-sentry-scenarios.sh" "Test Suite 1: Basic Sentry Scenarios"; then
        passed_suites=$((passed_suites + 1))
    else
        failed_suites=$((failed_suites + 1))
        log_error "Basic scenarios test failed - aborting remaining tests"
        generate_summary $total_suites $passed_suites $failed_suites
        exit 1
    fi

    # Test Suite 2: Load Distribution
    total_suites=$((total_suites + 1))
    if run_test_suite "$SCRIPT_DIR/test-load-distribution.sh" "Test Suite 2: Load Distribution"; then
        passed_suites=$((passed_suites + 1))
    else
        failed_suites=$((failed_suites + 1))
    fi

    # Test Suite 3: Network Chaos
    log_section "Test Suite 3: Network Chaos"
    echo -e "${YELLOW}WARNING: This test will temporarily disrupt network connectivity${NC}"
    echo "Network will automatically recover after each test"
    echo ""

    if [ "$1" != "--skip-chaos" ]; then
        total_suites=$((total_suites + 1))
        if run_test_suite "$SCRIPT_DIR/test-network-chaos.sh" "Test Suite 3: Network Chaos"; then
            passed_suites=$((passed_suites + 1))
        else
            failed_suites=$((failed_suites + 1))
        fi
    else
        log_info "Skipping chaos tests (--skip-chaos flag provided)"
    fi

    # Generate summary
    generate_summary $total_suites $passed_suites $failed_suites

    # Exit code
    if [ $failed_suites -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Run all tests
main "$@"
