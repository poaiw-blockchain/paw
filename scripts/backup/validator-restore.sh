#!/bin/bash
# ============================================================================
# Validator Key Restore Script
# ============================================================================
# Downloads and decrypts validator keys from Cloudflare R2 backup.
#
# Usage:
#   ./validator-restore.sh [r2-bucket] [backup-filename]
#   ./validator-restore.sh paw-testnet-artifacts validator-keys-paw-20240101-120000.tar.gz.gpg
#
# Environment Variables:
#   BACKUP_PASSPHRASE - GPG passphrase (optional, will prompt if not set)
#   VALIDATOR_HOME    - Override restore destination
#   DRY_RUN          - Set to 1 to preview without restoring
# ============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
R2_BUCKET="${1:-}"
BACKUP_FILE="${2:-}"
DRY_RUN="${DRY_RUN:-0}"
RESTORE_DIR="/tmp/validator-restore-$$"

# ============================================================================
# Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

cleanup() {
    if [ -d "$RESTORE_DIR" ]; then
        # Securely wipe temporary files
        if command -v shred &>/dev/null; then
            find "$RESTORE_DIR" -type f -exec shred -vfz -n 3 {} \; 2>/dev/null || true
        fi
        rm -rf "$RESTORE_DIR"
        log_info "Cleaned up temporary files"
    fi
}

trap cleanup EXIT

usage() {
    echo "Usage: $0 <r2-bucket> <backup-filename>"
    echo ""
    echo "Arguments:"
    echo "  r2-bucket       - R2 bucket name (e.g., paw-testnet-artifacts)"
    echo "  backup-filename - Encrypted backup file name"
    echo ""
    echo "Options:"
    echo "  --list          - List available backups in the bucket"
    echo ""
    echo "Examples:"
    echo "  $0 paw-testnet-artifacts validator-keys-paw-20240101-120000.tar.gz.gpg"
    echo "  $0 --list paw-testnet-artifacts"
    exit 1
}

list_backups() {
    local bucket="$1"
    log_info "Listing backups in ${bucket}..."
    echo ""

    wrangler r2 object list "${bucket}" --prefix "backups/validator-keys/" 2>/dev/null | \
        grep -E "\.gpg$" | \
        sort -r | \
        head -20

    echo ""
    echo "Showing last 20 backups. Use wrangler directly for full list."
}

get_passphrase() {
    if [ -n "${BACKUP_PASSPHRASE:-}" ]; then
        log_info "Using passphrase from BACKUP_PASSPHRASE environment variable"
        echo "$BACKUP_PASSPHRASE"
        return
    fi

    echo -e "${YELLOW}Enter backup decryption passphrase:${NC}" >&2
    read -rs passphrase
    echo >&2

    echo "$passphrase"
}

download_backup() {
    local bucket="$1"
    local filename="$2"

    mkdir -p "$RESTORE_DIR"
    chmod 700 "$RESTORE_DIR"

    local r2_path="backups/validator-keys/${filename}"
    local local_file="$RESTORE_DIR/$filename"

    log_info "Downloading from R2: ${bucket}/${r2_path}"

    wrangler r2 object get "${bucket}/${r2_path}" \
        --file "$local_file" \
        --remote 2>&1 | grep -v "wrangler"

    if [ ! -f "$local_file" ]; then
        log_error "Download failed or file not found"
        exit 1
    fi

    log_success "Downloaded: $filename"
    echo "$local_file"
}

decrypt_backup() {
    local encrypted_file="$1"
    local passphrase="$2"

    log_info "Decrypting backup..."

    local decrypted="${encrypted_file%.gpg}"

    echo "$passphrase" | gpg --batch --yes --passphrase-fd 0 \
        --decrypt \
        --output "$decrypted" \
        "$encrypted_file" 2>/dev/null

    if [ ! -f "$decrypted" ]; then
        log_error "Decryption failed - check passphrase"
        exit 1
    fi

    log_success "Backup decrypted"
    echo "$decrypted"
}

extract_backup() {
    local tarball="$1"

    log_info "Extracting backup..."

    local extract_dir="$RESTORE_DIR/extracted"
    mkdir -p "$extract_dir"

    tar -xzf "$tarball" -C "$extract_dir"

    # Find the backup directory
    local backup_dir=$(find "$extract_dir" -maxdepth 1 -type d -name "validator-keys-*" | head -1)

    if [ -z "$backup_dir" ]; then
        log_error "Invalid backup archive structure"
        exit 1
    fi

    log_success "Extracted backup"
    echo "$backup_dir"
}

verify_checksums() {
    local backup_dir="$1"

    log_info "Verifying checksums..."

    if [ ! -f "$backup_dir/checksums.sha256" ]; then
        log_warn "No checksum file found - skipping verification"
        return 0
    fi

    (cd "$backup_dir" && sha256sum -c checksums.sha256 --quiet)

    if [ $? -ne 0 ]; then
        log_error "Checksum verification failed - backup may be corrupted"
        exit 1
    fi

    log_success "All checksums verified"
}

show_backup_contents() {
    local backup_dir="$1"

    echo ""
    echo -e "${BLUE}Backup Contents:${NC}"
    echo "----------------------------------------"

    # Show metadata
    if [ -f "$backup_dir/BACKUP_METADATA.json" ]; then
        cat "$backup_dir/BACKUP_METADATA.json"
        echo ""
    fi

    # List files
    echo "Files:"
    find "$backup_dir" -type f -name "*.json" -o -name "*.toml" | \
        sed "s|$backup_dir/||g" | \
        sort

    echo "----------------------------------------"
}

