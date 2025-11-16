# TLS Certificates Directory

This directory contains TLS certificates for the PAW Blockchain API server.

## Directory Structure

```
certs/
├── README.md                        # This file
├── PRODUCTION_CERTIFICATES.md       # Production certificate guide
├── .gitignore                       # Prevents committing certificate files
├── dev/                             # Development certificates (self-signed)
│   ├── ca-cert.pem
│   ├── ca-key.pem
│   ├── server-cert.pem
│   ├── server-key.pem
│   ├── openssl.cnf
│   └── server-config-example.yaml
└── staging/                         # Staging certificates (self-signed)
    ├── ca-cert.pem
    ├── ca-key.pem
    ├── server-cert.pem
    ├── server-key.pem
    ├── openssl.cnf
    └── server-config-example.yaml
```

## Quick Start

### Development / Testing

Generate self-signed certificates for local development:

**Linux / macOS / Git Bash:**

```bash
./scripts/generate-tls-certs.sh dev
```

**Windows PowerShell:**

```powershell
.\scripts\generate-tls-certs.ps1 -Environment dev
```

This will create certificates in `certs/dev/` directory.

### Staging Environment

Generate self-signed certificates for staging:

**Linux / macOS / Git Bash:**

```bash
./scripts/generate-tls-certs.sh staging
```

**Windows PowerShell:**

```powershell
.\scripts\generate-tls-certs.ps1 -Environment staging
```

This will create certificates in `certs/staging/` directory.

### Production

**⚠️ NEVER use self-signed certificates in production!**

For production deployments, obtain certificates from a trusted Certificate Authority:

1. **Let's Encrypt (Free, Recommended)**
   - See: [PRODUCTION_CERTIFICATES.md](PRODUCTION_CERTIFICATES.md#lets-encrypt-recommended)
   - Automated, 90-day validity, auto-renewal

2. **Commercial CAs (DigiCert, Sectigo, GlobalSign)**
   - See: [PRODUCTION_CERTIFICATES.md](PRODUCTION_CERTIFICATES.md#commercial-certificate-authorities)
   - Extended validation, 1-year validity, support

## Using Certificates

### Configure API Server

Edit your `config.yaml`:

```yaml
api:
  tls_enabled: true
  tls_cert_file: './certs/dev/server-cert.pem'
  tls_key_file: './certs/dev/server-key.pem'
  address: '0.0.0.0:8443'
```

Or use environment variables:

```bash
export PAW_API_TLS_ENABLED=true
export PAW_API_TLS_CERT_FILE="./certs/dev/server-cert.pem"
export PAW_API_TLS_KEY_FILE="./certs/dev/server-key.pem"
export PAW_API_ADDRESS="0.0.0.0:8443"
```

### Trust Self-Signed Certificates (Development Only)

#### macOS

```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain \
  ./certs/dev/ca-cert.pem
```

#### Linux (Ubuntu/Debian)

```bash
sudo cp ./certs/dev/ca-cert.pem /usr/local/share/ca-certificates/paw-ca.crt
sudo update-ca-certificates
```

#### Windows

1. Open Certificate Manager (`certmgr.msc`)
2. Navigate to: **Trusted Root Certification Authorities > Certificates**
3. Right-click > **All Tasks > Import**
4. Import: `certs\dev\ca-cert.pem`

Or use PowerShell (Admin):

```powershell
Import-Certificate -FilePath ".\certs\dev\ca-cert.pem" `
  -CertStoreLocation Cert:\LocalMachine\Root
```

### Test TLS Connection

```bash
# Test with curl (using CA certificate)
curl --cacert ./certs/dev/ca-cert.pem https://localhost:8443/api/v1/health

# Test with OpenSSL
openssl s_client -connect localhost:8443 -servername localhost

# Verify certificate details
openssl x509 -in ./certs/dev/server-cert.pem -noout -text
```

## Security Notes

### DO NOT Commit Private Keys

The `.gitignore` file in this directory prevents committing:

- `*.pem` - Certificate and key files
- `*.key` - Private keys
- `*.crt` - Certificate files
- `*.csr` - Certificate signing requests

### File Permissions

Ensure proper permissions on certificate files:

**Linux/macOS:**

```bash
chmod 600 ./certs/dev/server-key.pem    # Private key - restrictive
chmod 644 ./certs/dev/server-cert.pem   # Certificate - readable
```

**Windows:**

```powershell
icacls ".\certs\dev\server-key.pem" /inheritance:r /grant:r "$env:USERNAME:(R)"
```

### Certificate Validity

- **Development certificates**: Valid for 365 days from generation
- **Let's Encrypt**: Valid for 90 days, automatic renewal
- **Commercial CAs**: Typically 1 year

### Renewal

Self-signed certificates must be manually regenerated before expiry:

```bash
# Check certificate expiry
openssl x509 -in ./certs/dev/server-cert.pem -noout -enddate

# Regenerate if needed
./scripts/generate-tls-certs.sh dev
```

## Certificate Components

### CA Certificate (`ca-cert.pem`)

- Certificate Authority certificate
- Used to sign server certificates
- Import into client trust stores for development

### CA Private Key (`ca-key.pem`)

- CA's private key
- **HIGHLY SENSITIVE** - protect with restrictive permissions
- Required to sign new server certificates

### Server Certificate (`server-cert.pem`)

- Server's public certificate
- Presented to clients during TLS handshake
- Contains domain names and IP addresses (SANs)

### Server Private Key (`server-key.pem`)

- Server's private key
- **HIGHLY SENSITIVE** - never share or commit
- Used to decrypt TLS traffic

### OpenSSL Config (`openssl.cnf`)

- Configuration for certificate generation
- Defines Subject Alternative Names (SANs)
- Specifies key usage and extensions

## Troubleshooting

### Certificate Not Trusted

**Problem**: Browser shows "Certificate not trusted" warning

**Solution**:

1. Import CA certificate into system trust store (see above)
2. Restart browser
3. For production, use a trusted CA (Let's Encrypt)

### Connection Refused

**Problem**: `curl: (7) Failed to connect to localhost port 8443`

**Solution**:

1. Verify API server is running: `ps aux | grep pawd`
2. Check TLS is enabled in config
3. Verify port 8443 is listening: `netstat -an | grep 8443` (Linux) or `netstat -an | findstr 8443` (Windows)

### Certificate Expired

**Problem**: `certificate has expired or is not yet valid`

**Solution**:

1. Check expiry: `openssl x509 -in ./certs/dev/server-cert.pem -noout -dates`
2. Regenerate: `./scripts/generate-tls-certs.sh dev`
3. Restart API server

### Wrong Certificate

**Problem**: Server serves certificate for different domain

**Solution**:

1. Verify SANs: `openssl x509 -in ./certs/dev/server-cert.pem -noout -text | grep -A1 "Subject Alternative Name"`
2. Ensure config points to correct certificate files
3. Regenerate if needed

### Permission Denied

**Problem**: API server can't read certificate files

**Solution**:

```bash
# Check permissions
ls -la ./certs/dev/

# Fix permissions
chmod 600 ./certs/dev/server-key.pem
chmod 644 ./certs/dev/server-cert.pem

# If running as different user, fix ownership
sudo chown paw:paw ./certs/dev/*.pem
```

## Additional Resources

- **OpenSSL Documentation**: https://www.openssl.org/docs/
- **Let's Encrypt**: https://letsencrypt.org/
- **SSL Labs Test**: https://www.ssllabs.com/ssltest/
- **Mozilla SSL Configuration Generator**: https://ssl-config.mozilla.org/

## Support

For issues or questions:

1. Check [PRODUCTION_CERTIFICATES.md](PRODUCTION_CERTIFICATES.md) for detailed guides
2. Review logs: `journalctl -u pawd -f` (Linux) or Windows Event Viewer
3. Test with OpenSSL: `openssl s_client -connect localhost:8443 -servername localhost`
4. Create an issue on GitHub

---

**Generated**: 2025-11-13
**Version**: 1.0
**Maintained by**: PAW Blockchain Security Team
