#!/bin/bash
# Simple script to add t.Parallel() to test functions

# Process property tests (excluding compute which is already done)
sed -i '/^func TestProperty.*t \*testing\.T) {$/a\\tt.Parallel()' /home/decri/blockchain-projects/paw/tests/property/dex_properties_test.go
sed -i '/^func TestProperty.*t \*testing\.T) {$/a\\tt.Parallel()' /home/decri/blockchain-projects/paw/tests/property/oracle_properties_test.go

# Process differential tests
sed -i '/^func TestPAW.*t \*testing\.T) {$/a\\tt.Parallel()' /home/decri/blockchain-projects/paw/tests/differential/dex_differential_test.go
sed -i '/^func TestPAW.*t \*testing\.T) {$/a\\tt.Parallel()' /home/decri/blockchain-projects/paw/tests/differential/oracle_differential_test.go

echo "Added t.Parallel() markers to property and differential tests"
