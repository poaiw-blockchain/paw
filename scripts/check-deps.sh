#!/bin/bash
# Dependency Security Check Script for PAW Blockchain
# Checks for known vulnerabilities in Go dependencies

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}PAW Blockchain - Dependency Security Check${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print section headers
print_header() {
    echo ""
    echo -e "${BLUE}==== $1 ====${NC}"
    echo ""
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

FAILED=0

# 1. Run govulncheck (official Go vulnerability scanner)
print_header "Running govulncheck"
if command_exists govulncheck; then
    if govulncheck ./...; then
        print_success "No vulnerabilities found by govulncheck"
    else
        print_error "Vulnerabilities found by govulncheck"
        FAILED=1
    fi
else
    print_warning "govulncheck not installed"
    echo "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
    FAILED=1
fi

# 2. Verify go.mod integrity
print_header "Verifying go.mod and go.sum integrity"
if go mod verify; then
    print_success "All dependencies verified successfully"
else
    print_error "Dependency verification failed"
    FAILED=1
fi

# 3. Check for outdated dependencies
print_header "Checking for outdated dependencies"
if command_exists go-mod-outdated; then
    go list -u -m -json all | go-mod-outdated -update -direct || true
else
    print_warning "go-mod-outdated not installed (optional)"
    echo "Install with: go install github.com/psampaz/go-mod-outdated@latest"
fi

# 4. List all dependencies
print_header "Current Dependencies"
echo "Direct dependencies:"
go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all

echo ""
echo "Indirect dependencies:"
go list -m -f '{{if .Indirect}}{{.Path}} {{.Version}}{{end}}' all | head -20
echo "... (showing first 20)"

# 5. Check for replace directives (potential security concern)
print_header "Checking for replace directives"
if grep -q "^replace" go.mod 2>/dev/null; then
    print_warning "Found replace directives in go.mod:"
    grep "^replace" go.mod
    echo "Review these carefully - they override dependency resolution"
else
    print_success "No replace directives found"
fi

# 6. Check for local file:// dependencies (security concern)
print_header "Checking for local path dependencies"
if grep -q "=> \\.\\." go.mod 2>/dev/null || grep -q "=> /" go.mod 2>/dev/null; then
    print_error "Found local path dependencies in go.mod:"
    grep "=> " go.mod | grep -E "(\\.\\.|/)"
    echo "Local paths should not be used in production"
    FAILED=1
else
    print_success "No local path dependencies found"
fi

# 7. Check dependency licenses (if go-licenses is installed)
print_header "Checking dependency licenses"
if command_exists go-licenses; then
    echo "Saving license report..."
    go-licenses report ./... > security/dependency-licenses.txt 2>/dev/null || true
    print_success "License report saved to security/dependency-licenses.txt"
else
    print_warning "go-licenses not installed (optional)"
    echo "Install with: go install github.com/google/go-licenses@latest"
fi

# 8. Generate dependency graph (if modgraphviz is installed)
print_header "Generating dependency graph"
if command_exists modgraphviz; then
    go mod graph | modgraphviz | dot -Tpng -o security/dependency-graph.png 2>/dev/null || true
    if [ -f security/dependency-graph.png ]; then
        print_success "Dependency graph saved to security/dependency-graph.png"
    fi
else
    print_warning "modgraphviz not installed (optional)"
    echo "Install with: go install golang.org/x/exp/cmd/modgraphviz@latest"
fi

# 9. Check for known bad packages
print_header "Checking for known problematic packages"
BAD_PACKAGES=0

# Check for packages with known issues (add more as needed)
if go list -m all | grep -q "github.com/ugorji/go/codec"; then
    print_warning "Found github.com/ugorji/go/codec - consider alternatives"
    BAD_PACKAGES=1
fi

if [ $BAD_PACKAGES -eq 0 ]; then
    print_success "No known problematic packages found"
fi

# 10. Summary
echo ""
echo -e "${BLUE}================================================${NC}"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}Dependency Security Check: PASSED${NC}"
    echo -e "${BLUE}================================================${NC}"
    exit 0
else
    echo -e "${RED}Dependency Security Check: FAILED${NC}"
    echo -e "${RED}Please review the issues found above${NC}"
    echo -e "${BLUE}================================================${NC}"
    exit 1
fi
