# Production TLS Certificates for PAW Blockchain

This guide explains how to obtain and configure production-grade TLS certificates from trusted Certificate Authorities (CAs) for the PAW blockchain API server.

**⚠️ WARNING: NEVER use self-signed certificates in production!**

## Table of Contents

1. [Why Production Certificates Matter](#why-production-certificates-matter)
2. [Let's Encrypt (Recommended)](#lets-encrypt-recommended)
3. [Commercial Certificate Authorities](#commercial-certificate-authorities)
4. [Certificate Installation](#certificate-installation)
5. [Certificate Renewal](#certificate-renewal)
6. [Security Best Practices](#security-best-practices)
7. [Troubleshooting](#troubleshooting)

---

## Why Production Certificates Matter

Production certificates from trusted CAs provide:

- ✅ **Browser Trust**: Certificates are automatically trusted by all major browsers and clients
- ✅ **No Security Warnings**: Users won't see scary certificate warnings
- ✅ **Industry Standard**: Required for compliance and professional deployments
- ✅ **Free Options**: Let's Encrypt provides free, automated certificates
- ✅ **Revocation Support**: Ability to revoke compromised certificates

**Self-signed certificates should ONLY be used for:**

- Local development
- Internal testing environments
- Staging/QA environments (with proper documentation)

---

## Let's Encrypt (Recommended)

[Let's Encrypt](https://letsencrypt.org/) is a free, automated, and open Certificate Authority. It's the recommended option for most production deployments.

### Prerequisites

1. **Public Domain Name**: You need a registered domain (e.g., `api.pawchain.io`)
2. **DNS Configuration**: Domain must point to your server's public IP
3. **Port 80/443 Access**: Ports must be accessible for HTTP-01 challenge
4. **Root/Sudo Access**: Required to install certbot

### Option 1: Certbot (Recommended)

Certbot is the official Let's Encrypt client with automatic certificate management.

#### Installation

**Ubuntu/Debian:**

```bash
sudo apt-get update
sudo apt-get install certbot
```

**CentOS/RHEL:**

```bash
sudo yum install certbot
```

**macOS:**

```bash
brew install certbot
```

#### Obtaining Certificates

1. **Standalone Mode** (if you haven't started the API server yet):

```bash
# Stop any services using port 80
sudo systemctl stop nginx  # if running

# Obtain certificate
sudo certbot certonly --standalone \
  -d api.pawchain.io \
  -d www.pawchain.io \
  --email admin@pawchain.io \
  --agree-tos \
  --non-interactive
```

2. **Webroot Mode** (if you have a web server running):

```bash
sudo certbot certonly --webroot \
  -w /var/www/html \
  -d api.pawchain.io \
  --email admin@pawchain.io \
  --agree-tos
```

3. **DNS Challenge** (for wildcard certificates):

```bash
# Requires DNS plugin for your provider
sudo certbot certonly --manual \
  --preferred-challenges dns \
  -d *.pawchain.io \
  -d pawchain.io \
  --email admin@pawchain.io \
  --agree-tos
```

#### Certificate Locations

After successful issuance, certificates are stored at:

```
/etc/letsencrypt/live/api.pawchain.io/
├── fullchain.pem  → Use this as TLS_CERT_FILE
├── privkey.pem    → Use this as TLS_KEY_FILE
├── chain.pem      → Intermediate certificates
└── cert.pem       → Your domain certificate only
```

#### PAW API Configuration

Update your `config.yaml`:

```yaml
api:
  tls_enabled: true
  tls_cert_file: '/etc/letsencrypt/live/api.pawchain.io/fullchain.pem'
  tls_key_file: '/etc/letsencrypt/live/api.pawchain.io/privkey.pem'
  address: '0.0.0.0:8443'
```

Or use environment variables:

```bash
export PAW_API_TLS_ENABLED=true
export PAW_API_TLS_CERT_FILE="/etc/letsencrypt/live/api.pawchain.io/fullchain.pem"
export PAW_API_TLS_KEY_FILE="/etc/letsencrypt/live/api.pawchain.io/privkey.pem"
export PAW_API_ADDRESS="0.0.0.0:8443"
```

#### Automatic Renewal

Let's Encrypt certificates expire after **90 days**. Set up automatic renewal:

```bash
# Test renewal (dry run)
sudo certbot renew --dry-run

# Add to crontab for automatic renewal
sudo crontab -e

# Add this line to renew daily at 3 AM
0 3 * * * certbot renew --quiet --post-hook "systemctl reload pawd"
```

Or use systemd timer (modern approach):

```bash
# Enable certbot timer
sudo systemctl enable certbot.timer
sudo systemctl start certbot.timer

# Check timer status
sudo systemctl status certbot.timer
```

---

### Option 2: acme.sh (Alternative)

[acme.sh](https://github.com/acmesh-official/acme.sh) is a pure Unix shell script implementing ACME client protocol.

#### Installation

```bash
curl https://get.acme.sh | sh -s email=admin@pawchain.io
```

#### Obtaining Certificates

```bash
# HTTP challenge
acme.sh --issue -d api.pawchain.io -d www.pawchain.io --standalone

# DNS challenge (automatic with DNS provider API)
acme.sh --issue -d api.pawchain.io --dns dns_cloudflare

# Wildcard certificate
acme.sh --issue -d "*.pawchain.io" --dns dns_cloudflare
```

#### Install Certificates

```bash
acme.sh --install-cert -d api.pawchain.io \
  --key-file /etc/paw/tls/privkey.pem \
  --fullchain-file /etc/paw/tls/fullchain.pem \
  --reloadcmd "systemctl reload pawd"
```

---

## Commercial Certificate Authorities

For enterprise deployments, you may prefer commercial CAs that offer:

- Extended Validation (EV) certificates
- Organization Validation (OV) certificates
- Wildcard and multi-domain certificates
- Insurance coverage
- 24/7 support

### Recommended Commercial CAs

1. **DigiCert** - Industry leader, trusted by enterprises
   - https://www.digicert.com/
   - Pricing: $200-$1500/year

2. **Sectigo (formerly Comodo)** - Affordable option
   - https://sectigo.com/
   - Pricing: $50-$500/year

3. **GlobalSign** - Good for international deployments
   - https://www.globalsign.com/
   - Pricing: $100-$800/year

4. **GoDaddy** - Easy for small businesses
   - https://www.godaddy.com/web-security/ssl-certificate
   - Pricing: $70-$300/year

### General Process for Commercial CAs

1. **Generate CSR (Certificate Signing Request)**

```bash
# Generate private key
openssl genrsa -out pawchain.key 2048

# Generate CSR
openssl req -new -key pawchain.key -out pawchain.csr

# You'll be prompted for:
# - Country Name (2 letter code): US
# - State: California
# - Locality: San Francisco
# - Organization: PAW Blockchain Inc
# - Common Name: api.pawchain.io
```

2. **Submit CSR to CA**: Upload the `.csr` file to your chosen CA

3. **Domain Validation**: Verify domain ownership via:
   - Email validation (email to admin@pawchain.io)
   - HTTP validation (hosting a specific file)
   - DNS validation (adding a TXT record)

4. **Receive Certificate**: CA will email you the certificate and chain

5. **Install Certificate**: Download and configure as shown above

---

## Certificate Installation

### File Permissions

Secure your certificate files:

```bash
# Create TLS directory
sudo mkdir -p /etc/paw/tls
sudo chown root:root /etc/paw/tls
sudo chmod 755 /etc/paw/tls

# Set file permissions
sudo chmod 600 /etc/paw/tls/privkey.pem   # Private key - most restrictive
sudo chmod 644 /etc/paw/tls/fullchain.pem # Certificate - readable
```

### Systemd Service Configuration

If using systemd to manage PAW daemon:

```ini
# /etc/systemd/system/pawd.service
[Unit]
Description=PAW Blockchain Node
After=network.target

[Service]
Type=simple
User=paw
Group=paw
Environment="PAW_API_TLS_ENABLED=true"
Environment="PAW_API_TLS_CERT_FILE=/etc/paw/tls/fullchain.pem"
Environment="PAW_API_TLS_KEY_FILE=/etc/paw/tls/privkey.pem"
Environment="PAW_API_ADDRESS=0.0.0.0:8443"
ExecStart=/usr/local/bin/pawd start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart pawd
```

### Testing Configuration

Test the TLS connection:

```bash
# Test with OpenSSL
openssl s_client -connect api.pawchain.io:8443 -servername api.pawchain.io

# Test with curl
curl -v https://api.pawchain.io:8443/api/v1/health

# Check certificate details
echo | openssl s_client -connect api.pawchain.io:8443 -servername api.pawchain.io 2>/dev/null | openssl x509 -noout -text
```

Expected output should show:

- ✅ Certificate chain validated
- ✅ Valid dates (not expired)
- ✅ Correct subject/issuer
- ✅ Proper SANs (Subject Alternative Names)

---

## Certificate Renewal

### Let's Encrypt Renewal Monitoring

Monitor renewal status:

```bash
# List all certificates and expiry dates
sudo certbot certificates

# Force renewal (if <30 days until expiry)
sudo certbot renew --force-renewal

# Renew specific domain
sudo certbot renew --cert-name api.pawchain.io
```

### Renewal Notifications

Set up monitoring and alerts:

1. **Email Alerts**: Let's Encrypt sends expiry warnings to registered email

2. **Monitoring Script**:

```bash
#!/bin/bash
# /usr/local/bin/check-cert-expiry.sh

DOMAIN="api.pawchain.io"
DAYS_WARNING=30

# Get expiry date
EXPIRY=$(echo | openssl s_client -servername $DOMAIN -connect $DOMAIN:8443 2>/dev/null | openssl x509 -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s)
NOW_EPOCH=$(date +%s)
DAYS_LEFT=$(( ($EXPIRY_EPOCH - $NOW_EPOCH) / 86400 ))

if [ $DAYS_LEFT -lt $DAYS_WARNING ]; then
    echo "WARNING: Certificate for $DOMAIN expires in $DAYS_LEFT days!"
    # Send alert (email, Slack, PagerDuty, etc.)
fi
```

3. **Add to Cron**:

```bash
# Check daily at 9 AM
0 9 * * * /usr/local/bin/check-cert-expiry.sh
```

### Commercial CA Renewal

Commercial certificates typically last 1 year. Renewal process:

1. **60 Days Before Expiry**: Start renewal process
2. **Generate New CSR**: Use same process as initial purchase
3. **Submit to CA**: Most CAs offer renewal discounts
4. **Install New Certificate**: Replace old files with new ones
5. **Reload Service**: `sudo systemctl reload pawd`

---

## Security Best Practices

### 1. Use Strong Cipher Suites

Ensure your API server config uses TLS 1.3 with secure ciphers (already implemented in PAW):

```go
// In api/server.go
TLSConfig: &tls.Config{
    MinVersion: tls.VersionTLS13,
    CipherSuites: []uint16{
        tls.TLS_AES_128_GCM_SHA256,
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
    },
}
```

### 2. Enable HSTS (HTTP Strict Transport Security)

Force clients to use HTTPS:

```go
// Add to API middleware
c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
```

### 3. Rotate Certificates Regularly

Even though certificates are valid for 90 days (Let's Encrypt) or 1 year (commercial), rotate them more frequently for security.

### 4. Monitor Certificate Transparency Logs

Use services like https://crt.sh/ to monitor all certificates issued for your domain.

### 5. Implement Certificate Pinning (Advanced)

For critical deployments, pin specific certificates or public keys in client applications.

### 6. Secure Private Keys

- **Never commit** private keys to version control
- Store in **encrypted volumes** or **HSM** for production
- Use **restrictive file permissions** (600)
- Consider **key escrow** for disaster recovery

### 7. Regular Security Audits

- Run SSL Labs test: https://www.ssllabs.com/ssltest/
- Use testssl.sh: https://github.com/drwetter/testssl.sh
- Monitor for vulnerabilities: https://cve.mitre.org/

---

## Troubleshooting

### Certificate Chain Issues

**Problem**: Clients can't verify certificate chain

**Solution**: Ensure you're using `fullchain.pem` (not just `cert.pem`)

```bash
# Verify chain
openssl verify -CAfile /etc/ssl/certs/ca-certificates.crt /etc/paw/tls/fullchain.pem
```

### Permission Denied

**Problem**: Server can't read certificate files

**Solution**: Check file permissions and ownership

```bash
# Make readable by PAW service user
sudo chown paw:paw /etc/paw/tls/*.pem
sudo chmod 600 /etc/paw/tls/privkey.pem
sudo chmod 644 /etc/paw/tls/fullchain.pem
```

### Certificate Expired

**Problem**: Certificate has expired

**Solution**: Renew immediately

```bash
sudo certbot renew --force-renewal
sudo systemctl reload pawd
```

### Wrong Certificate Served

**Problem**: Server serves wrong certificate for domain

**Solution**: Check SNI (Server Name Indication) configuration

```bash
# Test with specific SNI
openssl s_client -connect api.pawchain.io:8443 -servername api.pawchain.io
```

### Rate Limiting (Let's Encrypt)

**Problem**: Hit Let's Encrypt rate limits

**Solution**:

- Certificates per Registered Domain: 50 per week
- Use staging environment for testing: `--test-cert`
- Wait for rate limit reset (weekly)

```bash
# Use staging for testing
certbot certonly --staging --standalone -d test.pawchain.io
```

---

## Quick Reference

### Let's Encrypt Certificate Paths

```
Certificate:     /etc/letsencrypt/live/DOMAIN/fullchain.pem
Private Key:     /etc/letsencrypt/live/DOMAIN/privkey.pem
Chain:           /etc/letsencrypt/live/DOMAIN/chain.pem
Cert Only:       /etc/letsencrypt/live/DOMAIN/cert.pem
```

### Common Commands

```bash
# List certificates
sudo certbot certificates

# Renew all
sudo certbot renew

# Revoke certificate
sudo certbot revoke --cert-path /etc/letsencrypt/live/DOMAIN/cert.pem

# Delete certificate
sudo certbot delete --cert-name DOMAIN

# Check expiry
echo | openssl s_client -connect DOMAIN:443 -servername DOMAIN 2>/dev/null | openssl x509 -noout -dates
```

### Support Resources

- Let's Encrypt Community: https://community.letsencrypt.org/
- Certbot Documentation: https://eff-certbot.readthedocs.io/
- SSL/TLS Best Practices: https://wiki.mozilla.org/Security/Server_Side_TLS

---

**Last Updated**: 2025-11-13
**Version**: 1.0

For development/testing certificates, see: `scripts/generate-tls-certs.sh` or `scripts/generate-tls-certs.ps1`
