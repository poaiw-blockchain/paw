# PAW Systemd Services

This directory contains systemd service files for running PAW blockchain nodes on Linux systems.

## Files

- `pawd.service` - Standard full node service
- `pawd-validator.service` - Validator node service (stricter security)
- `pawd.env` - Environment variable template

## Quick Installation

### 1. Create Service User

```bash
# Create paw user (no login shell)
sudo useradd -r -s /sbin/nologin -m -d /home/paw paw
```

### 2. Install Binary

```bash
# Copy binary
sudo cp pawd /usr/local/bin/
sudo chmod +x /usr/local/bin/pawd

# Verify
pawd version
```

### 3. Initialize Node

```bash
# As paw user
sudo -u paw pawd init <moniker> --chain-id paw-1 --home /home/paw/.paw

# Copy genesis file
sudo -u paw curl -o /home/paw/.paw/config/genesis.json <genesis_url>
```

### 4. Configure Environment

```bash
# Create config directory
sudo mkdir -p /etc/paw

# Copy and edit environment file
sudo cp pawd.env /etc/paw/pawd.env
sudo nano /etc/paw/pawd.env
```

### 5. Install Service

```bash
# For full node
sudo cp pawd.service /etc/systemd/system/

# OR for validator
sudo cp pawd-validator.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload
```

### 6. Start Service

```bash
# Enable on boot
sudo systemctl enable pawd  # or pawd-validator

# Start
sudo systemctl start pawd

# Check status
sudo systemctl status pawd

# View logs
journalctl -u pawd -f
```

## Validator-Specific Setup

For validators, additional steps are required:

1. **Secure Key Management**: Use HSM or encrypted storage for `priv_validator_key.json`
2. **Genesis Verification**: Set `GENESIS_CHECKSUM` in `/etc/paw/validator.env`
3. **Sentry Architecture**: Configure validator behind sentry nodes
4. **Backup**: Regular backup of `priv_validator_state.json`

## Service Management

```bash
# Start/stop/restart
sudo systemctl start pawd
sudo systemctl stop pawd
sudo systemctl restart pawd

# Enable/disable on boot
sudo systemctl enable pawd
sudo systemctl disable pawd

# View status
sudo systemctl status pawd

# View logs
journalctl -u pawd -f
journalctl -u pawd --since "1 hour ago"
journalctl -u pawd -n 100
```

## Upgrade Procedure

1. Stop the service:
   ```bash
   sudo systemctl stop pawd
   ```

2. Backup state:
   ```bash
   sudo -u paw cp /home/paw/.paw/data/priv_validator_state.json /home/paw/backup/
   ```

3. Replace binary:
   ```bash
   sudo cp pawd-new /usr/local/bin/pawd
   ```

4. Restart service:
   ```bash
   sudo systemctl start pawd
   ```

5. Verify:
   ```bash
   pawd version
   sudo systemctl status pawd
   ```

## Troubleshooting

### Service won't start

```bash
# Check logs
journalctl -u pawd -n 50 --no-pager

# Verify binary
/usr/local/bin/pawd version

# Check permissions
ls -la /home/paw/.paw/
```

### Out of memory

Edit resource limits in service file:
```ini
MemoryMax=32G
```

### Too many open files

Increase limits:
```bash
# Check current
cat /proc/$(pgrep pawd)/limits | grep "open files"

# Edit service
LimitNOFILE=131072
```

### Slow sync

Enable state sync in `/etc/paw/pawd.env`:
```bash
STATE_SYNC_RPC=https://rpc.paw.network:443
STATE_SYNC_TRUST_HEIGHT=<recent_block_height>
STATE_SYNC_TRUST_HASH=<block_hash>
```

## Security Recommendations

1. **File Permissions**:
   ```bash
   chmod 600 /home/paw/.paw/config/priv_validator_key.json
   chmod 600 /home/paw/.paw/config/node_key.json
   ```

2. **Firewall**:
   ```bash
   # Allow P2P
   sudo ufw allow 26656/tcp

   # Restrict RPC to localhost
   sudo ufw deny 26657/tcp
   ```

3. **Regular Updates**: Monitor releases and apply security patches promptly

4. **Monitoring**: Use Prometheus metrics endpoint (:26660) for alerting
