#!/bin/bash
# Script to add t.Parallel() to test functions that don't have shared state

# Add t.Parallel() to property tests
files=(
    "/home/decri/blockchain-projects/paw/tests/property/dex_properties_test.go"
    "/home/decri/blockchain-projects/paw/tests/property/oracle_properties_test.go"
    "/home/decri/blockchain-projects/paw/tests/differential/dex_differential_test.go"
    "/home/decri/blockchain-projects/paw/tests/differential/oracle_differential_test.go"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "Processing $file"
        # Add t.Parallel() after "func Test" declarations (excluding suite tests and benchmarks)
        sed -i '/^func Test[^(]*{[a-zA-Z]*Test(t \*testing\.T) {$/!b; n; /^\tt\.Parallel()/b; i\\tt.Parallel()' "$file"
    fi
done

echo "Parallel test markers added successfully!"
