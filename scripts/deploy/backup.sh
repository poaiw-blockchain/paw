#!/bin/bash

# PAW Blockchain - Backup Script
# Creates backups of blockchain data and configurations

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
BACKUP_DIR="${BACKUP_DIR:-$HOME/paw-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DEPLOYMENT_TYPE="${1:-local}"  # local, docker, k8s

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

create_backup_dir() {
    mkdir -p "$BACKUP_DIR"
    log_info "Backup directory: $BACKUP_DIR"
}

backup_local() {
    log_step "Backing up local installation..."

    HOME_DIR="$HOME/.paw"
    BACKUP_FILE="$BACKUP_DIR/paw-local-backup-$TIMESTAMP.tar.gz"

    if [ ! -d "$HOME_DIR" ]; then
        log_error "PAW home directory not found: $HOME_DIR"
        exit 1
    fi

    # Create backup
    tar czf "$BACKUP_FILE" -C "$HOME" .paw

    log_info "Backup created: $BACKUP_FILE"
    log_info "Size: $(du -h "$BACKUP_FILE" | cut -f1)"

    # Create metadata file
    cat > "$BACKUP_DIR/paw-local-backup-$TIMESTAMP.metadata" << EOF
Backup Type: Local Installation
Timestamp: $(date)
Chain ID: $(grep 'chain-id' "$HOME_DIR/config/config.toml" | cut -d'"' -f2 || echo "unknown")
Last Block Height: $(curl -s http://localhost:26657/status | jq -r .result.sync_info.latest_block_height 2>/dev/null || echo "unknown")
Backup File: $BACKUP_FILE
Size: $(du -h "$BACKUP_FILE" | cut -f1)
EOF

    log_info "Metadata saved"
}

backup_docker() {
    log_step "Backing up Docker deployment..."

    DOCKER_VOLUME="paw-data"
    BACKUP_FILE="$BACKUP_DIR/paw-docker-backup-$TIMESTAMP.tar.gz"

    if ! docker volume inspect "$DOCKER_VOLUME" &> /dev/null; then
        log_error "Docker volume not found: $DOCKER_VOLUME"
        exit 1
    fi

    # Backup volume data
    docker run --rm \
        -v "$DOCKER_VOLUME:/data:ro" \
        -v "$BACKUP_DIR:/backup" \
        alpine tar czf "/backup/paw-docker-backup-$TIMESTAMP.tar.gz" -C / data

    log_info "Backup created: $BACKUP_FILE"
    log_info "Size: $(du -h "$BACKUP_FILE" | cut -f1)"

    # Create metadata
    cat > "$BACKUP_DIR/paw-docker-backup-$TIMESTAMP.metadata" << EOF
Backup Type: Docker Deployment
Timestamp: $(date)
Volume: $DOCKER_VOLUME
Container: $(docker ps --filter "name=paw-node" --format "{{.Names}}" | head -1)
Backup File: $BACKUP_FILE
Size: $(du -h "$BACKUP_FILE" | cut -f1)
EOF

    log_info "Metadata saved"
}

backup_kubernetes() {
    log_step "Backing up Kubernetes deployment..."

    NAMESPACE="${PAW_NAMESPACE:-paw-blockchain}"
    BACKUP_FILE="$BACKUP_DIR/paw-k8s-backup-$TIMESTAMP.tar.gz"

    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi

    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace not found: $NAMESPACE"
        exit 1
    fi

    # Create temporary backup directory
    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    # Backup configurations
    log_info "Backing up Kubernetes configurations..."
    kubectl get all -n "$NAMESPACE" -o yaml > "$TEMP_DIR/all-resources.yaml"
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$TEMP_DIR/configmaps.yaml"
    kubectl get secrets -n "$NAMESPACE" -o yaml > "$TEMP_DIR/secrets.yaml"
    kubectl get pvc -n "$NAMESPACE" -o yaml > "$TEMP_DIR/pvcs.yaml"

    # Get pod for data backup
    POD=$(kubectl get pods -n "$NAMESPACE" -l app=paw-node -o jsonpath='{.items[0].metadata.name}')

    if [ -n "$POD" ]; then
        log_info "Backing up data from pod: $POD"

        # Create data backup
        kubectl exec -n "$NAMESPACE" "$POD" -- tar czf /tmp/data-backup.tar.gz -C /paw/.paw data config

        # Copy backup from pod
        kubectl cp "$NAMESPACE/$POD:/tmp/data-backup.tar.gz" "$TEMP_DIR/data-backup.tar.gz"

        # Clean up pod
        kubectl exec -n "$NAMESPACE" "$POD" -- rm /tmp/data-backup.tar.gz
    else
        log_warn "No running pods found for data backup"
    fi

    # Create final backup archive
    tar czf "$BACKUP_FILE" -C "$TEMP_DIR" .

    log_info "Backup created: $BACKUP_FILE"
    log_info "Size: $(du -h "$BACKUP_FILE" | cut -f1)"

    # Create metadata
    cat > "$BACKUP_DIR/paw-k8s-backup-$TIMESTAMP.metadata" << EOF
Backup Type: Kubernetes Deployment
Timestamp: $(date)
Namespace: $NAMESPACE
Cluster: $(kubectl config current-context)
Pod: $POD
Backup File: $BACKUP_FILE
Size: $(du -h "$BACKUP_FILE" | cut -f1)
EOF

    log_info "Metadata saved"
}

cleanup_old_backups() {
    log_step "Cleaning up old backups..."

    # Keep last 7 backups
    KEEP_COUNT=7

    cd "$BACKUP_DIR"
    BACKUP_COUNT=$(ls -1 paw-*-backup-*.tar.gz 2>/dev/null | wc -l)

    if [ "$BACKUP_COUNT" -gt "$KEEP_COUNT" ]; then
        log_info "Found $BACKUP_COUNT backups, keeping last $KEEP_COUNT"

        ls -1t paw-*-backup-*.tar.gz | tail -n +$((KEEP_COUNT + 1)) | while read -r file; do
            log_info "Removing old backup: $file"
            rm -f "$file"
            rm -f "${file%.tar.gz}.metadata"
        done
    else
        log_info "Found $BACKUP_COUNT backups (keeping all)"
    fi
}

verify_backup() {
    BACKUP_FILE="$1"

    log_step "Verifying backup integrity..."

    if tar tzf "$BACKUP_FILE" > /dev/null 2>&1; then
        log_info "Backup integrity verified: OK"
        return 0
    else
        log_error "Backup integrity check FAILED"
        return 1
    fi
}

display_summary() {
    echo ""
    echo "========================================"
    echo "Backup Summary"
    echo "========================================"
    echo "Backup Directory: $BACKUP_DIR"
    echo ""
    echo "Recent Backups:"
    ls -lh "$BACKUP_DIR"/paw-*-backup-*.tar.gz 2>/dev/null | tail -5 || echo "No backups found"
    echo ""
    echo "To restore from backup:"
    echo "  ./restore.sh $DEPLOYMENT_TYPE <backup-file>"
    echo "========================================"
}

# Main execution
main() {
    echo "========================================"
    echo "PAW Blockchain - Backup"
    echo "========================================"
    echo ""

    create_backup_dir

    case "$DEPLOYMENT_TYPE" in
        local)
            backup_local
            ;;
        docker)
            backup_docker
            ;;
        k8s|kubernetes)
            backup_kubernetes
            ;;
        *)
            log_error "Unknown deployment type: $DEPLOYMENT_TYPE"
            echo "Usage: $0 [local|docker|k8s]"
            exit 1
            ;;
    esac

    verify_backup "$BACKUP_FILE"
    cleanup_old_backups
    display_summary

    log_info "Backup completed successfully!"
}

main
