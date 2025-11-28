# TLS/SSL Certificate Management Scripts

This directory contains scripts for managing TLS/SSL certificates for PAW Chain services.

## Scripts Overview

| Script | Purpose | Usage |
|--------|---------|-------|
| `generate-dev-certs.sh` | Generate self-signed certificates for development | Development only |
| `setup-letsencrypt.sh` | Setup Let's Encrypt certificates | Production |
| `setup-cert-renewal.sh` | Configure automatic certificate renewal | Production |
| `renew-cert.sh` | Manually renew certificates | Production |
| `check-cert-expiry.sh` | Check certificate expiration | Monitoring |
| `validate-cert.sh` | Comprehensive certificate validation | Troubleshooting |

## Quick Start

### Development Setup

Generate self-signed certificates for local development:

```bash
./generate-dev-certs.sh localhost
```

Output:
```
certs/
├── privkey.pem      # Private key
├── fullchain.pem    # Certificate
├── chain.pem        # Certificate chain
└── dhparam.pem      # DH parameters
```

### Production Setup

1. **Initial Certificate Request:**
   ```bash
   sudo ./setup-letsencrypt.sh \
     --domain explorer.pawchain.network \
     --email admin@pawchain.network
   ```

2. **Setup Auto-Renewal:**
   ```bash
   sudo ./setup-cert-renewal.sh
   ```

3. **Verify Installation:**
   ```bash
   ./validate-cert.sh
   ```

## Detailed Documentation

### generate-dev-certs.sh

Generate self-signed TLS certificates for development and testing.

**Usage:**
```bash
./generate-dev-certs.sh [domain]
```

**Options:**
- `domain` - Domain name (default: localhost)

**Environment Variables:**
- `CERTS_DIR` - Output directory (default: ./certs)

**Examples:**
```bash
# Generate for localhost
./generate-dev-certs.sh

# Generate for custom domain
./generate-dev-certs.sh dev.pawchain.local

# Custom output directory
CERTS_DIR=/tmp/certs ./generate-dev-certs.sh
```

**Features:**
- 2048-bit RSA key
- SHA-256 signature
- 365-day validity
- Subject Alternative Names (SAN)
- Wildcard support
- DH parameters generation

**Output Files:**
- `privkey.pem` - Private key (600 permissions)
- `fullchain.pem` - Certificate (644 permissions)
- `chain.pem` - Certificate chain
- `dhparam.pem` - DH parameters
- `openssl.cnf` - OpenSSL configuration

### setup-letsencrypt.sh

Setup Let's Encrypt SSL/TLS certificates for production use.

**Usage:**
```bash
sudo ./setup-letsencrypt.sh [OPTIONS]
```

**Options:**
```
-d, --domain DOMAIN       Domain name (required)
-e, --email EMAIL         Email for notifications (required)
-n, --cert-name NAME      Certificate name (optional)
-w, --webroot PATH        Webroot path (default: /var/www/certbot)
-s, --staging             Use staging server (testing)
--dry-run                 Test without making changes
-h, --help                Show help message
```

**Examples:**
```bash
# Basic setup
sudo ./setup-letsencrypt.sh \
  -d explorer.pawchain.network \
  -e admin@pawchain.network

# With custom webroot
sudo ./setup-letsencrypt.sh \
  -d api.pawchain.network \
  -e admin@pawchain.network \
  -w /var/www/html

# Test with staging
sudo ./setup-letsencrypt.sh \
  -d test.pawchain.network \
  -e admin@pawchain.network \
  --staging

# Dry run (test only)
sudo ./setup-letsencrypt.sh \
  -d explorer.pawchain.network \
  -e admin@pawchain.network \
  --dry-run
```

**Requirements:**
- DNS records pointing to server
- Ports 80 and 443 accessible
- certbot installed
- sudo privileges

**What it does:**
1. Creates webroot directory
2. Requests certificate from Let's Encrypt
3. Generates DH parameters
4. Sets proper permissions
5. Provides next steps

### setup-cert-renewal.sh

Configure automatic certificate renewal using systemd or cron.

**Usage:**
```bash
sudo ./setup-cert-renewal.sh [--method systemd|cron]
```

**Options:**
- `--method` - Renewal method (default: auto-detect)
  - `systemd` - Use systemd timer (preferred)
  - `cron` - Use cron job (fallback)

**Examples:**
```bash
# Auto-detect method
sudo ./setup-cert-renewal.sh

# Force systemd
sudo ./setup-cert-renewal.sh --method systemd

# Force cron
sudo ./setup-cert-renewal.sh --method cron
```

**What it creates:**

