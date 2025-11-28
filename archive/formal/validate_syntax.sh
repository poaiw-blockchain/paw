#!/bin/bash
################################################################################
# TLA+ Syntax Validation Script
#
# Validates TLA+ specifications without running full model checking.
# Useful for quick syntax checks during development.
################################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TLC_JAR="${TLC_HOME:-/opt/TLA+Toolbox}/tla2tools.jar"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "Validating TLA+ specifications..."

# Function to validate a single spec
validate_spec() {
    local spec=$1
    echo -n "  Checking ${spec}... "

    if java -cp "$TLC_JAR" tla2sany.SANY "$SCRIPT_DIR/${spec}.tla" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        java -cp "$TLC_JAR" tla2sany.SANY "$SCRIPT_DIR/${spec}.tla"
        return 1
    fi
}

# Validate all specs
failed=0

validate_spec "dex_invariant" || ((failed++))
validate_spec "escrow_safety" || ((failed++))
validate_spec "oracle_bft" || ((failed++))

if [ $failed -eq 0 ]; then
    echo -e "\n${GREEN}All specifications are syntactically valid!${NC}"
    exit 0
else
    echo -e "\n${RED}$failed specification(s) have syntax errors!${NC}"
    exit 1
fi
