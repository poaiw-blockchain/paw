#!/bin/bash
# PAW Blockchain - Chain Upgrade Script
# Performs coordinated upgrade of blockchain nodes

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default values
NAMESPACE="${NAMESPACE:-paw-blockchain}"
NEW_VERSION="${NEW_VERSION:-}"
UPGRADE_HEIGHT="${UPGRADE_HEIGHT:-}"
BACKUP_BEFORE_UPGRADE="${BACKUP_BEFORE_UPGRADE:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if [ -z "$NEW_VERSION" ]; then
        log_error "New version is required. Use --version option."
        exit 1
    fi

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed."
        exit 1
    fi

    if ! kubectl cluster-info &> /dev/null; then
        log_error "Not connected to a Kubernetes cluster."
        exit 1
    fi

    log_info "Prerequisites check passed"
}

# Get current chain height
get_current_height() {
    log_info "Getting current chain height..."

    local POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l component=node -o jsonpath='{.items[0].metadata.name}')
    if [ -z "$POD_NAME" ]; then
        log_error "No running nodes found"
        exit 1
    fi

    CURRENT_HEIGHT=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- \
        curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

    log_info "Current chain height: $CURRENT_HEIGHT"

    if [ -z "$UPGRADE_HEIGHT" ]; then
        # Set upgrade height to current + 1000 blocks (~4000 seconds = ~1 hour)
        UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 1000))
        log_info "Calculated upgrade height: $UPGRADE_HEIGHT"
    fi
}

# Backup current state
backup_state() {
    if [ "$BACKUP_BEFORE_UPGRADE" = "true" ]; then
        log_step "Backing up current state..."

        "$SCRIPT_DIR/backup-state.sh" --namespace "$NAMESPACE" --tag "pre-upgrade-$NEW_VERSION"

        log_info "Backup completed"
    else
        log_warn "Skipping backup (not recommended)"
    fi
}

# Submit upgrade proposal
submit_upgrade_proposal() {
    log_step "Submitting upgrade proposal..."

    log_info "Creating upgrade proposal for height $UPGRADE_HEIGHT"

    # Get a validator pod
    local VALIDATOR_POD=$(kubectl get pods -n "$NAMESPACE" -l component=validator -o jsonpath='{.items[0].metadata.name}')

    if [ -z "$VALIDATOR_POD" ]; then
        log_error "No validator pod found"
        exit 1
    fi

    # Submit upgrade proposal
    log_info "Submitting proposal from $VALIDATOR_POD..."

    kubectl exec -n "$NAMESPACE" "$VALIDATOR_POD" -- pawd tx gov submit-proposal software-upgrade \
        "$NEW_VERSION" \
        --title "Upgrade to $NEW_VERSION" \
        --description "Upgrade PAW chain to version $NEW_VERSION" \
        --upgrade-height "$UPGRADE_HEIGHT" \
        --deposit 10000000upaw \
        --from validator \
        --chain-id paw-1 \
        --yes

    log_info "Upgrade proposal submitted"
    log_info "Validators should vote on the proposal before height $UPGRADE_HEIGHT"
}

# Update deployment images
update_images() {
    log_step "Updating deployment images to $NEW_VERSION..."

    # Update node deployment
    kubectl set image deployment/paw-node \
        -n "$NAMESPACE" \
        paw=paw-chain/paw-node:"$NEW_VERSION"

    # Update validator statefulset
    kubectl set image statefulset/paw-validator \
        -n "$NAMESPACE" \
        validator=paw-chain/paw-validator:"$NEW_VERSION"

    log_info "Images updated"
}

# Monitor upgrade progress
monitor_upgrade() {
    log_step "Monitoring upgrade progress..."

    log_info "Waiting for upgrade height $UPGRADE_HEIGHT..."

    while true; do
        local POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l component=node -o jsonpath='{.items[0].metadata.name}')
        if [ -z "$POD_NAME" ]; then
            log_warn "No running nodes found, waiting..."
            sleep 10
            continue
        fi

        local HEIGHT=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- \
            curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' || echo "0")

        if [ -z "$HEIGHT" ] || [ "$HEIGHT" = "null" ]; then
            log_warn "Could not fetch height, waiting..."
            sleep 10
            continue
        fi

        log_info "Current height: $HEIGHT / Upgrade height: $UPGRADE_HEIGHT"

        if [ "$HEIGHT" -ge "$UPGRADE_HEIGHT" ]; then
            log_info "Upgrade height reached!"
            break
        fi

        sleep 30
    done
}

