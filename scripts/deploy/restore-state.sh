#!/bin/bash
# PAW Blockchain - State Restore Script
# Restores blockchain state from backup

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default values
NAMESPACE="${NAMESPACE:-paw-blockchain}"
BACKUP_DIR="${BACKUP_DIR:-/tmp/paw-backups}"
BACKUP_TAG="${BACKUP_TAG:-}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:-}"
RESTORE_VALIDATORS="${RESTORE_VALIDATORS:-true}"
RESTORE_CONFIG="${RESTORE_CONFIG:-true}"

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

    if [ -z "$BACKUP_TAG" ]; then
        log_error "Backup tag is required. Use --backup option."
        list_available_backups
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

# List available backups
list_available_backups() {
    log_info "Available backups in $BACKUP_DIR:"
    if [ -d "$BACKUP_DIR" ]; then
        ls -lh "$BACKUP_DIR"/*.tar.gz 2>/dev/null || ls -d "$BACKUP_DIR"/*/ 2>/dev/null || log_info "No backups found"
    else
        log_info "Backup directory does not exist"
    fi
}

# Download from S3 if needed
download_from_s3() {
    if [ -n "$BACKUP_S3_BUCKET" ]; then
        log_info "Downloading backup from S3..."

        if ! command -v aws &> /dev/null; then
            log_error "AWS CLI not installed"
            exit 1
        fi

        aws s3 cp "s3://$BACKUP_S3_BUCKET/backups/$BACKUP_TAG.tar.gz" "$BACKUP_DIR/"

        if [ $? -ne 0 ]; then
            log_error "Failed to download backup from S3"
            exit 1
        fi

        log_info "Backup downloaded from S3"
    fi
}

# Extract backup if compressed
extract_backup() {
    local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG"

    if [ -f "$BACKUP_PATH.tar.gz" ]; then
        log_info "Extracting backup archive..."

        cd "$BACKUP_DIR"
        tar -xzf "$BACKUP_TAG.tar.gz"

        if [ $? -ne 0 ]; then
            log_error "Failed to extract backup"
            exit 1
        fi

        log_info "Backup extracted to $BACKUP_PATH"
    elif [ -d "$BACKUP_PATH" ]; then
        log_info "Using uncompressed backup at $BACKUP_PATH"
    else
        log_error "Backup not found: $BACKUP_PATH or $BACKUP_PATH.tar.gz"
        exit 1
    fi
}

# Stop validators before restore
stop_validators() {
    log_warn "Stopping validators..."

    # Scale down validators to 0
    kubectl scale statefulset/paw-validator -n "$NAMESPACE" --replicas=0

    # Wait for pods to terminate
    log_info "Waiting for validator pods to terminate..."
    kubectl wait --for=delete pod -n "$NAMESPACE" -l component=validator --timeout=5m || true

    log_info "Validators stopped"
}

