#!/usr/bin/env bash
# test-multinode.sh - Main orchestrator for Phase 3: Multi-Node Network & Consensus Testing
# This script coordinates all Phase 3 tests for the PAW blockchain
#
# Usage:
#   ./scripts/test-multinode.sh [phase]
#   ./scripts/test-multinode.sh all           # Run all Phase 3 tests
#   ./scripts/test-multinode.sh 3.1           # Run only Phase 3.1
#   ./scripts/test-multinode.sh 3.2           # Run only Phase 3.2
#   ./scripts/test-multinode.sh 3.3           # Run only Phase 3.3
#   ./scripts/test-multinode.sh 3.4           # Run only Phase 3.4
#   ./scripts/test-multinode.sh cleanup       # Cleanup all resources

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPORT_DIR="${PROJECT_ROOT}/test-reports/phase3"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="${REPORT_DIR}/multinode_test_${TIMESTAMP}.txt"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Test results tracking
declare -A TEST_RESULTS
declare -A TEST_DURATIONS

# Logging functions
log() {
    local level=$1; shift
    local color=$NC
    case "$level" in
        INFO)  color=$BLUE ;;
        PASS)  color=$GREEN ;;
        FAIL)  color=$RED ;;
        WARN)  color=$YELLOW ;;
        SECTION) color=$CYAN ;;
    esac
    echo -e "${color}[$(date +'%Y-%m-%d %H:%M:%S')] [$level] $*${NC}" | tee -a "$REPORT_FILE"
}

log_section() {
    echo "" | tee -a "$REPORT_FILE"
    echo -e "${CYAN}${BOLD}========================================${NC}" | tee -a "$REPORT_FILE"
    echo -e "${CYAN}${BOLD}$*${NC}" | tee -a "$REPORT_FILE"
    echo -e "${CYAN}${BOLD}========================================${NC}" | tee -a "$REPORT_FILE"
    echo "" | tee -a "$REPORT_FILE"
}

# Create report directory
mkdir -p "$REPORT_DIR"

# Initialize report
cat > "$REPORT_FILE" <<EOF
================================================================================
PAW Blockchain - Phase 3: Multi-Node Network & Consensus Testing Report
================================================================================
Test Run: ${TIMESTAMP}
Project Root: ${PROJECT_ROOT}

EOF

# Cleanup function
cleanup_all() {
    log INFO "Performing full cleanup..."

    # Run individual phase cleanup scripts
    for script in phase3.1-devnet-baseline.sh phase3.2-consensus-liveness.sh \
                  phase3.3-network-conditions.sh phase3.4-malicious-peer.sh; do
        if [[ -x "${SCRIPT_DIR}/${script}" ]]; then
            log INFO "Running cleanup for ${script}..."
            "${SCRIPT_DIR}/${script}" cleanup 2>&1 | tee -a "$REPORT_FILE" || true
        fi
    done

    # Global Docker cleanup
    log INFO "Cleaning up Docker resources..."
    docker compose -f "${PROJECT_ROOT}/compose/docker-compose.devnet.yml" down -v 2>&1 | tee -a "$REPORT_FILE" || true

    # Reset network conditions on all interfaces
    log INFO "Resetting network conditions..."
    if [[ $EUID -eq 0 ]]; then
        for container in paw-node1 paw-node2 paw-node3 paw-node4; do
            if docker ps --format '{{.Names}}' | grep -qx "$container"; then
                sudo ~/blockchain-projects/scripts/network-sim.sh reset "$container" 2>&1 | tee -a "$REPORT_FILE" || true
            fi
        done
    fi

    log PASS "Cleanup completed"
}

# Trap for cleanup on exit
trap cleanup_all EXIT INT TERM

# Run individual test phase
run_phase() {
    local phase=$1
    local script_name=$2
    local description=$3

    log_section "Phase ${phase}: ${description}"

    local start_time=$(date +%s)
    local script_path="${SCRIPT_DIR}/${script_name}"

    if [[ ! -x "$script_path" ]]; then
        log FAIL "Script not found or not executable: ${script_path}"
        TEST_RESULTS[$phase]="SKIP"
        return 1
    fi

    log INFO "Starting ${script_name}..."

    if "$script_path" 2>&1 | tee -a "$REPORT_FILE"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        TEST_RESULTS[$phase]="PASS"
        TEST_DURATIONS[$phase]=$duration
        log PASS "Phase ${phase} completed successfully in ${duration}s"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        TEST_RESULTS[$phase]="FAIL"
        TEST_DURATIONS[$phase]=$duration
        log FAIL "Phase ${phase} failed after ${duration}s"
        return 1
    fi
}

