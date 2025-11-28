# PGP Key Setup for Bug Bounty Program

## Overview

This guide explains how to generate and manage PGP keys for encrypted vulnerability reports. Security researchers can use our public key to encrypt sensitive reports, and the PAW security team uses the private key to decrypt them.

## For Security Researchers: Encrypting Reports

### Step 1: Install GPG

#### Linux (Debian/Ubuntu)

```bash
sudo apt-get update
sudo apt-get install gnupg
```

#### macOS

```bash
brew install gnupg
```

#### Windows

Download and install GPG4Win: https://www.gpg4win.org/

### Step 2: Import PAW Security Team Public Key

Download and import our public key:

```bash
# Download the key
curl https://paw-blockchain.org/security.asc -o paw-security.asc

# Import the key
gpg --import paw-security.asc

# Verify the fingerprint
gpg --fingerprint security@paw-blockchain.org
```

**Expected Fingerprint**: `[FINGERPRINT_WILL_BE_INSERTED]`

### Step 3: Verify the Key

Always verify the key fingerprint through multiple channels:

- Check the fingerprint on our website
- Verify on our  repository
- Cross-reference with our social media posts

### Step 4: Encrypt Your Report

Create your vulnerability report as a markdown file, then encrypt it:

```bash
# Encrypt the report
gpg --encrypt --recipient security@paw-blockchain.org \
    --armor --output vulnerability-report.asc \
    vulnerability-report.md

# Or encrypt with output to stdout
gpg --encrypt --recipient security@paw-blockchain.org \
    --armor vulnerability-report.md > vulnerability-report.asc
```

### Step 5: Send Encrypted Report

Email the encrypted file to: security@paw-blockchain.org

**Email Subject**: `[SECURITY] [Encrypted] Brief Description`

**Email Body**:

```
Hi PAW Security Team,

Please find attached an encrypted vulnerability report.

Key Fingerprint Verified: [Yes/No]
Severity: [Critical/High/Medium/Low]
Tracking Info: [Optional - any reference info]

Best regards,
[Your Name/Handle]
```

## For PAW Security Team: Key Management

### Generating the Security Team PGP Key

**IMPORTANT**: This should only be done once during initial setup.

```bash
# Generate a new key pair
gpg --full-generate-key

# Configuration:
# - Key type: RSA and RSA
# - Key size: 4096 bits
# - Expiration: 2 years (with annual renewal reminder)
# - Name: PAW Blockchain Security Team
# - Email: security@paw-blockchain.org
# - Comment: Bug Bounty Program
```

### Key Parameters

```
Key Type: RSA 4096-bit
Name: PAW Blockchain Security Team
Email: security@paw-blockchain.org
Comment: Bug Bounty Program
Expiration: 2 years
Usage: Sign, Encrypt
```

### Export Public Key

```bash
# Export ASCII-armored public key
gpg --armor --export security@paw-blockchain.org > paw-security.asc

# Verify the export
cat paw-security.asc

# Get the fingerprint
gpg --fingerprint security@paw-blockchain.org
```

### Publish Public Key

1. ** Repository**
   - Add to repository root as `SECURITY_PGP_KEY.asc`
   - Reference in SECURITY.md

2. **Website**
   - Publish at https://paw-blockchain.org/security.asc
   - Display fingerprint prominently

3. **Keyservers**

   ```bash
   # Upload to keyservers
   gpg --keyserver keys.openpgp.org --send-keys [KEY_ID]
   gpg --keyserver keyserver.ubuntu.com --send-keys [KEY_ID]
   gpg --keyserver pgp.mit.edu --send-keys [KEY_ID]
   ```

4. **Social Media**
   - Post fingerprint on Twitter/X
   - Pin message in Discord
   - Announcement on blog

### Key Backup and Storage

**CRITICAL**: The private key must be securely backed up and stored.

```bash
# Export private key (SECURE THIS FILE!)
gpg --export-secret-keys --armor security@paw-blockchain.org > \
    paw-security-private.asc

# Create encrypted backup
gpg --symmetric --armor paw-security-private.asc
```

**Storage Requirements**:

1. **Primary Storage**
   - Hardware Security Module (HSM) if available
   - Encrypted disk on secure server
   - Access restricted to security team leads only

2. **Backup Storage**
   - Encrypted offline backup (USB drive in safe)
   - Paper backup of private key (in secure facility)
   - Split key backup across multiple secure locations

3. **Access Control**
   - Minimum 2 people must have access
   - Maximum 4 people should have access
   - Document who has access
   - Annual access review

### Decrypting Reports

```bash
# Decrypt a received report
gpg --decrypt vulnerability-report.asc > vulnerability-report.md

# Or decrypt and view
gpg --decrypt vulnerability-report.asc | less

# Verify signature if signed
gpg --verify signed-report.asc
```

### Key Renewal Process

**Timeline**: 60 days before expiration

1. **Extend Expiration**

   ```bash
   # Edit key
   gpg --edit-key security@paw-blockchain.org

   # At gpg prompt:
   expire
   # Choose new expiration (2 years)
   save
   ```