determine_chain() {
    local backup_dir="$1"

    if [ -f "$backup_dir/BACKUP_METADATA.json" ]; then
        local chain=$(grep -o '"chain": *"[^"]*"' "$backup_dir/BACKUP_METADATA.json" | \
            cut -d'"' -f4)
        if [ -n "$chain" ]; then
            echo "$chain"
            return
        fi
    fi

    # Fallback: extract from directory name
    local dirname=$(basename "$backup_dir")
    if [[ "$dirname" =~ validator-keys-([a-z]+)- ]]; then
        echo "${BASH_REMATCH[1]}"
        return
    fi

    echo "unknown"
}

restore_files() {
    local backup_dir="$1"
    local validator_home="$2"

    log_info "Restoring files to: $validator_home"

    # Create backup of existing files
    local existing_backup="$validator_home/pre-restore-backup-$(date +%Y%m%d-%H%M%S)"

    if [ -f "$validator_home/config/priv_validator_key.json" ]; then
        log_warn "Existing validator key found - creating backup"
        mkdir -p "$existing_backup/config"
        cp "$validator_home/config/priv_validator_key.json" "$existing_backup/config/" 2>/dev/null || true
        cp "$validator_home/config/node_key.json" "$existing_backup/config/" 2>/dev/null || true
        log_info "Existing keys backed up to: $existing_backup"
    fi

    # Ensure config directory exists
    mkdir -p "$validator_home/config"

    # Restore critical files
    if [ -f "$backup_dir/config/priv_validator_key.json" ]; then
        cp "$backup_dir/config/priv_validator_key.json" "$validator_home/config/"
        chmod 600 "$validator_home/config/priv_validator_key.json"
        log_success "Restored: priv_validator_key.json"
    fi

    if [ -f "$backup_dir/config/node_key.json" ]; then
        cp "$backup_dir/config/node_key.json" "$validator_home/config/"
        chmod 600 "$validator_home/config/node_key.json"
        log_success "Restored: node_key.json"
    fi

    # Optionally restore config files (prompt user)
    echo ""
    echo -e "${YELLOW}Do you want to restore config files (config.toml, app.toml)?${NC}"
    echo "This will overwrite existing configuration."
    read -p "Restore configs? [y/N] " -n 1 -r
    echo ""

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        for config in config.toml app.toml client.toml; do
            if [ -f "$backup_dir/config/$config" ]; then
                cp "$backup_dir/config/$config" "$validator_home/config/"
                log_success "Restored: $config"
            fi
        done
    fi

    # Restore keyring if present
    if [ -d "$backup_dir/keyring/keyring-test" ]; then
        echo ""
        read -p "Restore keyring-test? [y/N] " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            cp -r "$backup_dir/keyring/keyring-test" "$validator_home/"
            log_success "Restored: keyring-test/"
        fi
    fi

    if [ -d "$backup_dir/keyring/keyring-file" ]; then
        echo ""
        read -p "Restore keyring-file? [y/N] " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            cp -r "$backup_dir/keyring/keyring-file" "$validator_home/"
            log_success "Restored: keyring-file/"
        fi
    fi
}

print_summary() {
    local validator_home="$1"
    local chain="$2"

    echo ""
    echo "============================================================================"
    echo -e "${GREEN}RESTORE COMPLETE${NC}"
    echo "============================================================================"
    echo ""
    echo -e "${BLUE}Restored to:${NC} $validator_home"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo "1. Verify the restored validator key address:"
    echo "   ${chain}d tendermint show-validator --home $validator_home"
    echo ""
    echo "2. Start the node:"
    echo "   ${chain}d start --home $validator_home"
    echo ""
    echo "3. Check validator status after sync:"
    echo "   ${chain}d query staking validator \$(${chain}d keys show validator --bech val -a)"
    echo ""
    echo "============================================================================"
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Handle --list option
    if [ "${1:-}" == "--list" ]; then
        if [ -z "${2:-}" ]; then
            log_error "Bucket name required for --list"
            usage
        fi
        list_backups "$2"
        exit 0
    fi

    # Validate arguments
    if [ -z "$R2_BUCKET" ] || [ -z "$BACKUP_FILE" ]; then
        usage
    fi

    echo "============================================================================"
    echo -e "${BLUE}Validator Key Restore${NC}"
    echo "============================================================================"
    echo ""

    local passphrase
    passphrase=$(get_passphrase)

    # Verify passphrase fingerprint
    local passphrase_hash=$(echo -n "$passphrase" | sha256sum | cut -c1-16)
    echo -e "${BLUE}Passphrase fingerprint:${NC} $passphrase_hash"
    echo "Verify this matches the fingerprint from backup."
    echo ""
    read -p "Continue with restore? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled"
        exit 0
    fi

    local encrypted_file
    encrypted_file=$(download_backup "$R2_BUCKET" "$BACKUP_FILE")

    local tarball
    tarball=$(decrypt_backup "$encrypted_file" "$passphrase")

    local backup_dir
    backup_dir=$(extract_backup "$tarball")

    verify_checksums "$backup_dir"
    show_backup_contents "$backup_dir"

    local chain
    chain=$(determine_chain "$backup_dir")
    log_info "Detected chain: $chain"

    local validator_home="${VALIDATOR_HOME:-$HOME/.${chain}}"

    if [ "$DRY_RUN" == "1" ]; then
        log_info "DRY RUN - no files will be restored"
        log_info "Would restore to: $validator_home"
        exit 0
    fi

    echo ""
    echo -e "${YELLOW}This will restore validator keys to: $validator_home${NC}"
    read -p "Proceed with restore? [y/N] " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled"
        exit 0
    fi

    restore_files "$backup_dir" "$validator_home"
    print_summary "$validator_home" "$chain"
}

main "$@"
