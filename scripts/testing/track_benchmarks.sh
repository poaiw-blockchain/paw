#!/bin/bash
# Benchmark tracking system with regression detection

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
PROJECT_ROOT="/home/decri/blockchain-projects/paw"
BENCH_DIR="$PROJECT_ROOT/test-results/benchmarks"
BENCH_HISTORY="$BENCH_DIR/history"
REGRESSION_THRESHOLD=10  # 10% slowdown is considered a regression

mkdir -p "$BENCH_HISTORY"

echo "======================================"
echo "PAW Chain Benchmark Tracking"
echo "======================================"
echo ""

# Get current timestamp
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
CURRENT_RESULTS="$BENCH_DIR/bench_$TIMESTAMP.txt"
CURRENT_JSON="$BENCH_DIR/bench_$TIMESTAMP.json"

# Run benchmarks
echo "Running benchmarks..."
cd "$PROJECT_ROOT"

go test -bench=. -benchmem -benchtime=3s \
    ./tests/benchmarks/... \
    > "$CURRENT_RESULTS" 2>&1 || true

echo "Benchmarks completed. Results saved to: $CURRENT_RESULTS"
echo ""

# Parse benchmark results to JSON
echo "Parsing results..."

cat > /tmp/parse_bench.py << 'EOFPYTHON'
#!/usr/bin/env python3
import json
import re
import sys
from datetime import datetime

def parse_benchmark_line(line):
    """Parse a single benchmark result line."""
    # Format: BenchmarkName-N    iterations    ns/op    bytes/op    allocs/op
    match = re.match(r'(\S+)\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?', line)

    if match:
        name, iterations, ns_per_op, bytes_per_op, allocs_per_op = match.groups()
        return {
            'name': name,
            'iterations': int(iterations),
            'ns_per_op': float(ns_per_op),
            'bytes_per_op': int(bytes_per_op) if bytes_per_op else 0,
            'allocs_per_op': int(allocs_per_op) if allocs_per_op else 0
        }
    return None

