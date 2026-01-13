#!/bin/bash
#
# Deploy Hardened Configuration to PAW Testnet Validators
#
# This script applies hardened configurations to all 4 validators
# with proper backup and rollback support.
#
# Usage: ./deploy-hardened-config.sh [--dry-run] [--validator VAL_NUM]
#
# WARNING: This will restart validators! Run during low-activity periods.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="/tmp/paw-config-backup-$(date +%Y%m%d-%H%M%S)"

# Validator configuration
declare -A VALIDATORS=(
    ["val1"]="paw-testnet:/home/ubuntu/.paw-val1:11657:11656:11090:11317:11660"
    ["val2"]="paw-testnet:/home/ubuntu/.paw-val2:11757:11756:11190:11417:11760"
    ["val3"]="services-testnet:/home/ubuntu/.paw-val3:11857:11856:11290:11517:11860"
    ["val4"]="services-testnet:/home/ubuntu/.paw-val4:11957:11956:11390:11617:11960"
)

DRY_RUN=false
TARGET_VALIDATOR=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run) DRY_RUN=true; shift ;;
        --validator) TARGET_VALIDATOR="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

log() {
    echo "[$(date '+%H:%M:%S')] $1"
}

backup_config() {
    local server=$1
    local home=$2
    local name=$3

    log "Backing up $name config from $server..."
    mkdir -p "$BACKUP_DIR/$name"

    if [[ "$DRY_RUN" == "false" ]]; then
        ssh "$server" "cp $home/config/config.toml $home/config/config.toml.bak" || true
        ssh "$server" "cp $home/config/app.toml $home/config/app.toml.bak" || true
        scp "$server:$home/config/config.toml" "$BACKUP_DIR/$name/" 2>/dev/null || true
        scp "$server:$home/config/app.toml" "$BACKUP_DIR/$name/" 2>/dev/null || true
    else
        log "[DRY-RUN] Would backup config from $server:$home"
    fi
}

apply_config_patches() {
    local server=$1
    local home=$2
    local name=$3
    local rpc_port=$4
    local p2p_port=$5
    local grpc_port=$6
    local rest_port=$7
    local prom_port=$8

    log "Applying hardened config patches to $name..."

    # Generate patch commands
    local patch_cmds="
# Enable Prometheus
sed -i 's/^prometheus = false/prometheus = true/' $home/config/config.toml

# Set Prometheus port
sed -i 's/^prometheus_listen_addr = .*/prometheus_listen_addr = \":$prom_port\"/' $home/config/config.toml

# Optimize mempool
sed -i 's/^size = 5000/size = 2000/' $home/config/config.toml
sed -i 's/^max_txs_bytes = 1073741824/max_txs_bytes = 536870912/' $home/config/config.toml
sed -i 's/^cache_size = 10000/cache_size = 5000/' $home/config/config.toml

# Tune consensus for cross-server latency
sed -i 's/^timeout_prevote = \"1s\"/timeout_prevote = \"1500ms\"/' $home/config/config.toml
sed -i 's/^timeout_precommit = \"1s\"/timeout_precommit = \"1500ms\"/' $home/config/config.toml

# Tune P2P
sed -i 's/^max_num_inbound_peers = 40/max_num_inbound_peers = 25/' $home/config/config.toml
sed -i 's/^send_rate = 5120000/send_rate = 10240000/' $home/config/config.toml
sed -i 's/^recv_rate = 5120000/recv_rate = 10240000/' $home/config/config.toml
sed -i 's/^handshake_timeout = \"20s\"/handshake_timeout = \"30s\"/' $home/config/config.toml
sed -i 's/^dial_timeout = \"3s\"/dial_timeout = \"5s\"/' $home/config/config.toml

# Tune RPC limits
sed -i 's/^max_open_connections = 900/max_open_connections = 450/' $home/config/config.toml
sed -i 's/^grpc_max_open_connections = 900/grpc_max_open_connections = 450/' $home/config/config.toml

# Double sign protection
sed -i 's/^double_sign_check_height = 0/double_sign_check_height = 10/' $home/config/config.toml

# Close slow WebSocket clients
sed -i 's/^experimental_close_on_slow_client = false/experimental_close_on_slow_client = true/' $home/config/config.toml

# Enable telemetry in app.toml
sed -i 's/^enabled = false/enabled = true/' $home/config/app.toml

# Optimize pruning
sed -i 's/^pruning = \"default\"/pruning = \"custom\"/' $home/config/app.toml
sed -i 's/^pruning-keep-recent = \"0\"/pruning-keep-recent = \"100000\"/' $home/config/app.toml
sed -i 's/^pruning-interval = \"0\"/pruning-interval = \"100\"/' $home/config/app.toml

# Enable snapshots
sed -i 's/^snapshot-interval = 0/snapshot-interval = 1000/' $home/config/app.toml
"

    if [[ "$DRY_RUN" == "false" ]]; then
        ssh "$server" "$patch_cmds"
    else
        log "[DRY-RUN] Would apply patches:"
        echo "$patch_cmds" | head -20
        log "[DRY-RUN] ... (truncated)"
    fi
}

restart_validator() {
    local server=$1
    local name=$2
    local num=$3

    log "Restarting $name on $server..."

    if [[ "$DRY_RUN" == "false" ]]; then
        ssh "$server" "sudo systemctl restart pawd-val@$num"
        sleep 5
        local status=$(ssh "$server" "systemctl is-active pawd-val@$num" || echo "unknown")
        if [[ "$status" == "active" ]]; then
            log "$name restarted successfully"
        else
            log "WARNING: $name may not have started correctly (status: $status)"
        fi
    else
        log "[DRY-RUN] Would restart pawd-val@$num on $server"
    fi
}

deploy_to_validator() {
    local name=$1
    local config=${VALIDATORS[$name]}

    IFS=':' read -r server home rpc p2p grpc rest prom <<< "$config"
    local num="${name#val}"

    log "=== Deploying to $name ==="
    backup_config "$server" "$home" "$name"
    apply_config_patches "$server" "$home" "$name" "$rpc" "$p2p" "$grpc" "$rest" "$prom"
    restart_validator "$server" "$name" "$num"
    log ""
}

# Main
log "PAW Testnet Hardened Config Deployment"
log "======================================"
[[ "$DRY_RUN" == "true" ]] && log "DRY RUN MODE - No changes will be made"
log ""

mkdir -p "$BACKUP_DIR"
log "Backups will be saved to: $BACKUP_DIR"
log ""

if [[ -n "$TARGET_VALIDATOR" ]]; then
    if [[ -v "VALIDATORS[$TARGET_VALIDATOR]" ]]; then
        deploy_to_validator "$TARGET_VALIDATOR"
    else
        echo "Unknown validator: $TARGET_VALIDATOR"
        echo "Valid options: ${!VALIDATORS[*]}"
        exit 1
    fi
else
    # Deploy to all validators in sequence
    for name in val1 val2 val3 val4; do
        deploy_to_validator "$name"
        [[ "$DRY_RUN" == "false" ]] && sleep 10  # Wait between validators
    done
fi

log "Deployment complete!"
log ""
log "Verify with: ./validator-health-monitor.sh"
log "Rollback with backups in: $BACKUP_DIR"
