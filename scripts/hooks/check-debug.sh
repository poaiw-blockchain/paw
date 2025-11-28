#!/bin/bash
# Check for debug statements in Go files

# Exit codes:
# 0 - No debug statements found
# 1 - Debug statements found

set -e

# Debug patterns to check for in Go files
PATTERNS=(
    'fmt\.Println\('
    'fmt\.Printf\("DEBUG'
    'log\.Println\("DEBUG'
    'panic\('
    'TODO:'
    'FIXME:'
    'XXX:'
)

FOUND=0
FILES="$@"

for file in $FILES; do
    if [ ! -f "$file" ]; then
        continue
    fi

    for pattern in "${PATTERNS[@]}"; do
        if grep -n -E "$pattern" "$file" > /dev/null; then
            echo "⚠ Debug statement found in $file:"
            grep -n -E "$pattern" "$file"
            FOUND=1
        fi
    done
done

if [ $FOUND -eq 1 ]; then
    echo ""
    echo "Warning: Debug statements detected. Please remove them before committing."
    echo "If these are intentional, you can bypass this check with --no-verify"
    # Return 0 to allow commit with warning, change to exit 1 to block
    exit 0
fi

exit 0