2. **Re-publish Updated Key**
   - Export updated public key
   - Update all published locations
   - Announce on security channels

3. **Notify Active Researchers**
   - Email researchers with pending reports
   - Announce on bug bounty program page

### Key Revocation Certificate

Generate and securely store a revocation certificate:

```bash
# Generate revocation certificate
gpg --output paw-security-revoke.asc \
    --gen-revoke security@paw-blockchain.org

# Store securely:
# - Encrypted offline storage
# - Multiple backup locations
# - Document storage locations
```

**Use revocation certificate only if**:

- Private key is compromised
- Private key is lost
- Email address is compromised
- Key needs to be replaced

### Multi-Team Access Setup

For teams with multiple members who need decryption access:

```bash
# Option 1: Import same private key (simpler)
# Each team member imports the same private key
# PRO: Simple, everyone can decrypt
# CON: More copies of private key exist

# Option 2: Subkeys for team members (more secure)
gpg --edit-key security@paw-blockchain.org
# At prompt:
addkey
# Add encryption subkey for each team member
```

## Security Best Practices

### For Key Generation

1. **Use Strong Entropy**

   ```bash
   # On Linux, ensure good entropy
   cat /proc/sys/kernel/random/entropy_avail
   # Should be > 1000

   # Generate entropy if needed
   sudo apt-get install rng-tools
   sudo rngd -r /dev/urandom
   ```

2. **Use Strong Passphrase**
   - Minimum 20 characters
   - Use passphrase generator
   - Store in password manager
   - Never share passphrase

3. **Protect Private Key**
   - Never transmit over unencrypted channels
   - Never store in cloud without encryption
   - Never commit to version control
   - Limit access to minimum necessary people

### For Key Usage

1. **Verify Before Encrypting**
   - Always verify key fingerprint
   - Check through multiple channels
   - Don't trust keys from unknown sources

2. **Regular Key Rotation**
   - Review key expiration annually
   - Rotate keys every 2-3 years
   - Plan migration before expiration

3. **Monitor for Compromise**
   - Watch for unauthorized key usage
   - Monitor keyserver changes
   - Review access logs

## Troubleshooting

### Cannot Import Key

```bash
# Check GPG version
gpg --version

# Verify key file format
file paw-security.asc

# Try importing with verbose output
gpg --import --verbose paw-security.asc
```

### Encryption Fails

```bash
# Verify key is imported
gpg --list-keys security@paw-blockchain.org

# Check key trust
gpg --edit-key security@paw-blockchain.org
trust
# Choose appropriate trust level
quit
```

### Decryption Fails

```bash
# Verify you have the private key
gpg --list-secret-keys

# Check passphrase
gpg --export-secret-keys security@paw-blockchain.org > /dev/null
# Will prompt for passphrase

# Verify file is properly encrypted
gpg --list-packets encrypted-file.asc
```

## Alternative: Using Keybase

Security researchers can also use Keybase for encryption:

```bash
# Install Keybase
# https://keybase.io/download

# Encrypt for PAW team (if we have Keybase account)
keybase encrypt pawblockchain -m "vulnerability report content" \
    -o vulnerability-report.txt
```

## PGP Best Practices for Researchers

1. **Generate Your Own Key**
   - Create a personal PGP key
   - Sign your vulnerability reports
   - Helps us verify authenticity

2. **Use Key for Identity**
   - Consistent identity across reports
   - Build reputation
   - Enable encrypted responses

3. **Backup Your Key**
   - Backup your private key securely
   - Store revocation certificate
   - Don't lose access to rewards!

4. **Verify Responses**
   - We will sign responses with our key
   - Verify signatures on emails
   - Beware of impersonation

## Example: Complete Workflow

### Researcher Side

```bash
# 1. Import PAW security key
curl https://paw-blockchain.org/security.asc | gpg --import

# 2. Verify fingerprint
gpg --fingerprint security@paw-blockchain.org

# 3. Create report
cat > vulnerability.md << 'EOF'
# Vulnerability Report
[Your detailed report here]
EOF

# 4. Encrypt report
gpg --encrypt --recipient security@paw-blockchain.org \
    --armor vulnerability.md

# 5. Email vulnerability.md.asc to security@paw-blockchain.org
```

### Security Team Side

```bash
# 1. Receive encrypted report via email
# Save attachment as vulnerability.asc

# 2. Decrypt report
gpg --decrypt vulnerability.asc > vulnerability.md

# 3. Review report
cat vulnerability.md

# 4. Create encrypted response (if researcher provided key)
gpg --encrypt --recipient researcher@domain.com \
    --sign --local-user security@paw-blockchain.org \
    --armor response.md

# 5. Send encrypted response
```

## Contact for Key Issues

If you have issues with PGP encryption:

- **Email**: security@paw-blockchain.org (with unencrypted message explaining issue)
- **Subject**: `[PGP HELP] Brief description of issue`
- **Alternative**: Use  Security Advisory (no encryption required)

---

**Document Version**: 1.0
**Last Updated**: November 14, 2025
**Next Review**: February 14, 2026

**Note**: This is a guide. Actual PGP key will be generated and published separately.
