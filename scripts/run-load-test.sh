#!/bin/bash

# PAW Blockchain Comprehensive Load Test Runner
# This script runs all load testing tools and generates a consolidated report

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPORT_DIR="tests/load/reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="${REPORT_DIR}/load-test-${TIMESTAMP}.html"
JSON_REPORT="${REPORT_DIR}/load-test-${TIMESTAMP}.json"

# Test configuration
BASE_URL="${BASE_URL:-http://localhost:1317}"
RPC_URL="${RPC_URL:-http://localhost:26657}"
WS_URL="${WS_URL:-ws://localhost:26657/websocket}"
SCENARIO="${SCENARIO:-normal}"

# Ensure report directory exists
mkdir -p "${REPORT_DIR}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}PAW Blockchain Load Test Runner${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Configuration:"
echo "  API URL: ${BASE_URL}"
echo "  RPC URL: ${RPC_URL}"
echo "  WebSocket URL: ${WS_URL}"
echo "  Scenario: ${SCENARIO}"
echo "  Report Dir: ${REPORT_DIR}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if the blockchain is running
check_blockchain() {
    echo -e "${YELLOW}Checking blockchain connectivity...${NC}"

    if curl -s "${BASE_URL}/cosmos/base/tendermint/v1beta1/node_info" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ API endpoint is reachable${NC}"
    else
        echo -e "${RED}✗ API endpoint is not reachable at ${BASE_URL}${NC}"
        echo "Please start the blockchain with: make localnet-start"
        exit 1
    fi

    if curl -s "${RPC_URL}/status" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ RPC endpoint is reachable${NC}"
    else
        echo -e "${RED}✗ RPC endpoint is not reachable at ${RPC_URL}${NC}"
        exit 1
    fi

    echo ""
}

# Function to run k6 tests
run_k6_tests() {
    if ! command_exists k6; then
        echo -e "${YELLOW}⚠ k6 not found, skipping k6 tests${NC}"
        echo "Install k6: https://k6.io/docs/getting-started/installation"
        return
    fi

    echo -e "${GREEN}Running k6 tests...${NC}"

    # Blockchain load test
    echo "  → Blockchain load test..."
    BASE_URL="${BASE_URL}" RPC_URL="${RPC_URL}" \
        k6 run --out json="${REPORT_DIR}/k6-blockchain-${TIMESTAMP}.json" \
        tests/load/k6/blockchain-load-test.js || true

    # DEX load test
    echo "  → DEX load test..."
    BASE_URL="${BASE_URL}" \
        k6 run --out json="${REPORT_DIR}/k6-dex-${TIMESTAMP}.json" \
        tests/load/k6/dex-swap-test.js || true

    # WebSocket test
    echo "  → WebSocket test..."
    WS_URL="${WS_URL}" \
        k6 run --out json="${REPORT_DIR}/k6-websocket-${TIMESTAMP}.json" \
        tests/load/k6/websocket-test.js || true

    echo -e "${GREEN}✓ k6 tests completed${NC}"
    echo ""
}

# Function to run Locust tests
run_locust_tests() {
    if ! command_exists locust; then
        echo -e "${YELLOW}⚠ Locust not found, skipping Locust tests${NC}"
        echo "Install locust: pip install locust"
        return
    fi

    echo -e "${GREEN}Running Locust tests...${NC}"

    # Determine test parameters based on scenario
    case "${SCENARIO}" in
        light)
            USERS=10
            SPAWN_RATE=2
            DURATION=5m
            ;;
        normal)
            USERS=100
            SPAWN_RATE=10
            DURATION=10m
            ;;
        peak)
            USERS=500
            SPAWN_RATE=50
            DURATION=15m
            ;;
        stress)
            USERS=1000
            SPAWN_RATE=100
            DURATION=30m
            ;;
        *)
            USERS=100
            SPAWN_RATE=10
            DURATION=10m
            ;;
    esac

    echo "  → Running with ${USERS} users for ${DURATION}..."
    locust -f tests/load/locust/locustfile.py \
        --headless \
        --users ${USERS} \
        --spawn-rate ${SPAWN_RATE} \
        --run-time ${DURATION} \
        --host "${BASE_URL}" \
        --html "${REPORT_DIR}/locust-${TIMESTAMP}.html" \
        --csv "${REPORT_DIR}/locust-${TIMESTAMP}" || true

    echo -e "${GREEN}✓ Locust tests completed${NC}"
    echo ""
}

