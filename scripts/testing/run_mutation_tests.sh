#!/bin/bash
# Run mutation testing on PAW Chain codebase

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="/home/decri/blockchain-projects/paw"
RESULTS_DIR="$PROJECT_ROOT/test-results/mutation"
MIN_SCORE=80

# Create results directory
mkdir -p "$RESULTS_DIR"

echo "======================================"
echo "PAW Chain Mutation Testing"
echo "======================================"
echo ""

# Check if go-mutesting is installed
if ! command -v go-mutesting &> /dev/null; then
    echo -e "${YELLOW}go-mutesting not found. Installing...${NC}"
    go install example.com/zimmski/go-mutesting/cmd/go-mutesting@latest
fi

# Modules to test
MODULES=(
    "x/dex/keeper"
    "x/oracle/keeper"
    "x/compute/keeper"
)

# Run mutation tests for each module
for module in "${MODULES[@]}"; do
    echo -e "${GREEN}Testing module: $module${NC}"
    module_name=$(echo "$module" | tr '/' '_')

    # Run go-mutesting
    cd "$PROJECT_ROOT"

    # Run mutation testing with timeout
    timeout 10m go-mutesting \
        --exec="go test" \
        --exec-timeout="5m" \
        --black-list=".*_test\\.go$,.*\\.pb\\.go$" \
        --debug \
        "$module" \
        > "$RESULTS_DIR/${module_name}_mutations.txt" 2>&1 || true

    # Parse results
    if [ -f "$RESULTS_DIR/${module_name}_mutations.txt" ]; then
        survived=$(grep -c "SURVIVED" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")
        killed=$(grep -c "KILLED" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")
        timeout_count=$(grep -c "TIMEOUT" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")

        total=$((survived + killed + timeout_count))

        if [ $total -gt 0 ]; then
            score=$((killed * 100 / total))

            echo ""
            echo "Results for $module:"
            echo "  Mutations Created: $total"
            echo "  Killed: $killed"
            echo "  Survived: $survived"
            echo "  Timeout: $timeout_count"
            echo "  Score: ${score}%"
            echo ""

            if [ $score -ge $MIN_SCORE ]; then
                echo -e "${GREEN}✓ PASS - Score: ${score}% (target: ${MIN_SCORE}%)${NC}"
            else
                echo -e "${RED}✗ FAIL - Score: ${score}% (target: ${MIN_SCORE}%)${NC}"
            fi
        else
            echo -e "${YELLOW}⚠ WARNING - No mutations generated for $module${NC}"
        fi
    fi

    echo ""
    echo "----------------------------------------"
    echo ""
done

# Generate combined report
echo -e "${GREEN}Generating combined mutation report...${NC}"

cat > "$RESULTS_DIR/mutation_report.md" << 'EOF'
# Mutation Testing Report

## Overview
This report shows the mutation testing results for PAW Chain core modules.

## Methodology
Mutation testing involves making small changes (mutations) to the code and verifying that tests catch these changes.
A high mutation score indicates strong test coverage and test quality.

## Results

EOF

for module in "${MODULES[@]}"; do
    module_name=$(echo "$module" | tr '/' '_')

    if [ -f "$RESULTS_DIR/${module_name}_mutations.txt" ]; then
        survived=$(grep -c "SURVIVED" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")
        killed=$(grep -c "KILLED" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")
        total=$((survived + killed))

        if [ $total -gt 0 ]; then
            score=$((killed * 100 / total))

            cat >> "$RESULTS_DIR/mutation_report.md" << EOF
### $module
- **Mutations**: $total
- **Killed**: $killed
- **Survived**: $survived
- **Score**: ${score}%
- **Status**: $([ $score -ge $MIN_SCORE ] && echo "✅ PASS" || echo "❌ FAIL")

EOF
        fi
    fi
done

cat >> "$RESULTS_DIR/mutation_report.md" << 'EOF'

## Score Interpretation
- **90%+**: Excellent - Very strong test suite
- **80-89%**: Good - Meets quality standards
- **70-79%**: Acceptable - Room for improvement
- **<70%**: Poor - Significant test gaps

## Recommendations
1. Review survived mutations to identify test gaps
2. Add targeted tests for uncaught mutations
3. Consider edge cases and boundary conditions
4. Verify error handling paths

EOF

echo -e "${GREEN}Report generated: $RESULTS_DIR/mutation_report.md${NC}"

# Create JSON summary for CI
cat > "$RESULTS_DIR/mutation_summary.json" << 'EOF'
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "modules": [
EOF

first=true
for module in "${MODULES[@]}"; do
    module_name=$(echo "$module" | tr '/' '_')

    if [ -f "$RESULTS_DIR/${module_name}_mutations.txt" ]; then
        survived=$(grep -c "SURVIVED" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")
        killed=$(grep -c "KILLED" "$RESULTS_DIR/${module_name}_mutations.txt" || echo "0")
        total=$((survived + killed))

        if [ $total -gt 0 ]; then
            score=$((killed * 100 / total))

            if [ "$first" = false ]; then
                echo "," >> "$RESULTS_DIR/mutation_summary.json"
            fi
            first=false

            cat >> "$RESULTS_DIR/mutation_summary.json" << EOF
    {
      "module": "$module",
      "total_mutations": $total,
      "killed": $killed,
      "survived": $survived,
      "score": $score,
      "pass": $([ $score -ge $MIN_SCORE ] && echo "true" || echo "false")
    }
EOF
        fi
    fi
done

cat >> "$RESULTS_DIR/mutation_summary.json" << 'EOF'
  ]
}
EOF

echo ""
echo -e "${GREEN}✓ Mutation testing complete!${NC}"
echo "Results saved to: $RESULTS_DIR"
