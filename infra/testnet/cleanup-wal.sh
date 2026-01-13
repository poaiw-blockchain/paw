#!/bin/bash
#
# PAW Testnet WAL Cleanup Script
# Removes old WAL segment files to prevent disk space issues
#
# Usage: ./cleanup-wal.sh [--keep N] [--dry-run]
#
# Default: Keep last 10 WAL segment files

set -euo pipefail

KEEP=10
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --keep) KEEP="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

cleanup_wal() {
    local server=$1
    local home=$2
    local name=$3

    echo "=== $name on $server ==="

    # Get WAL segment files (wal.000, wal.001, etc.) sorted by name
    local files=$(ssh "$server" "ls -1 $home/data/cs.wal/wal.* 2>/dev/null | sort -t. -k2 -n" || echo "")

    if [[ -z "$files" ]]; then
        echo "  No WAL segment files found"
        return
    fi

    local count=$(echo "$files" | wc -l)
    local to_delete=$((count - KEEP))

    if [[ $to_delete -le 0 ]]; then
        echo "  $count files found, keeping all (threshold: $KEEP)"
        return
    fi

    echo "  $count files found, deleting oldest $to_delete (keeping $KEEP)"

    local delete_list=$(echo "$files" | head -n "$to_delete")

    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [DRY-RUN] Would delete:"
        echo "$delete_list" | sed 's/^/    /'
    else
        echo "$delete_list" | while read -r f; do
            echo "  Deleting: $f"
            ssh "$server" "rm -f '$f'"
        done
        echo "  Done"
    fi
    echo ""
}

echo "PAW Testnet WAL Cleanup"
echo "======================="
echo "Keeping last $KEEP segment files per validator"
[[ "$DRY_RUN" == "true" ]] && echo "DRY RUN MODE"
echo ""

cleanup_wal paw-testnet "/home/ubuntu/.paw-val1" "val1"
cleanup_wal paw-testnet "/home/ubuntu/.paw-val2" "val2"
cleanup_wal services-testnet "/home/ubuntu/.paw-val3" "val3"
cleanup_wal services-testnet "/home/ubuntu/.paw-val4" "val4"

echo "Cleanup complete"
