#!/bin/bash
# Validate proto files for syntax and style

set -e

FILES="$@"
ERRORS=0

for file in $FILES; do
    if [ ! -f "$file" ]; then
        continue
    fi

    echo "Checking proto file: $file"

    # Check syntax with protoc if available
    if command -v protoc &> /dev/null; then
        PROTO_DIR=$(dirname "$file")
        if protoc --proto_path="$PROTO_DIR" --descriptor_set_out=/dev/null "$file" 2>&1; then
            echo "✓ Syntax OK: $file"
        else
            echo "✗ Syntax error in: $file"
            ERRORS=$((ERRORS + 1))
        fi
    else
        echo "⚠ protoc not installed, skipping syntax check"
    fi

    # Basic style checks
    # Check for proper package declaration
    if ! grep -q "^package " "$file"; then
        echo "✗ Missing package declaration in: $file"
        ERRORS=$((ERRORS + 1))
    fi

    # Check for go_package option
    if ! grep -q "option go_package" "$file"; then
        echo "⚠ Missing go_package option in: $file (recommended)"
    fi

    # Check for proper syntax version
    if ! grep -q 'syntax = "proto3"' "$file"; then
        echo "⚠ Missing or incorrect syntax version in: $file"
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "✗ Proto validation failed with $ERRORS error(s)"
    exit 1
fi

echo ""
echo "✓ All proto files validated successfully"
exit 0