# Function to run tm-load-test
run_tm_load_test() {
    if ! command_exists tm-load-test; then
        echo -e "${YELLOW}⚠ tm-load-test not found, skipping Tendermint load test${NC}"
        echo "Install tm-load-test: go install github.com/informalsystems/tm-load-test@latest"
        return
    fi

    echo -e "${GREEN}Running Tendermint load test...${NC}"

    # Determine parameters based on scenario
    case "${SCENARIO}" in
        light)
            CONNECTIONS=5
            DURATION=60
            RATE=50
            ;;
        normal)
            CONNECTIONS=10
            DURATION=120
            RATE=100
            ;;
        peak)
            CONNECTIONS=20
            DURATION=180
            RATE=500
            ;;
        stress)
            CONNECTIONS=50
            DURATION=300
            RATE=1000
            ;;
        *)
            CONNECTIONS=10
            DURATION=120
            RATE=100
            ;;
    esac

    echo "  → Running with ${CONNECTIONS} connections, ${RATE} tx/s for ${DURATION}s..."
    tm-load-test \
        -c ${CONNECTIONS} \
        -T ${DURATION} \
        -r ${RATE} \
        -s 250 \
        --broadcast-tx-method async \
        --endpoints "${WS_URL}" \
        --stats-output json \
        > "${REPORT_DIR}/tm-load-test-${TIMESTAMP}.json" 2>&1 || true

    echo -e "${GREEN}✓ Tendermint load test completed${NC}"
    echo ""
}

# Function to run Go benchmarks
run_go_benchmarks() {
    echo -e "${GREEN}Running Go benchmarks...${NC}"

    echo "  → DEX benchmarks..."
    go test -bench=. -benchmem -benchtime=10s \
        -cpuprofile="${REPORT_DIR}/cpu-${TIMESTAMP}.prof" \
        -memprofile="${REPORT_DIR}/mem-${TIMESTAMP}.prof" \
        ./tests/benchmarks/ > "${REPORT_DIR}/go-bench-${TIMESTAMP}.txt" 2>&1 || true

    echo -e "${GREEN}✓ Go benchmarks completed${NC}"
    echo ""
}

# Function to run custom Go tester
run_custom_tester() {
    if [ ! -f "tests/load/gotester/gotester" ]; then
        echo -e "${YELLOW}Building custom Go tester...${NC}"
        cd tests/load/gotester
        go build -o gotester main.go
        cd - > /dev/null
    fi

    echo -e "${GREEN}Running custom Go load tester...${NC}"

    case "${SCENARIO}" in
        light)
            DURATION=5m
            CONCURRENCY=10
            RATE=10
            ;;
        normal)
            DURATION=10m
            CONCURRENCY=50
            RATE=100
            ;;
        peak)
            DURATION=15m
            CONCURRENCY=100
            RATE=500
            ;;
        stress)
            DURATION=30m
            CONCURRENCY=200
            RATE=1000
            ;;
        *)
            DURATION=10m
            CONCURRENCY=50
            RATE=100
            ;;
    esac

    echo "  → Mixed workload test..."
    tests/load/gotester/gotester \
        --rpc "${RPC_URL}" \
        --api "${BASE_URL}" \
        --duration ${DURATION} \
        --concurrency ${CONCURRENCY} \
        --rate ${RATE} \
        --type mixed \
        --output "${REPORT_DIR}/gotester-${TIMESTAMP}.json" || true

    echo -e "${GREEN}✓ Custom Go tester completed${NC}"
    echo ""
}

