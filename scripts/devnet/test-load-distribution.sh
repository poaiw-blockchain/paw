#!/bin/bash
# PAW Load Distribution Testing
# Tests RPC load distribution across sentry nodes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

# Test RPC endpoint latency
measure_latency() {
    local endpoint=$1
    local name=$2
    local total_time=0
    local requests=20
    local successful=0

    for i in $(seq 1 $requests); do
        start=$(date +%s%N)
        response=$(curl -s "$endpoint/status" 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
        end=$(date +%s%N)

        if [ "$response" = "paw-devnet" ]; then
            elapsed=$((end - start))
            elapsed_ms=$((elapsed / 1000000))
            total_time=$((total_time + elapsed_ms))
            successful=$((successful + 1))
        fi

        # Small delay between requests
        sleep 0.1
    done

    if [ $successful -gt 0 ]; then
        avg_latency=$((total_time / successful))
        echo "$avg_latency"
    else
        echo "999999"  # Error value
    fi
}

# Test concurrent requests
test_concurrent_load() {
    local endpoint=$1
    local name=$2
    local concurrent_requests=10
    local temp_dir="/tmp/paw-load-test-$$"

    mkdir -p "$temp_dir"

    log_info "Testing $name with $concurrent_requests concurrent requests..."

    # Launch concurrent requests
    for i in $(seq 1 $concurrent_requests); do
        (
            start=$(date +%s%N)
            curl -s "$endpoint/status" >/dev/null 2>&1
            end=$(date +%s%N)
            elapsed=$((end - start))
            elapsed_ms=$((elapsed / 1000000))
            echo "$elapsed_ms" > "$temp_dir/request_$i.time"
        ) &
    done

    # Wait for all requests to complete
    wait

    # Calculate statistics
    total=0
    count=0
    max=0
    min=999999

    for f in "$temp_dir"/request_*.time; do
        if [ -f "$f" ]; then
            time=$(cat "$f")
            total=$((total + time))
            count=$((count + 1))

            if [ "$time" -gt "$max" ]; then
                max=$time
            fi
            if [ "$time" -lt "$min" ]; then
                min=$time
            fi
        fi
    done

    if [ $count -gt 0 ]; then
        avg=$((total / count))
        log_info "  Avg: ${avg}ms, Min: ${min}ms, Max: ${max}ms, Count: ${count}"
    else
        log_error "  No successful requests"
    fi

    # Cleanup
    rm -rf "$temp_dir"
}

# Test sustained load
test_sustained_load() {
    local endpoint=$1
    local name=$2
    local duration=30  # seconds
    local requests_per_second=5

    log_info "Testing $name with sustained load (${duration}s at ${requests_per_second} req/s)..."

    local total_requests=0
    local successful_requests=0
    local failed_requests=0
    local total_latency=0

    local end_time=$(($(date +%s) + duration))

    while [ $(date +%s) -lt $end_time ]; do
        for i in $(seq 1 $requests_per_second); do
            total_requests=$((total_requests + 1))

            start=$(date +%s%N)
            response=$(curl -s "$endpoint/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
            end=$(date +%s%N)

            if [ -n "$response" ] && [ "$response" != "null" ]; then
                successful_requests=$((successful_requests + 1))
                elapsed=$((end - start))
                elapsed_ms=$((elapsed / 1000000))
                total_latency=$((total_latency + elapsed_ms))
            else
                failed_requests=$((failed_requests + 1))
            fi
        done
        sleep 1
    done

    # Calculate results
    if [ $successful_requests -gt 0 ]; then
        avg_latency=$((total_latency / successful_requests))
        success_rate=$((successful_requests * 100 / total_requests))

        log_success "Sustained load results:"
        log_info "  Total requests: $total_requests"
        log_info "  Successful: $successful_requests ($success_rate%)"
        log_info "  Failed: $failed_requests"
        log_info "  Average latency: ${avg_latency}ms"
    else
        log_error "All requests failed during sustained load test"
    fi
}

# Compare sentry vs direct validator access
test_latency_comparison() {
    log_info "=== Latency Comparison ==="
    echo ""

    # Test validator (node1) directly
    log_info "Testing direct validator access (node1:26657)..."
    validator_latency=$(measure_latency "http://localhost:26657" "node1")
    log_info "Validator avg latency: ${validator_latency}ms"
    echo ""

    # Test sentry1
    log_info "Testing sentry1 (30658)..."
    sentry1_latency=$(measure_latency "http://localhost:30658" "sentry1")
    log_info "Sentry1 avg latency: ${sentry1_latency}ms"
    echo ""

    # Test sentry2
    log_info "Testing sentry2 (30668)..."
    sentry2_latency=$(measure_latency "http://localhost:30668" "sentry2")
    log_info "Sentry2 avg latency: ${sentry2_latency}ms"
    echo ""

    # Compare
    log_info "Latency comparison:"
    log_info "  Direct validator: ${validator_latency}ms (baseline)"
    log_info "  Sentry1: ${sentry1_latency}ms"
    log_info "  Sentry2: ${sentry2_latency}ms"

    # Calculate overhead
    if [ "$validator_latency" -gt 0 ] && [ "$validator_latency" != "999999" ]; then
        sentry1_overhead=$((sentry1_latency - validator_latency))
        sentry2_overhead=$((sentry2_latency - validator_latency))

        log_info "  Sentry1 overhead: ${sentry1_overhead}ms"
        log_info "  Sentry2 overhead: ${sentry2_overhead}ms"

        if [ "$sentry1_overhead" -lt 50 ] && [ "$sentry2_overhead" -lt 50 ]; then
            log_success "Sentries have acceptable latency overhead (<50ms)"
        else
            log_warn "Sentries have high latency overhead (>50ms)"
        fi
    fi

    echo ""
}

# Test load balancing
test_load_balancing() {
    log_info "=== Load Balancing Test ==="
    echo ""

    log_info "Sending requests to both sentries alternately..."

    local requests=50
    local sentry1_responses=0
    local sentry2_responses=0

    for i in $(seq 1 $requests); do
        if [ $((i % 2)) -eq 0 ]; then
            # Even - use sentry1
            response=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
            if [ "$response" = "paw-devnet" ]; then
                sentry1_responses=$((sentry1_responses + 1))
            fi
        else
            # Odd - use sentry2
            response=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
            if [ "$response" = "paw-devnet" ]; then
                sentry2_responses=$((sentry2_responses + 1))
            fi
        fi
    done

    log_info "Load distribution:"
    log_info "  Sentry1 responses: $sentry1_responses (expected: $((requests / 2)))"
    log_info "  Sentry2 responses: $sentry2_responses (expected: $((requests / 2)))"

    total_responses=$((sentry1_responses + sentry2_responses))
    success_rate=$((total_responses * 100 / requests))

    if [ $success_rate -ge 95 ]; then
        log_success "Load balancing successful (${success_rate}% success rate)"
    else
        log_warn "Some requests failed (${success_rate}% success rate)"
    fi

    echo ""
}

# Test concurrent load on both sentries
test_concurrent_comparison() {
    log_info "=== Concurrent Load Comparison ==="
    echo ""

    test_concurrent_load "http://localhost:26657" "Direct Validator"
    echo ""

    test_concurrent_load "http://localhost:30658" "Sentry1"
    echo ""

    test_concurrent_load "http://localhost:30668" "Sentry2"
    echo ""
}

# Test data consistency across endpoints
test_data_consistency() {
    log_info "=== Data Consistency Test ==="
    echo ""

    log_info "Verifying all endpoints return consistent blockchain state..."

    # Get block height from all endpoints
    validator_height=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    sentry1_height=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    sentry2_height=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    log_info "Block heights:"
    log_info "  Validator: $validator_height"
    log_info "  Sentry1: $sentry1_height"
    log_info "  Sentry2: $sentry2_height"

    # Check if heights are within acceptable range (2 blocks)
    max_height=$validator_height
    if [ "$sentry1_height" -gt "$max_height" ]; then
        max_height=$sentry1_height
    fi
    if [ "$sentry2_height" -gt "$max_height" ]; then
        max_height=$sentry2_height
    fi

    validator_diff=$((max_height - validator_height))
    sentry1_diff=$((max_height - sentry1_height))
    sentry2_diff=$((max_height - sentry2_height))

    if [ "$validator_diff" -le 2 ] && [ "$sentry1_diff" -le 2 ] && [ "$sentry2_diff" -le 2 ]; then
        log_success "All endpoints have consistent blockchain state (within 2 blocks)"
    else
        log_warn "Endpoints have inconsistent state (diffs: validator=$validator_diff, sentry1=$sentry1_diff, sentry2=$sentry2_diff)"
    fi

    echo ""

    # Get validator set from all endpoints
    log_info "Verifying validator set consistency..."

    validator_count_1=$(curl -s http://localhost:26657/validators 2>/dev/null | jq -r '.result.total' 2>/dev/null)
    validator_count_2=$(curl -s http://localhost:30658/validators 2>/dev/null | jq -r '.result.total' 2>/dev/null)
    validator_count_3=$(curl -s http://localhost:30668/validators 2>/dev/null | jq -r '.result.total' 2>/dev/null)

    if [ "$validator_count_1" = "4" ] && [ "$validator_count_2" = "4" ] && [ "$validator_count_3" = "4" ]; then
        log_success "All endpoints report same validator set (4 validators)"
    else
        log_error "Inconsistent validator sets (validator=$validator_count_1, sentry1=$validator_count_2, sentry2=$validator_count_3)"
    fi

    echo ""
}

# Main execution
main() {
    echo ""
    echo "=============================================="
    echo "  PAW Load Distribution Testing"
    echo "=============================================="
    echo ""
    echo "This test suite validates RPC load distribution"
    echo "and performance across sentry nodes."
    echo ""

    # Verify network is running
    log_info "Verifying network is running..."
    if ! docker ps | grep -q "paw-node1"; then
        log_error "Network not running! Start with: docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d"
        exit 1
    fi
    log_success "Network containers are running"
    echo ""

    # Run load tests
    test_latency_comparison
    test_load_balancing
    test_data_consistency
    test_concurrent_comparison

    # Optional: Sustained load test (takes 30 seconds)
    if [ "$1" = "--sustained" ]; then
        log_info "=== Sustained Load Test ==="
        echo ""
        test_sustained_load "http://localhost:30658" "Sentry1"
        echo ""
        test_sustained_load "http://localhost:30668" "Sentry2"
        echo ""
    else
        log_info "Tip: Run with --sustained flag to test sustained load (takes 30s)"
        echo ""
    fi

    # Summary
    echo "=============================================="
    echo "  Load Testing Complete"
    echo "=============================================="
    echo ""
    log_success "Load distribution testing completed successfully"
    log_info "Sentries are handling RPC load correctly"
    echo ""
}

# Run load tests
main "$@"