# Verify upgrade
verify_upgrade() {
    log_step "Verifying upgrade..."

    # Wait for nodes to restart and sync
    log_info "Waiting for nodes to restart..."
    sleep 60

    # Check node status
    local POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l component=node -o jsonpath='{.items[0].metadata.name}')

    if [ -z "$POD_NAME" ]; then
        log_error "No running nodes found after upgrade"
        return 1
    fi

    # Check if node is catching up
    local CATCHING_UP=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- \
        curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.catching_up' || echo "true")

    if [ "$CATCHING_UP" = "false" ]; then
        log_info "Node is synced and running new version"
    else
        log_warn "Node is still catching up"
    fi

    # Check all pods are running
    local RUNNING_NODES=$(kubectl get pods -n "$NAMESPACE" -l component=node --field-selector=status.phase=Running --no-headers | wc -l)
    local RUNNING_VALIDATORS=$(kubectl get pods -n "$NAMESPACE" -l component=validator --field-selector=status.phase=Running --no-headers | wc -l)

    log_info "Running nodes: $RUNNING_NODES"
    log_info "Running validators: $RUNNING_VALIDATORS"

    # Get version info
    local VERSION=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- pawd version 2>/dev/null || echo "unknown")
    log_info "Running version: $VERSION"

    return 0
}

# Rollback upgrade (if needed)
rollback_upgrade() {
    log_step "Rolling back upgrade..."

    log_warn "Restoring from backup..."
    "$SCRIPT_DIR/restore-state.sh" --namespace "$NAMESPACE" --backup "pre-upgrade-$NEW_VERSION"

    log_warn "Reverting images to previous version..."
    # This would need the previous version passed as parameter
    # For now, just log the command
    log_info "To rollback images, run:"
    echo "  kubectl rollout undo deployment/paw-node -n $NAMESPACE"
    echo "  kubectl rollout undo statefulset/paw-validator -n $NAMESPACE"
}

# Display upgrade summary
display_summary() {
    log_info "Upgrade Summary:"
    echo "  Namespace: $NAMESPACE"
    echo "  New Version: $NEW_VERSION"
    echo "  Upgrade Height: $UPGRADE_HEIGHT"
    echo ""

    log_info "Current Pod Status:"
    kubectl get pods -n "$NAMESPACE" -l app=paw

    echo ""
    log_info "To check upgrade status:"
    echo "  kubectl exec -n $NAMESPACE <pod-name> -- pawd query upgrade plan"

    echo ""
    log_info "To view logs:"
    echo "  kubectl logs -f -n $NAMESPACE -l app=paw"
}

# Main upgrade function
main() {
    log_info "Starting PAW chain upgrade process..."
    log_info "Configuration:"
    echo "  Namespace: $NAMESPACE"
    echo "  New Version: $NEW_VERSION"
    echo "  Upgrade Height: ${UPGRADE_HEIGHT:-auto}"
    echo "  Backup Before Upgrade: $BACKUP_BEFORE_UPGRADE"
    echo ""

    log_warn "WARNING: This will upgrade the entire blockchain network."
    log_warn "Make sure all validators have agreed to this upgrade!"
    echo ""

    read -p "Proceed with upgrade? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Upgrade cancelled"
        exit 0
    fi

    check_prerequisites
    get_current_height
    backup_state

    log_step "Upgrade process started"
    log_info "Phase 1: Submit upgrade proposal"
    submit_upgrade_proposal

    log_info "Phase 2: Update container images"
    update_images

    log_info "Phase 3: Monitor upgrade progress"
    monitor_upgrade

    log_info "Phase 4: Verify upgrade"
    if verify_upgrade; then
        log_info "Upgrade completed successfully!"
        display_summary
    else
        log_error "Upgrade verification failed!"
        log_warn "Consider rolling back if issues persist"
        read -p "Rollback upgrade? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rollback_upgrade
        fi
        exit 1
    fi
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Perform coordinated upgrade of PAW blockchain

OPTIONS:
    -n, --namespace         Kubernetes namespace (default: paw-blockchain)
    -v, --version          New version to upgrade to (required)
    -h, --upgrade-height   Block height for upgrade (default: auto)
    -b, --backup           Backup before upgrade (default: true)
    --no-backup            Skip backup before upgrade
    --help                 Show this help message

EXAMPLES:
    # Upgrade to v1.1.0 with auto height
    $0 --version v1.1.0

    # Upgrade at specific height
    $0 --version v1.1.0 --upgrade-height 1000000

    # Upgrade without backup (not recommended)
    $0 --version v1.1.0 --no-backup

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -v|--version)
            NEW_VERSION="$2"
            shift 2
            ;;
        -h|--upgrade-height)
            UPGRADE_HEIGHT="$2"
            shift 2
            ;;
        -b|--backup)
            BACKUP_BEFORE_UPGRADE="true"
            shift
            ;;
        --no-backup)
            BACKUP_BEFORE_UPGRADE="false"
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main function
main