# Generate summary report
generate_summary() {
    log_section "Test Summary"

    local total=0
    local passed=0
    local failed=0
    local skipped=0
    local total_duration=0

    for phase in "${!TEST_RESULTS[@]}"; do
        total=$((total + 1))
        local result="${TEST_RESULTS[$phase]}"
        local duration="${TEST_DURATIONS[$phase]:-0}"
        total_duration=$((total_duration + duration))

        case "$result" in
            PASS)
                passed=$((passed + 1))
                log PASS "Phase ${phase}: PASSED (${duration}s)"
                ;;
            FAIL)
                failed=$((failed + 1))
                log FAIL "Phase ${phase}: FAILED (${duration}s)"
                ;;
            SKIP)
                skipped=$((skipped + 1))
                log WARN "Phase ${phase}: SKIPPED"
                ;;
        esac
    done

    echo "" | tee -a "$REPORT_FILE"
    echo "Total Tests:    $total" | tee -a "$REPORT_FILE"
    echo "Passed:         $passed" | tee -a "$REPORT_FILE"
    echo "Failed:         $failed" | tee -a "$REPORT_FILE"
    echo "Skipped:        $skipped" | tee -a "$REPORT_FILE"
    echo "Total Duration: ${total_duration}s" | tee -a "$REPORT_FILE"
    echo "" | tee -a "$REPORT_FILE"

    if [[ $failed -gt 0 ]]; then
        log FAIL "Some tests failed. See report: ${REPORT_FILE}"
        return 1
    else
        log PASS "All tests passed!"
        return 0
    fi
}

# Show usage
show_usage() {
    cat <<EOF
PAW Multi-Node Network & Consensus Testing Suite

Usage:
  $0 [phase]

Phases:
  all      Run all Phase 3 tests (default)
  3.1      4-Node Devnet Baseline
  3.2      Consensus Liveness & Halt
  3.3      Network Variable Latency/Bandwidth
  3.4      Malicious Peer Ejection
  cleanup  Clean up all test resources

Examples:
  $0              # Run all tests
  $0 all          # Run all tests
  $0 3.2          # Run only Phase 3.2
  $0 cleanup      # Cleanup only

Environment Variables:
  SKIP_BUILD=1           Skip rebuilding binaries
  KEEP_RUNNING=1         Keep network running after tests
  VERBOSE=1              Enable verbose output

Report Location:
  ${REPORT_DIR}/

EOF
}

# Main execution
main() {
    local phase="${1:-all}"

    case "$phase" in
        cleanup)
            cleanup_all
            exit 0
            ;;
        help|--help|-h)
            show_usage
            exit 0
            ;;
    esac

    log_section "PAW Phase 3: Multi-Node Network & Consensus Testing"
    log INFO "Test suite started"
    log INFO "Report will be saved to: ${REPORT_FILE}"

    # Check prerequisites
    log INFO "Checking prerequisites..."
    local missing_deps=()
    for cmd in docker jq curl tc; do
        if ! command -v "$cmd" &>/dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log FAIL "Missing required dependencies: ${missing_deps[*]}"
        exit 1
    fi

    # Check if we need sudo for tc commands
    if [[ $EUID -ne 0 ]] && [[ "$phase" == "all" || "$phase" == "3.3" ]]; then
        log WARN "Phase 3.3 requires sudo privileges for network simulation"
        log WARN "You may be prompted for password during Phase 3.3"
    fi

    log PASS "Prerequisites check passed"

    # Run requested phase(s)
    case "$phase" in
        all)
            run_phase "3.1" "phase3.1-devnet-baseline.sh" "4-Node Devnet Baseline"
            run_phase "3.2" "phase3.2-consensus-liveness.sh" "Consensus Liveness & Halt"
            run_phase "3.3" "phase3.3-network-conditions.sh" "Network Variable Latency/Bandwidth"
            run_phase "3.4" "phase3.4-malicious-peer.sh" "Malicious Peer Ejection"
            ;;
        3.1)
            run_phase "3.1" "phase3.1-devnet-baseline.sh" "4-Node Devnet Baseline"
            ;;
        3.2)
            run_phase "3.2" "phase3.2-consensus-liveness.sh" "Consensus Liveness & Halt"
            ;;
        3.3)
            run_phase "3.3" "phase3.3-network-conditions.sh" "Network Variable Latency/Bandwidth"
            ;;
        3.4)
            run_phase "3.4" "phase3.4-malicious-peer.sh" "Malicious Peer Ejection"
            ;;
        *)
            log FAIL "Unknown phase: $phase"
            show_usage
            exit 1
            ;;
    esac

    # Generate summary
    generate_summary
}

main "$@"
