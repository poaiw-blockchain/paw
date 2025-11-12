#!/bin/bash
# Check for license headers in Go files

# Exit codes:
# 0 - All files have license headers
# 1 - Some files missing license headers

set -e

# Expected license header pattern
LICENSE_PATTERN="^// Copyright.*PAW"

MISSING=0
FILES="$@"

# Skip generated files
SKIP_PATTERNS=(
    "\.pb\.go$"
    "\.pb\.gw\.go$"
    "_test\.go$"
    "testutil/"
    "tests/"
)

should_skip() {
    local file="$1"
    for pattern in "${SKIP_PATTERNS[@]}"; do
        if echo "$file" | grep -qE "$pattern"; then
            return 0
        fi
    done
    return 1
}

for file in $FILES; do
    if [ ! -f "$file" ]; then
        continue
    fi

    if should_skip "$file"; then
        continue
    fi

    # Check first 10 lines for license header
    if ! head -n 10 "$file" | grep -qE "$LICENSE_PATTERN"; then
        echo "⚠ Missing license header: $file"
        MISSING=1
    fi
done

if [ $MISSING -eq 1 ]; then
    echo ""
    echo "Warning: Some files are missing license headers."
    echo "Consider adding the following header to new files:"
    echo ""
    echo "// Copyright $(date +%Y) PAW Blockchain"
    echo "// Licensed under the Apache License, Version 2.0"
    echo ""
    # Return 0 to allow commit with warning, change to exit 1 to block
    exit 0
fi

exit 0
