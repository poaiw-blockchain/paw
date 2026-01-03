#!/bin/bash
# ============================================================================
# Automated Daily Backup Script (for cron)
# ============================================================================
# This script runs daily backups unattended using a passphrase file.
#
# Setup:
# 1. Create passphrase file: ./setup-backup-passphrase.sh
# 2. Install cron job: ./validator-backup-cron.sh --install
#
# Cron runs daily at 3:00 AM, keeping 30 days of backups.
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$HOME/.validator-backups/logs"
PASSPHRASE_FILE="$HOME/.validator-backups/.backup-passphrase"
RETENTION_DAYS=30

# Default chains to backup
CHAINS="${BACKUP_CHAINS:-aura paw}"

log() {
    local level="$1"
    shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*"
}

install_cron() {
    local cron_entry="0 3 * * * $SCRIPT_DIR/validator-backup-cron.sh >> $LOG_DIR/backup.log 2>&1"

    # Check if already installed
    if crontab -l 2>/dev/null | grep -q "validator-backup-cron.sh"; then
        log "WARN" "Cron job already installed"
        crontab -l | grep "validator-backup-cron"
        return
    fi

    # Add to crontab
    (crontab -l 2>/dev/null; echo "$cron_entry") | crontab -

    log "INFO" "Installed cron job:"
    echo "  $cron_entry"
    echo ""
    echo "Backups will run daily at 3:00 AM"
    echo "Logs: $LOG_DIR/backup.log"
}

uninstall_cron() {
    crontab -l 2>/dev/null | grep -v "validator-backup-cron.sh" | crontab -
    log "INFO" "Removed cron job"
}

check_passphrase() {
    if [ ! -f "$PASSPHRASE_FILE" ]; then
        log "ERROR" "Passphrase file not found: $PASSPHRASE_FILE"
        log "ERROR" "Run: $SCRIPT_DIR/setup-backup-passphrase.sh"
        exit 1
    fi

    # Verify permissions
    local perms=$(stat -c %a "$PASSPHRASE_FILE" 2>/dev/null || stat -f %Lp "$PASSPHRASE_FILE")
    if [ "$perms" != "600" ]; then
        log "ERROR" "Passphrase file has insecure permissions: $perms (should be 600)"
        exit 1
    fi
}

cleanup_old_backups() {
    local bucket="$1"

    log "INFO" "Cleaning up backups older than $RETENTION_DAYS days in $bucket"

    # List and delete old backups
    local cutoff_date=$(date -d "-${RETENTION_DAYS} days" +%Y%m%d 2>/dev/null || \
                        date -v-${RETENTION_DAYS}d +%Y%m%d)

    wrangler r2 object list "$bucket" --prefix "backups/validator-keys/" 2>/dev/null | \
        grep -E "validator-keys-[a-z]+-[0-9]{8}" | \
        while read -r line; do
            local filename=$(echo "$line" | grep -oE "validator-keys-[a-z]+-[0-9]{8}-[0-9]{6}\.tar\.gz\.gpg")
            if [ -n "$filename" ]; then
                local backup_date=$(echo "$filename" | grep -oE "[0-9]{8}" | head -1)
                if [ "$backup_date" -lt "$cutoff_date" ]; then
                    log "INFO" "Deleting old backup: $filename"
                    wrangler r2 object delete "$bucket/backups/validator-keys/$filename" --remote 2>/dev/null || true
                fi
            fi
        done

    # Cleanup local backups
    find "$HOME/.validator-backups" -name "*.gpg" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
}

run_backup() {
    local chain="$1"
    local bucket="${chain}-testnet-artifacts"
    local validator_home="$HOME/.${chain}"

    # Skip if validator home doesn't exist
    if [ ! -d "$validator_home" ]; then
        log "WARN" "Skipping $chain - validator home not found: $validator_home"
        return 0
    fi

    # Skip if no validator key
    if [ ! -f "$validator_home/config/priv_validator_key.json" ]; then
        log "WARN" "Skipping $chain - no validator key found"
        return 0
    fi

    log "INFO" "Starting backup for $chain"

    export BACKUP_PASSPHRASE=$(cat "$PASSPHRASE_FILE")

    if "$SCRIPT_DIR/validator-backup.sh" "$chain" "$bucket"; then
        log "INFO" "Backup completed for $chain"
        cleanup_old_backups "$bucket"
    else
        log "ERROR" "Backup failed for $chain"
        return 1
    fi

    unset BACKUP_PASSPHRASE
}

send_notification() {
    local status="$1"
    local message="$2"

    # Placeholder for notification integration
    # Options: email, Slack, Discord, PagerDuty, etc.

    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -s -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"Validator Backup $status: $message\"}" \
            "$SLACK_WEBHOOK_URL" >/dev/null 2>&1 || true
    fi

    # Log notification
    log "INFO" "Notification: $status - $message"
}

main() {
    # Handle options
    case "${1:-}" in
        --install)
            install_cron
            exit 0
            ;;
        --uninstall)
            uninstall_cron
            exit 0
            ;;
        --help)
            echo "Usage: $0 [--install|--uninstall|--help]"
            echo ""
            echo "Options:"
            echo "  --install   - Install daily cron job"
            echo "  --uninstall - Remove cron job"
            echo ""
            echo "Environment Variables:"
            echo "  BACKUP_CHAINS     - Space-separated chains (default: aura paw)"
            echo "  SLACK_WEBHOOK_URL - Slack webhook for notifications"
            exit 0
            ;;
    esac

    # Setup logging
    mkdir -p "$LOG_DIR"

    log "INFO" "============================================"
    log "INFO" "Starting automated backup run"
    log "INFO" "Chains: $CHAINS"
    log "INFO" "============================================"

    check_passphrase

    local failed=0
    local success=0

    for chain in $CHAINS; do
        if run_backup "$chain"; then
            ((success++))
        else
            ((failed++))
        fi
    done

    log "INFO" "============================================"
    log "INFO" "Backup run complete: $success successful, $failed failed"
    log "INFO" "============================================"

    if [ $failed -gt 0 ]; then
        send_notification "FAILED" "$failed backup(s) failed"
        exit 1
    else
        send_notification "SUCCESS" "All $success backup(s) completed"
    fi
}

main "$@"