# Restore validator keys
restore_validator_keys() {
    if [ "$RESTORE_VALIDATORS" = "true" ]; then
        log_info "Restoring validator keys..."

        local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/validator-keys"

        if [ ! -d "$BACKUP_PATH" ]; then
            log_warn "No validator keys backup found at $BACKUP_PATH"
            return
        fi

        # Delete existing secret
        kubectl delete secret paw-validator-keys -n "$NAMESPACE" --ignore-not-found=true

        # Find validator directories
        for validator_dir in "$BACKUP_PATH"/*; do
            if [ -d "$validator_dir" ]; then
                local VALIDATOR_NAME=$(basename "$validator_dir")
                log_info "Restoring keys for $VALIDATOR_NAME..."

                # Create secret from backed up keys
                if [ -f "$validator_dir/priv_validator_key.json" ] && [ -f "$validator_dir/node_key.json" ]; then
                    kubectl create secret generic paw-validator-keys \
                        --from-file=priv_validator_key.json="$validator_dir/priv_validator_key.json" \
                        --from-file=node_key.json="$validator_dir/node_key.json" \
                        --namespace="$NAMESPACE" || log_warn "Failed to create secret for $VALIDATOR_NAME"

                    # Restore priv_validator_state.json (critical for double-sign prevention)
                    if [ -f "$validator_dir/priv_validator_state.json" ]; then
                        # Get PVC name
                        local PVC="data-$VALIDATOR_NAME"

                        # Create temporary pod to restore state file
                        cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: restore-state-$VALIDATOR_NAME
  namespace: $NAMESPACE
spec:
  containers:
  - name: restore
    image: busybox
    command: ["/bin/sh", "-c", "sleep 3600"]
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: $PVC
  restartPolicy: Never
EOF

                        # Wait for pod to be ready
                        kubectl wait --for=condition=Ready pod/restore-state-$VALIDATOR_NAME -n "$NAMESPACE" --timeout=2m

                        # Copy state file
                        kubectl cp "$validator_dir/priv_validator_state.json" \
                            "$NAMESPACE/restore-state-$VALIDATOR_NAME:/data/priv_validator_state.json"

                        # Delete temporary pod
                        kubectl delete pod restore-state-$VALIDATOR_NAME -n "$NAMESPACE"

                        log_info "Restored priv_validator_state.json for $VALIDATOR_NAME"
                    fi
                fi
            fi
        done

        log_info "Validator keys restored"
    else
        log_info "Skipping validator keys restore"
    fi
}

# Restore genesis file
restore_genesis() {
    log_info "Restoring genesis file..."

    local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/genesis"

    if [ ! -f "$BACKUP_PATH/genesis.json" ]; then
        log_warn "No genesis file backup found at $BACKUP_PATH"
        return
    fi

    # Update genesis in ConfigMap
    kubectl delete secret paw-genesis -n "$NAMESPACE" --ignore-not-found=true

    kubectl create secret generic paw-genesis \
        --from-file=genesis.json="$BACKUP_PATH/genesis.json" \
        --namespace="$NAMESPACE"

    log_info "Genesis file restored"
}

# Restore Kubernetes configuration
restore_configuration() {
    if [ "$RESTORE_CONFIG" = "true" ]; then
        log_info "Restoring Kubernetes configuration..."

        local BACKUP_PATH="$BACKUP_DIR/$BACKUP_TAG/k8s-config"

        if [ ! -d "$BACKUP_PATH" ]; then
            log_warn "No Kubernetes configuration backup found at $BACKUP_PATH"
            return
        fi

        # Restore ConfigMaps (skip if already exists)
        if [ -f "$BACKUP_PATH/configmaps.yaml" ]; then
            kubectl apply -f "$BACKUP_PATH/configmaps.yaml" || log_warn "Failed to restore ConfigMaps"
        fi

        log_info "Kubernetes configuration restored"
    else
        log_info "Skipping Kubernetes configuration restore"
    fi
}

# Restore from volume snapshots
restore_from_snapshots() {
    log_info "Checking for volume snapshots..."

    local SNAPSHOTS=$(kubectl get volumesnapshot -n "$NAMESPACE" --no-headers 2>/dev/null | grep "$BACKUP_TAG" | awk '{print $1}')

    if [ -z "$SNAPSHOTS" ]; then
        log_warn "No volume snapshots found for backup $BACKUP_TAG"
        return
    fi

    log_info "Found snapshots to restore:"
    echo "$SNAPSHOTS"

    read -p "Restore from volume snapshots? This will replace existing data! (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Skipping snapshot restore"
        return
    fi

    for snapshot in $SNAPSHOTS; do
        log_info "Restoring from snapshot: $snapshot..."

        # Extract pod name from snapshot name
        local POD_NAME=$(echo "$snapshot" | sed "s/-snapshot-$BACKUP_TAG//")
        local PVC="data-$POD_NAME"

        # Delete existing PVC
        kubectl delete pvc "$PVC" -n "$NAMESPACE" --wait=true

        # Create PVC from snapshot
        cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: $PVC
  namespace: $NAMESPACE
spec:
  accessModes:
  - ReadWriteOnce
  storageClassName: paw-storage-ssd
  dataSource:
    name: $snapshot
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  resources:
    requests:
      storage: 1Ti
EOF

        log_info "PVC $PVC restored from snapshot"
    done
}

# Start validators after restore
start_validators() {
    log_info "Starting validators..."

    # Scale up validators
    kubectl scale statefulset/paw-validator -n "$NAMESPACE" --replicas=3

    # Wait for pods to be ready
    log_info "Waiting for validator pods to be ready..."
    kubectl wait --for=condition=Ready pod -n "$NAMESPACE" -l component=validator --timeout=10m || \
        log_warn "Some validators did not become ready"

    log_info "Validators started"
}

# Verify restore
verify_restore() {
    log_info "Verifying restore..."

    # Check if pods are running
    local RUNNING_VALIDATORS=$(kubectl get pods -n "$NAMESPACE" -l component=validator --field-selector=status.phase=Running --no-headers | wc -l)

    log_info "Running validators: $RUNNING_VALIDATORS"

    # Check validator status
    local POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l component=validator -o jsonpath='{.items[0].metadata.name}')

    if [ -n "$POD_NAME" ]; then
        log_info "Checking validator status..."
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- pawcli status --node tcp://localhost:26657 || \
            log_warn "Validator not responding to status checks"
    fi

    log_info "Restore verification completed"
}

# Display restore info
display_info() {
    log_info "Restore completed!"
    echo ""
    log_info "Restored from backup: $BACKUP_TAG"
    echo ""

    log_info "Current Status:"
    kubectl get pods -n "$NAMESPACE" -l app=paw

    echo ""
    log_info "To check node status:"
    echo "  kubectl exec -n $NAMESPACE <pod-name> -- pawcli status"

    echo ""
    log_warn "IMPORTANT: Verify that validators are producing blocks before considering restore successful"
}

# Main restore function
main() {
    log_info "Starting PAW blockchain restore..."
    log_info "Configuration:"
    echo "  Namespace: $NAMESPACE"
    echo "  Backup Tag: $BACKUP_TAG"
    echo "  Backup Dir: $BACKUP_DIR"
    echo "  Restore Validators: $RESTORE_VALIDATORS"
    echo "  Restore Config: $RESTORE_CONFIG"
    echo ""

    log_warn "WARNING: This will restore blockchain state from backup."
    log_warn "This operation will stop validators and may cause downtime!"
    echo ""

    read -p "Proceed with restore? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled"
        exit 0
    fi

    check_prerequisites
    download_from_s3
    extract_backup
    stop_validators
    restore_validator_keys
    restore_genesis
    restore_configuration
    restore_from_snapshots
    start_validators
    verify_restore
    display_info

    log_info "Restore completed successfully"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Restore PAW blockchain state from backup

OPTIONS:
    -n, --namespace          Kubernetes namespace (default: paw-blockchain)
    -b, --backup             Backup tag to restore (required)
    -d, --backup-dir         Backup directory (default: /tmp/paw-backups)
    -s, --s3-bucket          S3 bucket to download from (optional)
    --skip-validators        Don't restore validator keys
    --skip-config            Don't restore Kubernetes config
    -h, --help               Show this help message

EXAMPLES:
    # List available backups
    $0 --backup list

    # Restore from local backup
    $0 --backup 20240101-120000

    # Restore from S3 backup
    $0 --backup 20240101-120000 --s3-bucket my-backup-bucket

    # Restore without validator keys
    $0 --backup 20240101-120000 --skip-validators

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -b|--backup)
            if [ "$2" = "list" ]; then
                list_available_backups
                exit 0
            fi
            BACKUP_TAG="$2"
            shift 2
            ;;
        -d|--backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        -s|--s3-bucket)
            BACKUP_S3_BUCKET="$2"
            shift 2
            ;;
        --skip-validators)
            RESTORE_VALIDATORS="false"
            shift
            ;;
        --skip-config)
            RESTORE_CONFIG="false"
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
