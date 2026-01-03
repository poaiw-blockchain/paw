#!/bin/bash
# PAW Validator Encrypted Backup Script
set -e

CHAIN="${1:-paw}"
BUCKET="${2:-paw-testnet-artifacts}"
BACKUP_DIR="$HOME/.validator-backups/${CHAIN}"
PASSPHRASE_FILE="$HOME/.validator-backups/.backup-passphrase"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"

if [ ! -f "$PASSPHRASE_FILE" ]; then
    echo "ERROR: Passphrase file not found at $PASSPHRASE_FILE"
    exit 1
fi

echo "=== PAW Validator Backup ==="
echo "Timestamp: $TIMESTAMP"

echo "Fetching validator keys..."
ssh paw-testnet 'tar -czf - -C ~/.paw/config priv_validator_key.json node_key.json genesis.json' > /tmp/paw-keys-${TIMESTAMP}.tar.gz

echo "Encrypting backup..."
gpg --batch --yes --passphrase-file "$PASSPHRASE_FILE" --symmetric --cipher-algo AES256 \
    -o "${BACKUP_DIR}/validator-keys-${CHAIN}-${TIMESTAMP}.tar.gz.gpg" \
    /tmp/paw-keys-${TIMESTAMP}.tar.gz

sha256sum "${BACKUP_DIR}/validator-keys-${CHAIN}-${TIMESTAMP}.tar.gz.gpg" > "${BACKUP_DIR}/validator-keys-${CHAIN}-${TIMESTAMP}.tar.gz.gpg.sha256"

echo "Uploading to R2..."
source ~/.nvm/nvm.sh && nvm use 20 > /dev/null 2>&1
wrangler r2 object put "${BUCKET}/backups/validator-keys-${CHAIN}-${TIMESTAMP}.tar.gz.gpg" \
    --file "${BACKUP_DIR}/validator-keys-${CHAIN}-${TIMESTAMP}.tar.gz.gpg" --remote

rm -f /tmp/paw-keys-${TIMESTAMP}.tar.gz

echo "Backup complete: validator-keys-${CHAIN}-${TIMESTAMP}.tar.gz.gpg"
