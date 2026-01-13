#!/bin/bash
#
# PAW Testnet Simple Health Check
# Reliable validator status check for 4-validator MVP testnet deployment
# Run this from bcpc (has SSH aliases configured)
#
# MVP Testnet Port Configuration:
#   val1 (paw-testnet:11657)     - 54.39.103.49
#   val2 (paw-testnet:11757)     - 54.39.103.49
#   val3 (services-testnet:11857) - 139.99.149.160
#   val4 (services-testnet:11957) - 139.99.149.160
#
# Public Endpoints:
#   RPC: https://testnet-rpc.poaiw.org
#   REST: https://testnet-api.poaiw.org
#   gRPC: testnet-grpc.poaiw.org:443
#   Explorer: https://testnet-explorer.poaiw.org
#   Faucet: https://testnet-faucet.poaiw.org

set -euo pipefail

echo "=== PAW Testnet Health Report ==="
echo "Time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo ""

# Get status from all validators
echo "Checking validators..."
echo ""

val1_height=$(ssh paw-testnet "curl -s http://127.0.0.1:11657/status | jq -r '.result.sync_info.latest_block_height'" 2>/dev/null || echo "ERROR")
val1_peers=$(ssh paw-testnet "curl -s http://127.0.0.1:11657/net_info | jq -r '.result.n_peers'" 2>/dev/null || echo "0")
val1_catching=$(ssh paw-testnet "curl -s http://127.0.0.1:11657/status | jq -r '.result.sync_info.catching_up'" 2>/dev/null || echo "true")

val2_height=$(ssh paw-testnet "curl -s http://127.0.0.1:11757/status | jq -r '.result.sync_info.latest_block_height'" 2>/dev/null || echo "ERROR")
val2_peers=$(ssh paw-testnet "curl -s http://127.0.0.1:11757/net_info | jq -r '.result.n_peers'" 2>/dev/null || echo "0")
val2_catching=$(ssh paw-testnet "curl -s http://127.0.0.1:11757/status | jq -r '.result.sync_info.catching_up'" 2>/dev/null || echo "true")

val3_height=$(ssh services-testnet "curl -s http://127.0.0.1:11857/status | jq -r '.result.sync_info.latest_block_height'" 2>/dev/null || echo "ERROR")
val3_peers=$(ssh services-testnet "curl -s http://127.0.0.1:11857/net_info | jq -r '.result.n_peers'" 2>/dev/null || echo "0")
val3_catching=$(ssh services-testnet "curl -s http://127.0.0.1:11857/status | jq -r '.result.sync_info.catching_up'" 2>/dev/null || echo "true")

val4_height=$(ssh services-testnet "curl -s http://127.0.0.1:11957/status | jq -r '.result.sync_info.latest_block_height'" 2>/dev/null || echo "ERROR")
val4_peers=$(ssh services-testnet "curl -s http://127.0.0.1:11957/net_info | jq -r '.result.n_peers'" 2>/dev/null || echo "0")
val4_catching=$(ssh services-testnet "curl -s http://127.0.0.1:11957/status | jq -r '.result.sync_info.catching_up'" 2>/dev/null || echo "true")

# Convert catching_up to Synced status
sync1="Yes"; [[ "$val1_catching" == "true" ]] && sync1="No"
sync2="Yes"; [[ "$val2_catching" == "true" ]] && sync2="No"
sync3="Yes"; [[ "$val3_catching" == "true" ]] && sync3="No"
sync4="Yes"; [[ "$val4_catching" == "true" ]] && sync4="No"

# Print table
printf "%-8s %-12s %-8s %-8s %-20s\n" "Node" "Height" "Peers" "Synced" "Server"
echo "----------------------------------------------------------------"
printf "%-8s %-12s %-8s %-8s %-20s\n" "val1" "$val1_height" "$val1_peers" "$sync1" "paw-testnet:11657"
printf "%-8s %-12s %-8s %-8s %-20s\n" "val2" "$val2_height" "$val2_peers" "$sync2" "paw-testnet:11757"
printf "%-8s %-12s %-8s %-8s %-20s\n" "val3" "$val3_height" "$val3_peers" "$sync3" "services-testnet:11857"
printf "%-8s %-12s %-8s %-8s %-20s\n" "val4" "$val4_height" "$val4_peers" "$sync4" "services-testnet:11957"
echo ""

# Calculate height diff (handle ERROR cases)
heights=()
for h in "$val1_height" "$val2_height" "$val3_height" "$val4_height"; do
    [[ "$h" != "ERROR" && "$h" =~ ^[0-9]+$ ]] && heights+=("$h")
done

if [[ ${#heights[@]} -gt 0 ]]; then
    max_h=$(printf '%s\n' "${heights[@]}" | sort -rn | head -1)
    min_h=$(printf '%s\n' "${heights[@]}" | sort -n | head -1)
    diff=$((max_h - min_h))

    echo "Height Stats:"
    echo "  Max: $max_h"
    echo "  Min: $min_h"
    echo "  Diff: $diff blocks"
    echo ""

    if [[ $diff -lt 2 ]]; then
        echo -e "Status: \033[0;32mHEALTHY\033[0m - All validators in sync"
        exit 0
    elif [[ $diff -lt 5 ]]; then
        echo -e "Status: \033[1;33mWARNING\033[0m - Minor height divergence"
        exit 1
    else
        echo -e "Status: \033[0;31mCRITICAL\033[0m - Significant height divergence"
        exit 2
    fi
else
    echo -e "Status: \033[0;31mCRITICAL\033[0m - Unable to get heights from validators"
    exit 3
fi