**Systemd (preferred):**
- `/etc/systemd/system/certbot-renewal.service`
- `/etc/systemd/system/certbot-renewal.timer`
- Runs daily at random time
- Reloads NGINX on success

**Cron (fallback):**
- Cron job: `0 3 * * *` (3:00 AM daily)
- Executes: `certbot renew --quiet`
- Reloads NGINX on success

**Additional files:**
- `/etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh`
- `/usr/local/bin/check-cert-expiry.sh`

**Verification:**
```bash
# Check systemd timer
systemctl status certbot-renewal.timer
systemctl list-timers

# Check cron job
crontab -l | grep certbot

# Test renewal
sudo certbot renew --dry-run
```

### renew-cert.sh

Manually renew Let's Encrypt certificates.

**Usage:**
```bash
sudo ./renew-cert.sh [OPTIONS]
```

**Options:**
```
--force              Force renewal even if not due
--cert-name NAME     Renew specific certificate
-h, --help           Show help message
```

**Examples:**
```bash
# Renew all certificates (if due)
sudo ./renew-cert.sh

# Force renewal
sudo ./renew-cert.sh --force

# Renew specific certificate
sudo ./renew-cert.sh --cert-name explorer.pawchain.network

# Force specific certificate
sudo ./renew-cert.sh --force --cert-name api.pawchain.network
```

**What it does:**
1. Runs certbot renew
2. Validates renewal
3. Reloads NGINX
4. Shows certificate info

**When to use:**
- Certificate about to expire
- Testing renewal process
- Recovering from failed auto-renewal
- After configuration changes

### check-cert-expiry.sh

Check SSL certificate expiration date and warn if close to expiry.

**Usage:**
```bash
./check-cert-expiry.sh [cert_path] [warning_days]
```

