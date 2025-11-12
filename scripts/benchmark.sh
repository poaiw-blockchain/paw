#!/bin/bash

# PAW Blockchain Benchmark Runner
# Runs Go benchmarks with profiling and generates reports

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
BENCHMARK_DIR="tests/benchmarks"
REPORT_DIR="tests/load/reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create report directory
mkdir -p "${REPORT_DIR}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}PAW Blockchain Benchmark Runner${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Function to run benchmarks with profiling
run_benchmarks() {
    local module=$1
    local output_prefix="${REPORT_DIR}/bench-${module}-${TIMESTAMP}"

    echo -e "${YELLOW}Running ${module} benchmarks...${NC}"

    go test -bench=. -benchmem -benchtime=10s \
        -cpuprofile="${output_prefix}-cpu.prof" \
        -memprofile="${output_prefix}-mem.prof" \
        -blockprofile="${output_prefix}-block.prof" \
        -mutexprofile="${output_prefix}-mutex.prof" \
        ./${BENCHMARK_DIR}/ \
        -run=^$ \
        > "${output_prefix}.txt" 2>&1

    echo -e "${GREEN}✓ Benchmarks completed${NC}"
    echo ""
}

# Function to analyze CPU profile
analyze_cpu_profile() {
    local profile=$1

    echo -e "${YELLOW}Analyzing CPU profile...${NC}"

    # Generate text report
    go tool pprof -text "${profile}" > "${profile%.prof}-analysis.txt" 2>&1

    # Generate top functions
    go tool pprof -top10 "${profile}" >> "${profile%.prof}-analysis.txt" 2>&1

    echo "CPU profile analysis saved to: ${profile%.prof}-analysis.txt"
    echo ""

    # Optionally start interactive web interface
    if [ "${INTERACTIVE}" = "true" ]; then
        echo "Starting interactive CPU profile viewer..."
        echo "Open http://localhost:8080 in your browser"
        go tool pprof -http=:8080 "${profile}"
    fi
}

# Function to analyze memory profile
analyze_mem_profile() {
    local profile=$1

    echo -e "${YELLOW}Analyzing memory profile...${NC}"

    # Generate text report
    go tool pprof -text "${profile}" > "${profile%.prof}-analysis.txt" 2>&1

    # Show allocations
    go tool pprof -alloc_space -text "${profile}" >> "${profile%.prof}-analysis.txt" 2>&1

    echo "Memory profile analysis saved to: ${profile%.prof}-analysis.txt"
    echo ""

    if [ "${INTERACTIVE}" = "true" ]; then
        echo "Starting interactive memory profile viewer..."
        go tool pprof -http=:8081 "${profile}"
    fi
}

# Function to compare benchmarks
compare_benchmarks() {
    if [ -f "${BASELINE_FILE}" ]; then
        echo -e "${YELLOW}Comparing with baseline...${NC}"

        # Install benchcmp if not available
        if ! command -v benchcmp &> /dev/null; then
            echo "Installing benchcmp..."
            go install golang.org/x/tools/cmd/benchcmp@latest
        fi

        local current="${REPORT_DIR}/bench-${TIMESTAMP}.txt"
        benchcmp "${BASELINE_FILE}" "${current}" > "${REPORT_DIR}/bench-comparison-${TIMESTAMP}.txt"

        echo "Comparison saved to: ${REPORT_DIR}/bench-comparison-${TIMESTAMP}.txt"
        echo ""
    fi
}

