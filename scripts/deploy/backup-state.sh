#!/bin/bash
# PAW Blockchain - State Backup Script
# Creates backups of blockchain state and validator keys

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default values
NAMESPACE="${NAMESPACE:-paw-blockchain}"
BACKUP_DIR="${BACKUP_DIR:-/tmp/paw-backups}"
BACKUP_TAG="${BACKUP_TAG:-$(date +%Y%m%d-%H%M%S)}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:-}"
COMPRESS="${COMPRESS:-true}"
INCLUDE_DATA="${INCLUDE_DATA:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed."
        exit 1
    fi

    if ! kubectl cluster-info &> /dev/null; then
        log_error "Not connected to a Kubernetes cluster."
        exit 1
    fi

    # Create backup directory
    mkdir -p "$BACKUP_DIR/$BACKUP_TAG"

    log_info "Prerequisites check passed"
}

# Backup validator keys
backup_validator_keys() {
    log_info "Backing up validator keys..."

    local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/validator-keys"
    mkdir -p "$BACKUP_PATH"

    # Get validator pods
    local VALIDATORS=$(kubectl get pods -n "$NAMESPACE" -l component=validator -o jsonpath='{.items[*].metadata.name}')

    if [ -z "$VALIDATORS" ]; then
        log_warn "No validator pods found"
        return
    fi

    for pod in $VALIDATORS; do
        log_info "Backing up keys from $pod..."

        mkdir -p "$BACKUP_PATH/$pod"

        # Copy priv_validator_key.json
        kubectl cp "$NAMESPACE/$pod:/home/validator/.paw/config/priv_validator_key.json" \
            "$BACKUP_PATH/$pod/priv_validator_key.json" 2>/dev/null || \
            log_warn "Could not backup priv_validator_key.json from $pod"

        # Copy node_key.json
        kubectl cp "$NAMESPACE/$pod:/home/validator/.paw/config/node_key.json" \
            "$BACKUP_PATH/$pod/node_key.json" 2>/dev/null || \
            log_warn "Could not backup node_key.json from $pod"

        # Copy priv_validator_state.json
        kubectl cp "$NAMESPACE/$pod:/home/validator/.paw/data/priv_validator_state.json" \
            "$BACKUP_PATH/$pod/priv_validator_state.json" 2>/dev/null || \
            log_warn "Could not backup priv_validator_state.json from $pod"
    done

    log_info "Validator keys backed up to $BACKUP_PATH"
}

# Backup genesis file
backup_genesis() {
    log_info "Backing up genesis file..."

    local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/genesis"
    mkdir -p "$BACKUP_PATH"

    # Get any pod
    local POD=$(kubectl get pods -n "$NAMESPACE" -l app=paw -o jsonpath='{.items[0].metadata.name}')

    if [ -z "$POD" ]; then
        log_error "No pods found"
        return 1
    fi

    # Determine if it's a validator or node pod
    if kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.metadata.labels.component}' | grep -q "validator"; then
        kubectl cp "$NAMESPACE/$POD:/home/validator/.paw/config/genesis.json" \
            "$BACKUP_PATH/genesis.json" || \
            log_error "Failed to backup genesis file"
    else
        kubectl cp "$NAMESPACE/$POD:/home/paw/.paw/config/genesis.json" \
            "$BACKUP_PATH/genesis.json" || \
            log_error "Failed to backup genesis file"
    fi

    log_info "Genesis file backed up to $BACKUP_PATH"
}

# Backup chain data
backup_chain_data() {
    if [ "$INCLUDE_DATA" = "true" ]; then
        log_info "Backing up chain data..."
        log_warn "This may take a long time depending on chain size..."

        local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/data"
        mkdir -p "$BACKUP_PATH"

        # Use volume snapshots for efficiency
        log_info "Creating volume snapshots..."

        local VALIDATORS=$(kubectl get pods -n "$NAMESPACE" -l component=validator -o jsonpath='{.items[*].metadata.name}')

        for pod in $VALIDATORS; do
            log_info "Creating snapshot for $pod data volume..."

            local PVC="data-$pod"

            # Create VolumeSnapshot
            cat <<EOF | kubectl apply -f -
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: ${pod}-snapshot-${BACKUP_TAG}
  namespace: $NAMESPACE
spec:
  volumeSnapshotClassName: paw-snapshot-class
  source:
    persistentVolumeClaimName: $PVC
EOF

            log_info "Snapshot created for $pod: ${pod}-snapshot-${BACKUP_TAG}"
        done

        log_info "Chain data snapshots created"
    else
        log_info "Skipping chain data backup (use --include-data to enable)"
    fi
}