def main():
    if len(sys.argv) < 3:
        print("Usage: parse_bench.py <input_file> <output_json>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    benchmarks = []

    with open(input_file, 'r') as f:
        for line in f:
            line = line.strip()
            if line.startswith('Benchmark'):
                result = parse_benchmark_line(line)
                if result:
                    benchmarks.append(result)

    # Create output structure
    output = {
        'timestamp': datetime.utcnow().isoformat() + 'Z',
        'benchmarks': benchmarks
    }

    with open(output_file, 'w') as f:
        json.dump(output, f, indent=2)

    print(f"Parsed {len(benchmarks)} benchmarks")

if __name__ == '__main__':
    main()
EOFPYTHON

chmod +x /tmp/parse_bench.py
python3 /tmp/parse_bench.py "$CURRENT_RESULTS" "$CURRENT_JSON"

# Copy to history
cp "$CURRENT_JSON" "$BENCH_HISTORY/bench_$TIMESTAMP.json"

echo ""
echo "Results parsed and saved to: $CURRENT_JSON"
echo ""

# Find previous benchmark result
PREVIOUS_JSON=$(ls -t "$BENCH_HISTORY"/bench_*.json 2>/dev/null | sed -n '2p' || echo "")

if [ -z "$PREVIOUS_JSON" ]; then
    echo -e "${YELLOW}No previous benchmark results found. Skipping regression check.${NC}"
    echo "This result will be used as baseline for future comparisons."
else
    echo "Comparing with previous results: $(basename $PREVIOUS_JSON)"
    echo ""

    # Compare benchmarks and detect regressions
    cat > /tmp/compare_bench.py << 'EOFPYTHON'
#!/usr/bin/env python3
import json
import sys

REGRESSION_THRESHOLD = 10.0  # 10%

def load_benchmarks(filepath):
    """Load benchmarks from JSON file."""
    with open(filepath, 'r') as f:
        data = json.load(f)
    return {b['name']: b for b in data['benchmarks']}

def compare_benchmarks(current_file, previous_file):
    """Compare current and previous benchmarks."""
    current = load_benchmarks(current_file)
    previous = load_benchmarks(previous_file)

    regressions = []
    improvements = []
    new_benchmarks = []

    for name, curr in current.items():
        if name not in previous:
            new_benchmarks.append(name)
            continue

        prev = previous[name]

        # Calculate percentage change in ns/op
        if prev['ns_per_op'] > 0:
            change = ((curr['ns_per_op'] - prev['ns_per_op']) / prev['ns_per_op']) * 100

            if change > REGRESSION_THRESHOLD:
                regressions.append({
                    'name': name,
                    'previous': prev['ns_per_op'],
                    'current': curr['ns_per_op'],
                    'change': change
                })
            elif change < -REGRESSION_THRESHOLD:
                improvements.append({
                    'name': name,
                    'previous': prev['ns_per_op'],
                    'current': curr['ns_per_op'],
                    'change': change
                })

    return regressions, improvements, new_benchmarks

def main():
    if len(sys.argv) < 3:
        print("Usage: compare_bench.py <current_json> <previous_json>")
        sys.exit(1)

    current_file = sys.argv[1]
    previous_file = sys.argv[2]

    regressions, improvements, new_benchmarks = compare_benchmarks(current_file, previous_file)

    # Print results
    print("=" * 60)
    print("BENCHMARK COMPARISON RESULTS")
    print("=" * 60)
    print()

    if regressions:
        print(f"⚠️  REGRESSIONS DETECTED ({len(regressions)}):")
        print()
        for r in regressions:
            print(f"  {r['name']}")
            print(f"    Previous: {r['previous']:.2f} ns/op")
            print(f"    Current:  {r['current']:.2f} ns/op")
            print(f"    Change:   {r['change']:+.1f}% (SLOWER)")
            print()

    if improvements:
        print(f"✅ IMPROVEMENTS ({len(improvements)}):")
        print()
        for i in improvements:
            print(f"  {i['name']}")
            print(f"    Previous: {i['previous']:.2f} ns/op")
            print(f"    Current:  {i['current']:.2f} ns/op")
            print(f"    Change:   {i['change']:+.1f}% (FASTER)")
            print()

    if new_benchmarks:
        print(f"🆕 NEW BENCHMARKS ({len(new_benchmarks)}):")
        for name in new_benchmarks:
            print(f"  {name}")
        print()

    if not regressions and not improvements and not new_benchmarks:
        print("No significant changes detected.")
        print()

    # Exit with error if regressions detected
    if regressions:
        sys.exit(1)

if __name__ == '__main__':
    main()
EOFPYTHON

    chmod +x /tmp/compare_bench.py

    if python3 /tmp/compare_bench.py "$CURRENT_JSON" "$PREVIOUS_JSON"; then
        echo -e "${GREEN}✓ No performance regressions detected!${NC}"
    else
        echo -e "${RED}✗ Performance regressions detected!${NC}"
        echo ""
        echo "Review the regressions above and investigate the cause."
        exit 1
    fi
fi

# Create summary report
echo ""
echo "Generating benchmark summary..."

cat > "$BENCH_DIR/benchmark_summary.md" << 'EOF'
# Benchmark Summary

## Latest Results

EOF

# Add timestamp
echo "**Timestamp:** $(date -u +%Y-%m-%d\ %H:%M:%S) UTC" >> "$BENCH_DIR/benchmark_summary.md"
echo "" >> "$BENCH_DIR/benchmark_summary.md"

# Add top 10 slowest benchmarks
echo "### Top 10 Slowest Benchmarks" >> "$BENCH_DIR/benchmark_summary.md"
echo "" >> "$BENCH_DIR/benchmark_summary.md"
echo "| Benchmark | ns/op | bytes/op | allocs/op |" >> "$BENCH_DIR/benchmark_summary.md"
echo "|-----------|-------|----------|-----------|" >> "$BENCH_DIR/benchmark_summary.md"

jq -r '.benchmarks | sort_by(.ns_per_op) | reverse | .[:10] | .[] | "| \(.name) | \(.ns_per_op | tostring) | \(.bytes_per_op | tostring) | \(.allocs_per_op | tostring) |"' "$CURRENT_JSON" >> "$BENCH_DIR/benchmark_summary.md"

echo "" >> "$BENCH_DIR/benchmark_summary.md"
echo "### All Benchmarks" >> "$BENCH_DIR/benchmark_summary.md"
echo "" >> "$BENCH_DIR/benchmark_summary.md"
echo "Total benchmarks: $(jq '.benchmarks | length' "$CURRENT_JSON")" >> "$BENCH_DIR/benchmark_summary.md"

echo ""
echo -e "${GREEN}✓ Benchmark tracking complete!${NC}"
echo "Summary: $BENCH_DIR/benchmark_summary.md"
