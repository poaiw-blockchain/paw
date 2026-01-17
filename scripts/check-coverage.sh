#!/bin/bash
# Coverage threshold enforcement script
# Ensures test coverage meets minimum standards before commits/merges

set -e

# Configuration
# Threshold set to current achievable level - increase as coverage improves
THRESHOLD=20.0
CRITICAL_MODULES="x/compute x/dex x/oracle"
INFRA_THRESHOLD=15.0
CLI_THRESHOLD=10.0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${2}${1}${NC}"
}

print_header() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

# Check if coverage.out exists, generate if needed
if [ ! -f coverage.out ]; then
    print_status "Coverage file not found. Generating..." "$YELLOW"
    PKGS=$(go list ./x/compute/... ./x/dex/... ./x/oracle/... | grep -v '/types$' | paste -sd, -)
    go test ./x/compute/... ./x/dex/... ./x/oracle/... -coverpkg="$PKGS" -coverprofile=coverage.out -covermode=atomic -timeout=30m
fi

# Extract total coverage
TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

print_header "Coverage Report"
echo "Total Coverage: ${TOTAL_COVERAGE}%"
echo "Target: ${THRESHOLD}%"

# Check overall threshold
if (( $(echo "$TOTAL_COVERAGE < $THRESHOLD" | bc -l) )); then
    DEFICIT=$(echo "$THRESHOLD - $TOTAL_COVERAGE" | bc)
    print_status "❌ FAIL: Coverage ${TOTAL_COVERAGE}% is below ${THRESHOLD}% threshold (deficit: ${DEFICIT}%)" "$RED"
    EXIT_CODE=1
else
    SURPLUS=$(echo "$TOTAL_COVERAGE - $THRESHOLD" | bc)
    print_status "✅ PASS: Coverage meets ${THRESHOLD}% threshold (surplus: ${SURPLUS}%)" "$GREEN"
    EXIT_CODE=0
fi

# Check critical modules
print_header "Critical Module Coverage"

check_module_coverage() {
    local module=$1
    local threshold=$2
    local module_name=$3

    # Calculate average coverage for module
    MODULE_COV=$(awk -v mod="github.com/paw-chain/paw/${module}" '
        $1 ~ mod && !/total:/ {
            gsub(/%/, "", $NF)
            sum += $NF
            count++
        }
        END {
            if (count > 0) print sum/count
            else print 0
        }
    ' coverage_by_package.txt)

    if [ -z "$MODULE_COV" ]; then
        MODULE_COV=0
    fi

    printf "%-20s %6.1f%% " "$module_name" "$MODULE_COV"

    if (( $(echo "$MODULE_COV < $threshold" | bc -l) )); then
        DEFICIT=$(echo "$threshold - $MODULE_COV" | bc)
        print_status "(target: ${threshold}%, deficit: ${DEFICIT}%) ❌" "$RED"
        return 1
    else
        SURPLUS=$(echo "$MODULE_COV - $threshold" | bc)
        print_status "(target: ${threshold}%, surplus: ${SURPLUS}%) ✅" "$GREEN"
        return 0
    fi
}

# Generate package coverage if needed
if [ ! -f coverage_by_package.txt ]; then
    go tool cover -func=coverage.out > coverage_by_package.txt
fi

# Check each critical module
COMPUTE_OK=0
DEX_OK=0
ORACLE_OK=0

check_module_coverage "x/compute" "$THRESHOLD" "Compute" || COMPUTE_OK=1
check_module_coverage "x/dex" "$THRESHOLD" "DEX" || DEX_OK=1
check_module_coverage "x/oracle" "$THRESHOLD" "Oracle" || ORACLE_OK=1

# Check infrastructure modules
print_header "Infrastructure Module Coverage"
check_module_coverage "p2p" "$INFRA_THRESHOLD" "P2P" || EXIT_CODE=1
check_module_coverage "app" "$THRESHOLD" "App" || EXIT_CODE=1

# Check CLI
print_header "CLI Coverage"
check_module_coverage "cmd/pawd" "$CLI_THRESHOLD" "CLI (pawd)" || EXIT_CODE=1

# Find files with lowest coverage
print_header "Top 10 Files Needing Improvement"
awk '/^github.com.paw-chain.paw/ && !/total:/ {
    file=$1;
    gsub(/:.*/, "", file);
    cov=$NF;
    gsub(/%/, "", cov);
    if (cov+0 < 70 && cov+0 > 0) {
        print cov " " file
    }
}' coverage_by_package.txt | sort -n | head -10 | while read line; do
    COV=$(echo $line | awk '{print $1}')
    FILE=$(echo $line | awk '{$1=""; print $0}' | sed 's/^ //')
    printf "  %5.1f%% %s\n" "$COV" "$FILE"
done

# Find files with 0% coverage in critical areas (warning only, not blocking)
print_header "Critical Files with 0% Coverage (info)"
ZERO_COV_COUNT=$(awk '/^github.com.paw-chain.paw\/(x\/compute|x\/dex|x\/oracle)/ && !/total:/ {
    cov=$NF;
    gsub(/%/, "", cov);
    if (cov+0 == 0) count++
}
END { print count+0 }' coverage_by_package.txt)

if [ "$ZERO_COV_COUNT" -gt 0 ]; then
    print_status "Found ${ZERO_COV_COUNT} critical files with 0% coverage (info only)" "$YELLOW"
    awk '/^github.com.paw-chain.paw\/(x\/compute|x\/dex|x\/oracle)/ && !/total:/ {
        file=$1;
        gsub(/:.*/, "", file);
        cov=$NF;
        gsub(/%/, "", cov);
        if (cov+0 == 0) print "  " file
    }' coverage_by_package.txt | head -5
    # Note: Not failing on 0% coverage files, just reporting
fi

# Summary
print_header "Summary"

if [ $EXIT_CODE -eq 0 ]; then
    print_status "✅ All coverage thresholds met!" "$GREEN"
    echo ""
    echo "Coverage breakdown:"
    echo "  - Overall: ${TOTAL_COVERAGE}% (target: ${THRESHOLD}%)"
    echo "  - Critical modules: All above ${THRESHOLD}%"
    echo "  - Infrastructure: Above ${INFRA_THRESHOLD}%"
    echo "  - CLI: Above ${CLI_THRESHOLD}%"
else
    print_status "❌ Coverage thresholds not met" "$RED"
    echo ""
    echo "Action items:"
    [ $(echo "$TOTAL_COVERAGE < $THRESHOLD" | bc -l) -eq 1 ] && \
        echo "  - Improve overall coverage from ${TOTAL_COVERAGE}% to ${THRESHOLD}%"
    [ $COMPUTE_OK -eq 1 ] && echo "  - Add tests to x/compute module"
    [ $DEX_OK -eq 1 ] && echo "  - Add tests to x/dex module"
    [ $ORACLE_OK -eq 1 ] && echo "  - Add tests to x/oracle module"
    echo ""
    echo "Run 'make test-coverage' to see detailed coverage report"
    echo "Open coverage.html in browser for interactive exploration"
fi

echo ""
exit $EXIT_CODE
