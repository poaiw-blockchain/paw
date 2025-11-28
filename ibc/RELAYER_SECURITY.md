# IBC Relayer Security Configuration

## Secure Key Management

The PAW IBC relayer has been configured to use secure key storage instead of test keystores.

### Key Storage Options

1. **OS Keychain (Recommended for Production)**
   - Uses the operating system's native keychain
   - macOS: Keychain Access
   - Windows: Windows Credential Manager
   - Linux: Secret Service API (libsecret)

2. **File-based Storage (Alternative)**
   - Encrypted key files stored on disk
   - Requires proper file permissions (0600)
   - Suitable for Linux servers

### Setup Instructions

#### Option 1: OS Keychain (Current Configuration)

```bash
# Add relayer key to OS keychain
hermes keys add --chain paw-1 --key-name relayer

# For each connected chain
hermes keys add --chain osmosis-1 --key-name relayer
hermes keys add --chain cosmoshub-4 --key-name relayer
hermes keys add --chain celestia --key-name relayer
hermes keys add --chain injective-1 --key-name relayer
```

#### Option 2: File-based Storage

If you need to use file-based storage (e.g., on Linux without keychain support):

1. Update the configuration in `relayer-config.yaml`:
   ```yaml
   key_store_type = 'file'
   ```

2. Create encrypted key files:
   ```bash
   # Create keys directory with secure permissions
   mkdir -p ~/.hermes/keys
   chmod 700 ~/.hermes/keys

   # Add keys
   hermes keys add --chain paw-1 --key-name relayer --key-file /path/to/key.json

   # Set secure permissions on key files
   chmod 600 ~/.hermes/keys/*
   ```

### Key Rotation Procedures

1. **Generate New Key**
   ```bash
   hermes keys add --chain paw-1 --key-name relayer-new
   ```

2. **Fund New Key**
   - Transfer sufficient tokens for gas fees to the new relayer address

3. **Update Configuration**
   - Update `key_name` in the configuration file
   - Restart the relayer service

4. **Delete Old Key** (after verification)
   ```bash
   hermes keys delete --chain paw-1 --key-name relayer
   ```

### Security Best Practices

1. **Never use Test keystore in production**
   - Test keystores store keys in plaintext
   - Keys are easily accessible and not encrypted

2. **Regular Key Rotation**
   - Rotate relayer keys every 90 days
   - Keep rotation logs for audit purposes

3. **Access Control**
   - Limit access to the relayer server
   - Use separate keys for each chain
   - Monitor relayer transactions for anomalies

4. **Backup Strategy**
   - Keep encrypted backups of relayer keys
   - Store backups in secure, offline locations
   - Document key recovery procedures

5. **Monitoring**
   - Monitor relayer balance changes
   - Alert on unexpected gas consumption
   - Track failed relay attempts

### Troubleshooting

#### Key Not Found Error

```bash
# List all keys
hermes keys list --chain paw-1

# Re-add key if missing
hermes keys add --chain paw-1 --key-name relayer
```

#### Permission Denied

```bash
# Fix file permissions
chmod 600 ~/.hermes/keys/*
chmod 700 ~/.hermes/keys
```

### Production Deployment Checklist

- [ ] Update all `key_store_type` from 'Test' to 'os' or 'file'
- [ ] Add keys to secure keystore
- [ ] Verify key permissions (0600 for files)
- [ ] Test relayer connectivity
- [ ] Fund relayer accounts with gas tokens
- [ ] Set up monitoring and alerting
- [ ] Document key backup locations
- [ ] Schedule first key rotation

### Emergency Procedures

If a relayer key is compromised:

1. **Immediately stop the relayer**
   ```bash
   systemctl stop hermes-relayer
   ```

2. **Generate new keys**
   ```bash
   hermes keys add --chain paw-1 --key-name relayer-emergency
   ```

3. **Transfer remaining funds**
   - Move any remaining tokens from compromised address

4. **Update configuration**
   - Point to new keys

5. **Investigate breach**
   - Review logs for unauthorized access
   - Check for unusual transactions

6. **Report incident**
   - Document the incident
   - Notify relevant stakeholders

### Additional Resources

- [Hermes Documentation](https://hermes.informal.systems/)
- [Cosmos IBC Security Best Practices](https://github.com/cosmos/ibc)
- [Key Management Standards](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
