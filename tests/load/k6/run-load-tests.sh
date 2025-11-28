#!/bin/bash
# Run all K6 load tests for PAW Chain

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/../../../test-results/load-tests"
BASE_URL="${BASE_URL:-http://localhost:1317}"

mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}======================================"
echo "PAW Chain Load Testing Suite"
echo "======================================${NC}"
echo ""
echo "Target: $BASE_URL"
echo "Results Directory: $RESULTS_DIR"
echo ""

# Check if K6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}Error: K6 is not installed.${NC}"
    echo "Please run: ./setup.sh"
    exit 1
fi

# Function to run a load test
run_load_test() {
    local test_name=$1
    local test_file=$2

    echo -e "${GREEN}Running $test_name load test...${NC}"

    k6 run \
        --out json="$RESULTS_DIR/${test_name}-results.json" \
        --summary-export="$RESULTS_DIR/${test_name}-summary.json" \
        -e BASE_URL="$BASE_URL" \
        "$test_file"

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $test_name load test passed${NC}"
    else
        echo -e "${RED}✗ $test_name load test failed${NC}"
        return 1
    fi

    echo ""
}

# Run all load tests
TESTS_PASSED=0
TESTS_FAILED=0

if run_load_test "dex" "$SCRIPT_DIR/dex-load-test.js"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

if run_load_test "oracle" "$SCRIPT_DIR/oracle-load-test.js"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

if run_load_test "compute" "$SCRIPT_DIR/compute-load-test.js"; then
    ((TESTS_PASSED++))
else
    ((TESTS_FAILED++))
fi

# Generate HTML report
echo -e "${BLUE}Generating HTML reports...${NC}"

for test in dex oracle compute; do
    if [ -f "$RESULTS_DIR/${test}-results.json" ]; then
        # Create HTML report
        cat > "$RESULTS_DIR/${test}-report.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>K6 Load Test Report - %%TEST_NAME%%</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .metric { background: #f9f9f9; padding: 15px; margin: 10px 0; border-left: 4px solid #4CAF50; }
        .metric-name { font-weight: bold; color: #333; }
        .metric-value { color: #666; margin-left: 10px; }
        .success { color: #4CAF50; }
        .warning { color: #FF9800; }
        .error { color: #F44336; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #4CAF50; color: white; }
        tr:hover { background-color: #f5f5f5; }
    </style>
</head>
<body>
    <div class="container">
        <h1>K6 Load Test Report: %%TEST_NAME%%</h1>
        <p>Target: %%BASE_URL%%</p>
        <p>Timestamp: %%TIMESTAMP%%</p>

        <h2>Summary</h2>
        <div id="summary">Loading...</div>

        <h2>Detailed Metrics</h2>
        <div id="metrics">Loading...</div>
    </div>

    <script>
        // Load and display summary
        fetch('%%TEST_NAME%%-summary.json')
            .then(r => r.json())
            .then(data => {
                document.getElementById('summary').innerHTML = `
                    <div class="metric">
                        <span class="metric-name">Total Requests:</span>
                        <span class="metric-value">${data.metrics.http_reqs?.values?.count || 0}</span>
                    </div>
                    <div class="metric">
                        <span class="metric-name">Failed Requests:</span>
                        <span class="metric-value ${(data.metrics.http_req_failed?.values?.rate || 0) > 0.05 ? 'error' : 'success'}">
                            ${((data.metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2)}%
                        </span>
                    </div>
                    <div class="metric">
                        <span class="metric-name">Avg Response Time:</span>
                        <span class="metric-value">${(data.metrics.http_req_duration?.values?.avg || 0).toFixed(2)}ms</span>
                    </div>
                    <div class="metric">
                        <span class="metric-name">P95 Response Time:</span>
                        <span class="metric-value">${(data.metrics.http_req_duration?.values?.['p(95)'] || 0).toFixed(2)}ms</span>
                    </div>
                `;
            })
            .catch(err => {
                document.getElementById('summary').innerHTML = '<p class="error">Failed to load summary data</p>';
            });
    </script>
</body>
</html>
EOF

        # Replace placeholders
        sed -i "s/%%TEST_NAME%%/${test}/g" "$RESULTS_DIR/${test}-report.html"
        sed -i "s#%%BASE_URL%%#${BASE_URL}#g" "$RESULTS_DIR/${test}-report.html"
        sed -i "s/%%TIMESTAMP%%/$(date -u +%Y-%m-%d\ %H:%M:%S\ UTC)/g" "$RESULTS_DIR/${test}-report.html"

        echo -e "${GREEN}✓ Generated HTML report for $test${NC}"
    fi
done

# Summary
echo ""
echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}Load Testing Summary${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"
echo ""
echo "Results saved to: $RESULTS_DIR"
echo "View HTML reports in your browser:"
for test in dex oracle compute; do
    if [ -f "$RESULTS_DIR/${test}-report.html" ]; then
        echo "  - file://$RESULTS_DIR/${test}-report.html"
    fi
done
echo ""

if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}✗ Some load tests failed${NC}"
    exit 1
else
    echo -e "${GREEN}✓ All load tests passed!${NC}"
    exit 0
fi