# Function to generate HTML report
generate_html_report() {
    local bench_file=$1
    local html_file="${bench_file%.txt}.html"

    echo -e "${YELLOW}Generating HTML report...${NC}"

    cat > "${html_file}" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>PAW Benchmark Report - ${TIMESTAMP}</title>
    <style>
        body {
            font-family: 'Courier New', monospace;
            margin: 20px;
            background-color: #1e1e1e;
            color: #d4d4d4;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background-color: #252526;
            padding: 30px;
            box-shadow: 0 0 20px rgba(0,0,0,0.5);
        }
        h1 {
            color: #4ec9b0;
            border-bottom: 2px solid #4ec9b0;
            padding-bottom: 10px;
        }
        h2 {
            color: #569cd6;
            margin-top: 30px;
        }
        pre {
            background-color: #1e1e1e;
            padding: 15px;
            border-left: 4px solid #4ec9b0;
            overflow-x: auto;
            font-size: 12px;
        }
        .metric {
            color: #ce9178;
        }
        .benchmark-name {
            color: #dcdcaa;
            font-weight: bold;
        }
        .fast {
            color: #4ec9b0;
        }
        .slow {
            color: #f48771;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>PAW Blockchain Benchmark Report</h1>
        <p>Generated: ${TIMESTAMP}</p>

        <h2>Benchmark Results</h2>
        <pre>$(cat "${bench_file}")</pre>

        <h2>Performance Profiles</h2>
        <ul>
            <li>CPU Profile: <a href="$(basename "${bench_file%.txt}-cpu.prof")">Download</a></li>
            <li>Memory Profile: <a href="$(basename "${bench_file%.txt}-mem.prof")">Download</a></li>
            <li>Block Profile: <a href="$(basename "${bench_file%.txt}-block.prof")">Download</a></li>
            <li>Mutex Profile: <a href="$(basename "${bench_file%.txt}-mutex.prof")">Download</a></li>
        </ul>

        <h2>How to Analyze Profiles</h2>
        <pre>
# CPU Profile
go tool pprof -http=:8080 $(basename "${bench_file%.txt}-cpu.prof")

# Memory Profile
go tool pprof -http=:8080 $(basename "${bench_file%.txt}-mem.prof")

# Generate flame graph
go tool pprof -http=:8080 -flame $(basename "${bench_file%.txt}-cpu.prof")
        </pre>
    </div>
</body>
</html>
EOF

    echo "HTML report generated: ${html_file}"
    echo ""
}

# Main execution
main() {
    # Check for flags
    while [[ $# -gt 0 ]]; do
        case $1 in
            --interactive|-i)
                INTERACTIVE=true
                shift
                ;;
            --baseline|-b)
                BASELINE_FILE="$2"
                shift 2
                ;;
            --module|-m)
                MODULE="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Run benchmarks
    if [ -n "${MODULE}" ]; then
        run_benchmarks "${MODULE}"
    else
        run_benchmarks "all"
    fi

    # Find the latest benchmark output
    LATEST_BENCH=$(ls -t "${REPORT_DIR}"/bench-*-${TIMESTAMP}.txt | head -1)
    LATEST_CPU=$(ls -t "${REPORT_DIR}"/bench-*-${TIMESTAMP}-cpu.prof | head -1)
    LATEST_MEM=$(ls -t "${REPORT_DIR}"/bench-*-${TIMESTAMP}-mem.prof | head -1)

    # Analyze profiles
    if [ -f "${LATEST_CPU}" ]; then
        analyze_cpu_profile "${LATEST_CPU}"
    fi

    if [ -f "${LATEST_MEM}" ]; then
        analyze_mem_profile "${LATEST_MEM}"
    fi

    # Compare with baseline if provided
    compare_benchmarks

    # Generate HTML report
    if [ -f "${LATEST_BENCH}" ]; then
        generate_html_report "${LATEST_BENCH}"
    fi

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Benchmarking completed!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Benchmark results: ${LATEST_BENCH}"
    echo "Reports directory: ${REPORT_DIR}"
    echo ""
    echo "To view profiles interactively, run:"
    echo "  $0 --interactive"
}

# Show usage if --help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --interactive, -i       Start interactive profile viewer"
    echo "  --baseline FILE, -b     Compare with baseline file"
    echo "  --module NAME, -m       Run benchmarks for specific module"
    echo "  --help, -h              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                      # Run all benchmarks"
    echo "  $0 --interactive        # Run and open interactive viewer"
    echo "  $0 --module dex         # Run only DEX benchmarks"
    echo "  $0 --baseline old.txt   # Compare with baseline"
    exit 0
fi

main "$@"
