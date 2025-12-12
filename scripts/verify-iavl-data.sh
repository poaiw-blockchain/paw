#!/bin/bash
# Verify that IAVL version data exists in the database

set -e

DB_PATH="$HOME/.paw/data/application.db"
CURRENT_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height')

echo "================================================"
echo "IAVL Data Verification Script"
echo "================================================"
echo ""
echo "Current block height: $CURRENT_HEIGHT"
echo "Database path: $DB_PATH"
echo ""

echo "=== Testing Version Metadata Existence ==="
echo ""

# Test a few key versions
for height in 1 10 100 1000 $((CURRENT_HEIGHT - 100)) $((CURRENT_HEIGHT - 10)) $CURRENT_HEIGHT; do
    if [ "$height" -lt 1 ]; then
        continue
    fi

    echo -n "Version $height: "
    result=$(ldb --db="$DB_PATH" get "s/$height" 2>&1 | head -1)

    if echo "$result" | grep -q "NotFound"; then
        echo "❌ NOT FOUND"
    else
        # Count the number of stores in this version
        store_count=$(ldb --db="$DB_PATH" get "s/$height" 2>/dev/null | strings | grep -c "k:")
        echo "✅ EXISTS ($store_count stores)"
    fi
done

echo ""
echo "=== Testing Query Path ==="
echo ""

echo -n "Querying current height via gRPC: "
query_result=$(grpcurl -plaintext localhost:9091 cosmos.bank.v1beta1.Query/TotalSupply 2>&1)

if echo "$query_result" | grep -q "version does not exist"; then
    echo "❌ FAILED - version does not exist error"
    echo "$query_result" | grep "Message:" | sed 's/^  /    /'
elif echo "$query_result" | grep -q '"supply"'; then
    echo "✅ SUCCESS"
else
    echo "⚠️  UNEXPECTED RESULT"
    echo "$query_result" | head -3 | sed 's/^/    /'
fi

echo ""
echo -n "Querying height 100 via gRPC: "
query_result=$(grpcurl -plaintext -H "x-cosmos-block-height: 100" localhost:9091 cosmos.bank.v1beta1.Query/TotalSupply 2>&1)

if echo "$query_result" | grep -q "version does not exist"; then
    echo "❌ FAILED - version does not exist error"
    echo "$query_result" | grep "Message:" | sed 's/^  /    /'
elif echo "$query_result" | grep -q '"supply"'; then
    echo "✅ SUCCESS"
else
    echo "⚠️  UNEXPECTED RESULT"
    echo "$query_result" | head -3 | sed 's/^/    /'
fi

echo ""
echo "=== Summary ==="
echo ""
echo "If version metadata EXISTS but queries FAIL:"
echo "  → Data is being saved ✅"
echo "  → LoadVersion() is broken ❌"
echo ""
echo "See docs/IAVL_INSPECTION_RESULTS.md for details."
echo ""
