# TLS/SSL Configuration Guide for PAW Chain

This guide covers TLS/SSL configuration for PAW Chain services, including certificate management and Let's Encrypt automation.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [NGINX TLS Configuration](#nginx-tls-configuration)
4. [Development Certificates](#development-certificates)
5. [Production Certificates (Let's Encrypt)](#production-certificates-lets-encrypt)
6. [Certificate Management](#certificate-management)
7. [Monitoring and Renewal](#monitoring-and-renewal)
8. [Troubleshooting](#troubleshooting)

## Overview

PAW Chain implements production-ready TLS/SSL configurations with:

- **TLS 1.3** support with fallback to TLS 1.2
- **HTTP/2** for improved performance
- **OCSP Stapling** for certificate validation
- **Strong cipher suites** following industry best practices
- **Security headers** (HSTS, CSP, etc.)
- **Rate limiting** to prevent abuse
- **Automatic certificate renewal** with Let's Encrypt

## Prerequisites

### System Requirements

- Ubuntu 20.04+ / Debian 11+ / CentOS 8+
- NGINX 1.18+ or Docker
- OpenSSL 1.1.1+
- Certbot (for Let's Encrypt)

### Installation

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y nginx certbot python3-certbot-nginx openssl
```

**CentOS/RHEL:**
```bash
sudo yum install -y nginx certbot python3-certbot-nginx openssl
```

**macOS:**
```bash
brew install nginx certbot openssl
```

## NGINX TLS Configuration

### Configuration File

The main NGINX configuration is located at:
```
/home/decri/blockchain-projects/paw/explorer/nginx.conf
```

### Key Features

1. **Automatic HTTP to HTTPS Redirect**
   - All HTTP traffic redirected to HTTPS
   - Exception for Let's Encrypt ACME challenge

2. **TLS Configuration**
   - TLS 1.2 and 1.3 support
   - Strong cipher suite preference
   - Session optimization

3. **Security Headers**
   - HSTS with preload
   - Content Security Policy
   - X-Frame-Options
   - X-Content-Type-Options

4. **Performance Optimization**
   - HTTP/2 enabled
   - Gzip compression
   - Static asset caching
   - Connection pooling

### Deployment

**Copy configuration:**
```bash
sudo cp explorer/nginx.conf /etc/nginx/sites-available/paw-explorer
sudo ln -s /etc/nginx/sites-available/paw-explorer /etc/nginx/sites-enabled/
```

**Test configuration:**
```bash
sudo nginx -t
```

**Reload NGINX:**
```bash
sudo systemctl reload nginx
```

## Development Certificates

For local development and testing, use self-signed certificates.

### Generate Development Certificates

```bash
cd /home/decri/blockchain-projects/paw
./scripts/tls/generate-dev-certs.sh localhost
```

This creates:
- `certs/privkey.pem` - Private key
- `certs/fullchain.pem` - Certificate
- `certs/chain.pem` - Certificate chain
- `certs/dhparam.pem` - DH parameters

### Custom Domain

Generate certificates for a custom domain:
```bash
./scripts/tls/generate-dev-certs.sh dev.pawchain.local
```

### Trust Certificate Locally

**macOS:**
```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain certs/fullchain.pem
```

**Linux:**
```bash
sudo cp certs/fullchain.pem /usr/local/share/ca-certificates/paw-dev.crt
sudo update-ca-certificates
```

**Windows:**
1. Open `certs/fullchain.pem`
2. Install Certificate → Trusted Root Certification Authorities

## Production Certificates (Let's Encrypt)

### Initial Setup

1. **Update DNS records:**
   ```bash
   # Ensure your domain points to your server
   dig explorer.pawchain.network
   ```

2. **Run setup script:**
   ```bash
   sudo ./scripts/tls/setup-letsencrypt.sh \
     --domain explorer.pawchain.network \
     --email admin@pawchain.network
   ```

3. **Verify certificate:**
   ```bash
   sudo ./scripts/tls/validate-cert.sh
   ```

### Multiple Domains

Request certificates for multiple domains:
```bash
sudo certbot certonly --webroot \
  -w /var/www/certbot \
  -d explorer.pawchain.network \
  -d www.explorer.pawchain.network \
  -d api.pawchain.network \
  --email admin@pawchain.network \
  --agree-tos
```

### Wildcard Certificates

For wildcard certificates, use DNS challenge:
```bash
sudo certbot certonly --dns-route53 \
  -d '*.pawchain.network' \
  -d pawchain.network \
  --email admin@pawchain.network \
  --agree-tos
```

## Certificate Management

### Check Certificate Expiry

```bash
./scripts/tls/check-cert-expiry.sh
```

### Validate Certificate

Comprehensive certificate validation:
```bash
./scripts/tls/validate-cert.sh /etc/letsencrypt/live/explorer.pawchain.network
```

This checks:
- File existence and permissions
- Certificate and key matching
- Expiry dates
- Key strength
- Signature algorithm

### Manual Renewal

Force certificate renewal:
```bash
./scripts/tls/renew-cert.sh --force
```

Renew specific certificate:
```bash
./scripts/tls/renew-cert.sh --cert-name explorer.pawchain.network
```

## Monitoring and Renewal

### Setup Automatic Renewal

```bash
sudo ./scripts/tls/setup-cert-renewal.sh
```

This configures automatic renewal using:
- **systemd timer** (preferred on modern Linux)
- **cron job** (fallback)

### Verify Auto-Renewal

**Check systemd timer:**
```bash
systemctl status certbot.timer
systemctl list-timers certbot.timer
```

**Check cron job:**
```bash
crontab -l | grep certbot
```

### Test Renewal

Dry run to test renewal:
```bash
sudo certbot renew --dry-run
```

### Monitoring Script

Run certificate monitoring:
```bash
sudo /usr/local/bin/check-cert-expiry.sh
```

Add to monitoring stack:
```bash
# Prometheus exporter example
curl https://explorer.pawchain.network | \
  openssl s_client -connect explorer.pawchain.network:443 2>/dev/null | \
  openssl x509 -noout -dates
```

## Docker Deployment

### Using Docker Compose

```bash
cd compose
docker-compose -f docker-compose.certbot.yml up -d
```

This starts:
- NGINX with SSL support
- Certbot for certificate management
- Automatic renewal service

### Environment Variables

Create `.env` file:
```bash
DOMAIN=explorer.pawchain.network
CERTBOT_EMAIL=admin@pawchain.network
```

### View Logs

```bash
docker logs paw-certbot
docker logs paw-certbot-renew
docker logs paw-nginx
```

## Security Best Practices

### 1. Strong Cipher Suites

The configuration uses:
- TLS 1.3: `TLS_AES_256_GCM_SHA384`, `TLS_CHACHA20_POLY1305_SHA256`
- TLS 1.2: `ECDHE-RSA-AES256-GCM-SHA384`

### 2. HSTS (HTTP Strict Transport Security)

```
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
```

Submit to HSTS preload list:
https://hstspreload.org/

### 3. OCSP Stapling

Enabled for improved certificate validation performance.

### 4. Perfect Forward Secrecy

DH parameters generated for enhanced security.

### 5. Certificate Pinning (Optional)

For mobile apps, implement certificate pinning:
```typescript
// Example for React Native
const certHash = 'sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=';
```

## Troubleshooting

### Common Issues

#### 1. Certificate Not Found

**Error:** `Certificate file not found`

**Solution:**
```bash
# Check certificate location
sudo certbot certificates

# Regenerate if missing
sudo certbot certonly --webroot -w /var/www/certbot \
  -d explorer.pawchain.network
```

#### 2. Permission Denied

**Error:** `Permission denied when accessing certificate`

**Solution:**
```bash
# Fix permissions
sudo chmod 644 /etc/letsencrypt/live/*/fullchain.pem
sudo chmod 600 /etc/letsencrypt/live/*/privkey.pem
```

#### 3. NGINX Won't Start

**Error:** `nginx: [emerg] SSL_CTX_use_PrivateKey_file() failed`

**Solution:**
```bash
# Validate certificate and key match
./scripts/tls/validate-cert.sh

# Test NGINX config
sudo nginx -t
```

#### 4. Certificate Expired

**Error:** `Certificate has expired`

**Solution:**
```bash
# Renew immediately
sudo certbot renew --force-renewal

# Reload NGINX
sudo systemctl reload nginx
```

#### 5. Rate Limit Exceeded

**Error:** `too many certificates already issued`

**Solution:**
- Use staging environment for testing: `--staging`
- Wait for rate limit reset (weekly)
- Use existing certificate

### Debug Mode

Enable debug logging:
```bash
# NGINX debug mode
sudo nginx -T

# Certbot verbose mode
sudo certbot renew --verbose --debug
```

### SSL Test

Test SSL configuration:
```bash
# Using OpenSSL
openssl s_client -connect explorer.pawchain.network:443 -tls1_3

# Using SSL Labs
# Visit: https://www.ssllabs.com/ssltest/
```

## Certificate Locations

### Let's Encrypt

```
/etc/letsencrypt/
├── live/
│   └── explorer.pawchain.network/
│       ├── fullchain.pem  (Certificate + Chain)
│       ├── privkey.pem    (Private Key)
│       ├── chain.pem      (Intermediate Chain)
│       └── cert.pem       (Certificate Only)
├── archive/              (Certificate versions)
└── renewal/              (Renewal configs)
```

### Development Certificates

```
./certs/
├── fullchain.pem
├── privkey.pem
├── chain.pem
└── dhparam.pem
```

## Performance Optimization

### 1. Session Resumption

Configured in nginx.conf:
```nginx
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_session_tickets off;
```

### 2. OCSP Stapling

```nginx
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
```

### 3. HTTP/2

```nginx
listen 443 ssl http2;
```

## Compliance

### PCI DSS Compliance

Configuration meets PCI DSS requirements:
- TLS 1.2+ only
- Strong cipher suites
- Regular certificate rotation

### HIPAA Compliance

Additional requirements:
- Encrypt data in transit (✓ TLS 1.3)
- Access controls (configured)
- Audit logging (NGINX logs)

### GDPR Compliance

- Secure data transmission (✓)
- Certificate for EU servers
- Privacy headers configured

## Resources

### Official Documentation
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [NGINX SSL/TLS Guide](https://nginx.org/en/docs/http/ngx_http_ssl_module.html)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)

### Testing Tools
- [SSL Labs SSL Test](https://www.ssllabs.com/ssltest/)
- [Security Headers](https://securityheaders.com/)
- [HTTP Security Report](https://httpsecurityreport.com/)

### PAW Chain Support
- Documentation: https://docs.pawchain.network
- Support: support@pawchain.network
- : https://github.com/paw-chain/paw

## License

This documentation is part of the PAW Chain project and is licensed under MIT License.