**Options:**
- `cert_path` - Path to certificate (default: Let's Encrypt live cert)
- `warning_days` - Warning threshold in days (default: 30)

**Examples:**
```bash
# Check default certificate
./check-cert-expiry.sh

# Check specific certificate
./check-cert-expiry.sh /etc/letsencrypt/live/api.pawchain.network/fullchain.pem

# Custom warning threshold (7 days)
./check-cert-expiry.sh /path/to/cert.pem 7
```

**Exit codes:**
- `0` - Certificate valid (> warning days)
- `1` - Warning (< warning days)
- `2` - Critical (< 7 days or expired)

**Output:**
```
Certificate: /etc/letsencrypt/live/explorer.pawchain.network/fullchain.pem
Expiry Date: Jan 15 12:00:00 2024 GMT
Days Until Expiry: 45

Certificate Details:
subject=CN = explorer.pawchain.network
issuer=C = US, O = Let's Encrypt, CN = R3
notBefore=Oct 17 12:00:00 2023 GMT
notAfter=Jan 15 12:00:00 2024 GMT

[OK] Certificate is valid for 45 more days.
```

**Monitoring integration:**
```bash
# Add to monitoring script
if ! ./check-cert-expiry.sh; then
  send_alert "Certificate expiring soon"
fi

# Use in cron for alerts
0 6 * * * /path/to/check-cert-expiry.sh || mail -s "Cert Alert" admin@github.com
```

### validate-cert.sh

Comprehensive certificate validation and security checks.

**Usage:**
```bash
./validate-cert.sh [cert_dir]
```

**Options:**
- `cert_dir` - Certificate directory (default: Let's Encrypt live dir)

**Examples:**
```bash
# Validate default certificate
./validate-cert.sh

# Validate specific certificate
./validate-cert.sh /etc/letsencrypt/live/api.pawchain.network

# Validate development certificate
./validate-cert.sh ./certs
```

**Checks performed:**

1. **File Existence**
   - Certificate file
   - Private key file
   - Chain file

2. **File Permissions**
   - Certificate: 644 or 444
   - Private key: 600 or 400

3. **Certificate Syntax**
   - Valid X.509 format
   - Parseable by OpenSSL

4. **Key Matching**
   - Certificate and key pair validation
   - Modulus comparison

5. **Expiry**
   - Current validity
   - Days until expiration
   - Expiry warnings

6. **Key Strength**
   - Algorithm (RSA, ECDSA)
   - Key size (>= 2048 bits)

7. **Signature**
   - Algorithm (SHA-256, SHA-384, etc.)
   - Security level

8. **Subject Alternative Names**
   - SAN existence
   - Domain coverage

**Exit codes:**
- `0` - All checks passed
- `1` - Warnings found
- `2` - Errors found

**Sample output:**
```
PAW Chain - Certificate Validation
====================================

Checking file existence...
[PASS] Certificate file exists
[PASS] Private key file exists
[PASS] Chain file exists

Checking file permissions...
[PASS] Certificate permissions are correct: 644
[PASS] Private key permissions are secure: 600

Validating certificate syntax...
[PASS] Certificate syntax is valid
[PASS] Private key syntax is valid

Checking certificate and key match...
[PASS] Certificate and private key match

Checking certificate expiry...
[PASS] Certificate valid for 45 days

Checking key algorithm and strength...
  Algorithm: rsaEncryption
  Key Size: 2048 bits
[PASS] Key size is adequate (2048 bits)

Checking signature algorithm...
  Signature Algorithm: sha256WithRSAEncryption
[PASS] Signature algorithm is secure

Certificate Information:
subject=CN = explorer.pawchain.network
issuer=C = US, O = Let's Encrypt, CN = R3

Checking Subject Alternative Names...
[PASS] SAN found: DNS:explorer.pawchain.network

==================================
Validation Summary:
  Errors: 0
  Warnings: 0

[SUCCESS] Certificate validation passed with no issues!
```

## Common Workflows

### Initial Production Setup

```bash
# 1. Generate production certificates
sudo ./setup-letsencrypt.sh \
  -d explorer.pawchain.network \
  -e admin@pawchain.network

# 2. Validate installation
./validate-cert.sh

# 3. Setup auto-renewal
sudo ./setup-cert-renewal.sh

# 4. Test renewal
sudo certbot renew --dry-run
```

### Development Workflow

```bash
# 1. Generate dev certificates
./generate-dev-certs.sh localhost

# 2. Copy to NGINX directory
sudo mkdir -p /etc/nginx/ssl
sudo cp certs/* /etc/nginx/ssl/

# 3. Update NGINX config
sudo nano /etc/nginx/sites-available/default

# 4. Test and reload
sudo nginx -t && sudo systemctl reload nginx
```

### Certificate Renewal

```bash
# 1. Check expiry
./check-cert-expiry.sh

# 2. Renew if needed
sudo ./renew-cert.sh

# 3. Validate renewal
./validate-cert.sh

# 4. Verify site
curl -I https://explorer.pawchain.network
```

### Troubleshooting

```bash
# 1. Validate certificate
./validate-cert.sh

# 2. Check expiry
./check-cert-expiry.sh

# 3. Test renewal
sudo certbot renew --dry-run

# 4. Check logs
sudo journalctl -u certbot-renewal
sudo tail -f /var/log/letsencrypt/letsencrypt.log

# 5. Force renewal if needed
sudo ./renew-cert.sh --force
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CERTS_DIR` | Certificate output directory | `./certs` |
| `CERTBOT_EMAIL` | Email for Let's Encrypt | - |
| `DOMAIN` | Domain name | - |

## File Permissions

Proper file permissions are critical for security:

```bash
# Certificate files (public)
chmod 644 fullchain.pem chain.pem cert.pem

# Private key (secret)
chmod 600 privkey.pem

# Directories
chmod 755 /etc/letsencrypt/live
chmod 700 /etc/letsencrypt/archive
```

## Best Practices

1. **Always validate** certificates after generation
2. **Test renewal** with `--dry-run` before production
3. **Monitor expiry** with automated checks
4. **Keep backups** of certificates
5. **Use strong DH parameters** (2048-bit minimum)
6. **Regular security audits** with validate-cert.sh
7. **Document changes** in production

## Troubleshooting

### Common Issues

**Certificate request failed:**
```bash
# Check DNS
dig explorer.pawchain.network

# Verify webroot
ls -la /var/www/certbot

# Check logs
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

**Renewal failed:**
```bash
# Check certificate status
sudo certbot certificates

# Test renewal
sudo certbot renew --dry-run

# Force renewal
sudo ./renew-cert.sh --force
```

**Permission errors:**
```bash
# Fix certificate permissions
sudo chmod 644 /etc/letsencrypt/live/*/fullchain.pem
sudo chmod 600 /etc/letsencrypt/live/*/privkey.pem
```

## Security Considerations

- Never commit private keys to version control
- Use strong DH parameters (2048-bit or higher)
- Enable HSTS in production
- Implement OCSP stapling
- Regular certificate rotation
- Monitor for certificate expiry
- Use Let's Encrypt rate limits wisely

## Support

For issues or questions:
- Documentation: `/home/decri/blockchain-projects/paw/docs/TLS_SSL_SETUP_GUIDE.md`
-  Issues: https://github.com/paw-chain/paw/issues
- Email: support@pawchain.network

## License

MIT License - See LICENSE file for details