# Function to generate consolidated report
generate_report() {
    echo -e "${GREEN}Generating consolidated report...${NC}"

    cat > "${REPORT_FILE}" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>PAW Load Test Report - ${TIMESTAMP}</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 30px;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 10px;
        }
        h2 {
            color: #555;
            margin-top: 30px;
            border-left: 4px solid #4CAF50;
            padding-left: 10px;
        }
        .summary {
            background-color: #e8f5e9;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .metric {
            display: inline-block;
            margin: 10px 20px;
        }
        .metric-label {
            font-weight: bold;
            color: #666;
        }
        .metric-value {
            font-size: 24px;
            color: #4CAF50;
        }
        .test-section {
            margin: 20px 0;
            padding: 15px;
            background-color: #f9f9f9;
            border-left: 4px solid #2196F3;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #4CAF50;
            color: white;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .status-pass {
            color: green;
            font-weight: bold;
        }
        .status-fail {
            color: red;
            font-weight: bold;
        }
        .footer {
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            color: #999;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>PAW Blockchain Load Test Report</h1>

        <div class="summary">
            <h3>Test Configuration</h3>
            <div class="metric">
                <span class="metric-label">Timestamp:</span>
                <span class="metric-value">${TIMESTAMP}</span>
            </div>
            <div class="metric">
                <span class="metric-label">Scenario:</span>
                <span class="metric-value">${SCENARIO}</span>
            </div>
            <div class="metric">
                <span class="metric-label">API URL:</span>
                <span class="metric-value">${BASE_URL}</span>
            </div>
            <div class="metric">
                <span class="metric-label">RPC URL:</span>
                <span class="metric-value">${RPC_URL}</span>
            </div>
        </div>

        <h2>Test Results</h2>

        <div class="test-section">
            <h3>k6 Tests</h3>
            <p>HTTP/WebSocket load testing results</p>
            <ul>
                <li>Blockchain API Test: <a href="k6-blockchain-${TIMESTAMP}.json">View Results</a></li>
                <li>DEX Test: <a href="k6-dex-${TIMESTAMP}.json">View Results</a></li>
                <li>WebSocket Test: <a href="k6-websocket-${TIMESTAMP}.json">View Results</a></li>
            </ul>
        </div>

        <div class="test-section">
            <h3>Locust Tests</h3>
            <p>Python-based distributed load testing</p>
            <ul>
                <li>HTML Report: <a href="locust-${TIMESTAMP}.html">View Report</a></li>
                <li>CSV Data: <a href="locust-${TIMESTAMP}_stats.csv">Download CSV</a></li>
            </ul>
        </div>

        <div class="test-section">
            <h3>Tendermint Load Test</h3>
            <p>Consensus layer performance testing</p>
            <ul>
                <li>Results: <a href="tm-load-test-${TIMESTAMP}.json">View JSON</a></li>
            </ul>
        </div>

        <div class="test-section">
            <h3>Go Benchmarks</h3>
            <p>Module-specific performance benchmarks</p>
            <ul>
                <li>Benchmark Results: <a href="go-bench-${TIMESTAMP}.txt">View Results</a></li>
                <li>CPU Profile: <a href="cpu-${TIMESTAMP}.prof">Download</a></li>
                <li>Memory Profile: <a href="mem-${TIMESTAMP}.prof">Download</a></li>
            </ul>
        </div>

        <div class="test-section">
            <h3>Custom Go Tester</h3>
            <p>Blockchain-specific load testing</p>
            <ul>
                <li>Detailed Report: <a href="gotester-${TIMESTAMP}.json">View JSON</a></li>
            </ul>
        </div>

        <h2>Performance Summary</h2>
        <table>
            <tr>
                <th>Metric</th>
                <th>Target</th>
                <th>Actual</th>
                <th>Status</th>
            </tr>
            <tr>
                <td>Transactions Per Second</td>
                <td>100+</td>
                <td>-</td>
                <td class="status-pass">See individual reports</td>
            </tr>
            <tr>
                <td>Query Latency (p95)</td>
                <td>&lt; 500ms</td>
                <td>-</td>
                <td class="status-pass">See individual reports</td>
            </tr>
            <tr>
                <td>Transaction Latency (p95)</td>
                <td>&lt; 2s</td>
                <td>-</td>
                <td class="status-pass">See individual reports</td>
            </tr>
            <tr>
                <td>Error Rate</td>
                <td>&lt; 1%</td>
                <td>-</td>
                <td class="status-pass">See individual reports</td>
            </tr>
        </table>

        <h2>Recommendations</h2>
        <ul>
            <li>Review individual test reports for detailed metrics</li>
            <li>Check CPU and memory profiles for optimization opportunities</li>
            <li>Monitor error logs for any issues during the test</li>
            <li>Compare results with baseline for performance regression detection</li>
        </ul>

        <div class="footer">
            Generated by PAW Load Test Runner on $(date)<br>
            Report location: ${REPORT_FILE}
        </div>
    </div>
</body>
</html>
EOF

    echo -e "${GREEN}✓ Report generated: ${REPORT_FILE}${NC}"
    echo ""
}

# Main execution
main() {
    check_blockchain
    run_k6_tests
    run_locust_tests
    run_tm_load_test
    run_go_benchmarks
    run_custom_tester
    generate_report

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Load testing completed!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Reports available in: ${REPORT_DIR}"
    echo "Main report: ${REPORT_FILE}"
    echo ""
    echo "To view the report, open: file://${PWD}/${REPORT_FILE}"
}

# Run main function
main
