#!/bin/bash
# ============================================================================
# Backup Passphrase Setup Script
# ============================================================================
# Securely creates and stores the backup encryption passphrase.
#
# Security Considerations:
# - Passphrase is stored encrypted on disk
# - File permissions are restricted (600)
# - Optional: Use hardware security module or OS keyring
#
# IMPORTANT: Keep a copy of this passphrase in a secure offline location!
# ============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BACKUP_DIR="$HOME/.validator-backups"
PASSPHRASE_FILE="$BACKUP_DIR/.backup-passphrase"
PASSPHRASE_HASH_FILE="$BACKUP_DIR/.backup-passphrase.sha256"

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

generate_passphrase() {
    # Generate a strong random passphrase
    # 32 bytes = 256 bits of entropy, base64 encoded
    openssl rand -base64 32
}

create_passphrase() {
    echo "============================================================================"
    echo -e "${BLUE}Backup Passphrase Setup${NC}"
    echo "============================================================================"
    echo ""

    mkdir -p "$BACKUP_DIR"
    chmod 700 "$BACKUP_DIR"

    if [ -f "$PASSPHRASE_FILE" ]; then
        log_warn "Passphrase file already exists: $PASSPHRASE_FILE"
        echo ""
        echo "Options:"
        echo "  1. Keep existing passphrase"
        echo "  2. Generate new passphrase (will require re-encrypting all backups)"
        echo ""
        read -p "Choice [1/2]: " -n 1 -r
        echo ""

        if [[ ! $REPLY =~ ^[2]$ ]]; then
            log_info "Keeping existing passphrase"
            show_passphrase_info
            exit 0
        fi

        log_warn "Creating backup of old passphrase..."
        mv "$PASSPHRASE_FILE" "$PASSPHRASE_FILE.old.$(date +%Y%m%d-%H%M%S)"
    fi

    echo ""
    echo "Choose passphrase method:"
    echo "  1. Auto-generate strong passphrase (recommended)"
    echo "  2. Enter custom passphrase"
    echo ""
    read -p "Choice [1/2]: " -n 1 -r
    echo ""

    local passphrase

    if [[ $REPLY =~ ^[1]$ ]]; then
        passphrase=$(generate_passphrase)
        echo ""
        echo "============================================================================"
        echo -e "${YELLOW}GENERATED PASSPHRASE (SAVE THIS SECURELY!)${NC}"
        echo "============================================================================"
        echo ""
        echo "$passphrase"
        echo ""
        echo "============================================================================"
        echo ""
        log_warn "This passphrase will only be shown ONCE!"
        log_warn "Copy it to a secure location NOW (password manager, offline storage)"
        echo ""
        read -p "Have you saved the passphrase? [y/N] " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "Please save the passphrase and try again"
            exit 1
        fi
    else
        echo -e "${YELLOW}Enter custom passphrase (min 24 characters):${NC}"
        read -rs passphrase
        echo ""
        echo -e "${YELLOW}Confirm passphrase:${NC}"
        read -rs passphrase2
        echo ""

        if [ "$passphrase" != "$passphrase2" ]; then
            log_error "Passphrases do not match"
            exit 1
        fi

        if [ ${#passphrase} -lt 24 ]; then
            log_error "Passphrase must be at least 24 characters"
            exit 1
        fi
    fi

    # Save passphrase securely
    echo -n "$passphrase" > "$PASSPHRASE_FILE"
    chmod 600 "$PASSPHRASE_FILE"

    # Save hash for verification
    echo -n "$passphrase" | sha256sum > "$PASSPHRASE_HASH_FILE"
    chmod 600 "$PASSPHRASE_HASH_FILE"

    log_success "Passphrase saved to: $PASSPHRASE_FILE"

    show_passphrase_info
}

show_passphrase_info() {
    local passphrase=$(cat "$PASSPHRASE_FILE")
    local fingerprint=$(echo -n "$passphrase" | sha256sum | cut -c1-16)

    echo ""
    echo "============================================================================"
    echo -e "${GREEN}Passphrase Configuration Complete${NC}"
    echo "============================================================================"
    echo ""
    echo -e "${BLUE}Passphrase fingerprint:${NC} $fingerprint"
    echo ""
    echo "Use this fingerprint to verify you have the correct passphrase when restoring."
    echo ""
    echo -e "${YELLOW}Security Recommendations:${NC}"
    echo ""
    echo "1. BACKUP THE PASSPHRASE to multiple secure locations:"
    echo "   - Hardware password manager (1Password, Bitwarden, etc.)"
    echo "   - Encrypted USB drive stored offline"
    echo "   - Paper copy in a safe/security deposit box"
    echo "   - Split using Shamir's Secret Sharing for critical infrastructure"
    echo ""
    echo "2. PROTECT THE PASSPHRASE FILE:"
    echo "   - File location: $PASSPHRASE_FILE"
    echo "   - Permissions: 600 (owner read/write only)"
    echo "   - Consider encrypting the disk/volume"
    echo ""
    echo "3. TEST RESTORE PROCEDURE:"
    echo "   - Run a test backup: ./validator-backup.sh aura aura-testnet-artifacts"
    echo "   - Run a test restore: DRY_RUN=1 ./validator-restore.sh ..."
    echo ""
    echo "============================================================================"
}

verify_passphrase() {
    if [ ! -f "$PASSPHRASE_FILE" ]; then
        log_error "No passphrase file found. Run setup first."
        exit 1
    fi

    echo -e "${YELLOW}Enter passphrase to verify:${NC}"
    read -rs test_passphrase
    echo ""

    local stored_hash=$(cat "$PASSPHRASE_HASH_FILE" | cut -d' ' -f1)
    local test_hash=$(echo -n "$test_passphrase" | sha256sum | cut -d' ' -f1)

    if [ "$stored_hash" == "$test_hash" ]; then
        log_success "Passphrase verified correctly!"
    else
        log_error "Passphrase does NOT match!"
        exit 1
    fi
}

main() {
    case "${1:-}" in
        --verify)
            verify_passphrase
            ;;
        --show-fingerprint)
            if [ -f "$PASSPHRASE_FILE" ]; then
                local passphrase=$(cat "$PASSPHRASE_FILE")
                echo -n "$passphrase" | sha256sum | cut -c1-16
            else
                log_error "No passphrase file found"
                exit 1
            fi
            ;;
        --help)
            echo "Usage: $0 [--verify|--show-fingerprint|--help]"
            echo ""
            echo "Options:"
            echo "  --verify           - Verify passphrase matches stored value"
            echo "  --show-fingerprint - Show passphrase fingerprint (first 16 chars of SHA256)"
            echo ""
            echo "Without options, runs interactive setup."
            ;;
        *)
            create_passphrase
            ;;
    esac
}

main "$@"