# Backup configuration
backup_configuration() {
    log_info "Backing up Kubernetes configuration..."

    local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/k8s-config"
    mkdir -p "$BACKUP_PATH"

    # Backup ConfigMaps
    kubectl get configmap -n "$NAMESPACE" -o yaml > "$BACKUP_PATH/configmaps.yaml"

    # Backup Services
    kubectl get svc -n "$NAMESPACE" -o yaml > "$BACKUP_PATH/services.yaml"

    # Backup Deployments
    kubectl get deployment -n "$NAMESPACE" -o yaml > "$BACKUP_PATH/deployments.yaml"

    # Backup StatefulSets
    kubectl get statefulset -n "$NAMESPACE" -o yaml > "$BACKUP_PATH/statefulsets.yaml"

    # Backup PVCs
    kubectl get pvc -n "$NAMESPACE" -o yaml > "$BACKUP_PATH/pvcs.yaml"

    log_info "Kubernetes configuration backed up to $BACKUP_PATH"
}

# Compress backup
compress_backup() {
    if [ "$COMPRESS" = "true" ]; then
        log_info "Compressing backup..."

        cd "$BACKUP_DIR"
        tar -czf "$BACKUP_TAG.tar.gz" "$BACKUP_TAG/"

        if [ $? -eq 0 ]; then
            log_info "Backup compressed: $BACKUP_DIR/$BACKUP_TAG.tar.gz"
            log_info "Removing uncompressed backup..."
            rm -rf "$BACKUP_TAG"
        else
            log_error "Failed to compress backup"
        fi
    fi
}

# Upload to S3
upload_to_s3() {
    if [ -n "$BACKUP_S3_BUCKET" ]; then
        log_info "Uploading backup to S3..."

        if ! command -v aws &> /dev/null; then
            log_warn "AWS CLI not installed, skipping S3 upload"
            return
        fi

        local BACKUP_FILE="$BACKUP_DIR/$BACKUP_TAG.tar.gz"
        if [ ! -f "$BACKUP_FILE" ]; then
            BACKUP_FILE="$BACKUP_DIR/$BACKUP_TAG"
        fi

        aws s3 cp "$BACKUP_FILE" "s3://$BACKUP_S3_BUCKET/backups/$(basename "$BACKUP_FILE")"

        if [ $? -eq 0 ]; then
            log_info "Backup uploaded to s3://$BACKUP_S3_BUCKET/backups/$(basename "$BACKUP_FILE")"
        else
            log_error "Failed to upload backup to S3"
        fi
    fi
}

# Display backup info
display_info() {
    log_info "Backup completed successfully!"
    echo ""
    log_info "Backup Information:"
    echo "  Tag: $BACKUP_TAG"
    echo "  Location: $BACKUP_DIR/$BACKUP_TAG"
    echo ""

    log_info "Backup Contents:"
    if [ -d "$BACKUP_DIR/$BACKUP_TAG" ]; then
        du -sh "$BACKUP_DIR/$BACKUP_TAG"/*
    elif [ -f "$BACKUP_DIR/$BACKUP_TAG.tar.gz" ]; then
        ls -lh "$BACKUP_DIR/$BACKUP_TAG.tar.gz"
    fi

    echo ""
    log_info "To restore this backup, run:"
    echo "  $SCRIPT_DIR/restore-state.sh --backup $BACKUP_TAG"

    echo ""
    log_warn "IMPORTANT: Store this backup in a secure location!"
}

# Main backup function
main() {
    log_info "Starting PAW blockchain backup..."
    log_info "Configuration:"
    echo "  Namespace: $NAMESPACE"
    echo "  Backup Dir: $BACKUP_DIR"
    echo "  Backup Tag: $BACKUP_TAG"
    echo "  Compress: $COMPRESS"
    echo "  Include Data: $INCLUDE_DATA"
    echo "  S3 Bucket: ${BACKUP_S3_BUCKET:-none}"
    echo ""

    check_prerequisites
    backup_validator_keys
    backup_genesis
    backup_configuration
    backup_chain_data
    compress_backup
    upload_to_s3
    display_info

    log_info "Backup completed successfully"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Create backup of PAW blockchain state

OPTIONS:
    -n, --namespace      Kubernetes namespace (default: paw-blockchain)
    -d, --backup-dir     Backup directory (default: /tmp/paw-backups)
    -t, --tag            Backup tag (default: timestamp)
    -s, --s3-bucket      S3 bucket for upload (optional)
    -c, --compress       Compress backup (default: true)
    --no-compress        Don't compress backup
    --include-data       Include chain data (default: false)
    -h, --help           Show this help message

EXAMPLES:
    # Basic backup
    $0

    # Backup with custom tag
    $0 --tag pre-upgrade

    # Backup including chain data
    $0 --include-data

    # Backup and upload to S3
    $0 --s3-bucket my-backup-bucket

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -d|--backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        -t|--tag)
            BACKUP_TAG="$2"
            shift 2
            ;;
        -s|--s3-bucket)
            BACKUP_S3_BUCKET="$2"
            shift 2
            ;;
        -c|--compress)
            COMPRESS="true"
            shift
            ;;
        --no-compress)
            COMPRESS="false"
            shift
            ;;
        --include-data)
            INCLUDE_DATA="true"
            shift
            ;;
        -h|--help)
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
