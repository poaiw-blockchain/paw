# TLS/SSL Configuration for PAW Blockchain

## Overview

This directory contains TLS/SSL configuration templates for securing PAW blockchain endpoints in production environments.

## Endpoints Requiring TLS

1. **RPC Endpoint** (port 26657) - Tendermint RPC
2. **gRPC Endpoint** (port 9090) - Cosmos gRPC
3. **REST API** (port 1317) - Cosmos REST API
4. **gRPC-Web** (port 9091) - gRPC-Web gateway
5. **P2P** (port 26656) - Optional: mTLS for enhanced security

## Quick Start

### 1. Generate Certificates

#### Option A: Let's Encrypt (Recommended for Production)

```bash
# Install certbot
sudo apt-get update
sudo apt-get install certbot

# Generate certificates
sudo certbot certonly --standalone \
  -d rpc.paw.network \
  -d grpc.paw.network \
  -d api.paw.network \
  --agree-tos \
  --email admin@paw.network

# Certificates will be in /etc/letsencrypt/live/
```

#### Option B: Self-Signed (Development/Testing Only)

```bash
./generate-self-signed.sh
```

### 2. Configure Node

Edit `~/.paw/config/config.toml`:

```toml
#######################################################
###           RPC Server Configuration            ###
#######################################################

[rpc]
laddr = "tcp://0.0.0.0:26657"

# Enable TLS
tls_cert_file = "/etc/paw/tls/server.crt"
tls_key_file = "/etc/paw/tls/server.key"
```

Edit `~/.paw/config/app.toml`:

```toml
#######################################################
###           gRPC Configuration                  ###
#######################################################

[grpc]
address = "0.0.0.0:9090"
enable = true

# Enable TLS
tls-cert-path = "/etc/paw/tls/server.crt"
tls-key-path = "/etc/paw/tls/server.key"

#######################################################
###           API Configuration                   ###
#######################################################

[api]
enable = true
swagger = false
address = "tcp://0.0.0.0:1317"

# Enable TLS
tls-cert-path = "/etc/paw/tls/server.crt"
tls-key-path = "/etc/paw/tls/server.key"
```

### 3. Restart Node

```bash
sudo systemctl restart pawd
```

## Detailed Configuration

### Certificate Requirements

- **Key Algorithm**: RSA 2048-bit or ECDSA P-256
- **Validity Period**: 90 days (Let's Encrypt) or 1 year (commercial CA)
- **Subject Alternative Names**: All domain names the node serves
- **Key Usage**: Digital Signature, Key Encipherment
- **Extended Key Usage**: Server Authentication

### Let's Encrypt Integration

#### Automatic Renewal

Create `/etc/systemd/system/certbot-renewal.service`:

```ini
[Unit]
Description=Certbot Renewal

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --deploy-hook /usr/local/bin/paw-cert-deploy.sh
```

Create `/etc/systemd/system/certbot-renewal.timer`:

```ini
[Unit]
Description=Certbot Renewal Timer

[Timer]
OnCalendar=daily
RandomizedDelaySec=1h

[Install]
WantedBy=timers.target
```

Create deployment hook `/usr/local/bin/paw-cert-deploy.sh`:

```bash
#!/bin/bash
# Deploy renewed certificates

set -e

CERT_DIR="/etc/letsencrypt/live/rpc.paw.network"
PAW_TLS_DIR="/etc/paw/tls"

# Copy certificates
cp "$CERT_DIR/fullchain.pem" "$PAW_TLS_DIR/server.crt"
cp "$CERT_DIR/privkey.pem" "$PAW_TLS_DIR/server.key"

# Set permissions
chown paw:paw "$PAW_TLS_DIR/server.crt" "$PAW_TLS_DIR/server.key"
chmod 600 "$PAW_TLS_DIR/server.key"
chmod 644 "$PAW_TLS_DIR/server.crt"

# Restart PAW node
systemctl restart pawd

echo "Certificates deployed successfully"
```

Enable automatic renewal:

```bash
sudo chmod +x /usr/local/bin/paw-cert-deploy.sh
sudo systemctl enable certbot-renewal.timer
sudo systemctl start certbot-renewal.timer
```

### Nginx Reverse Proxy with TLS

For better security and performance, use Nginx as a reverse proxy:

Create `/etc/nginx/sites-available/paw-rpc`:

```nginx
# RPC Endpoint
server {
    listen 443 ssl http2;
    server_name rpc.paw.network;

    # TLS Configuration
    ssl_certificate /etc/letsencrypt/live/rpc.paw.network/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/rpc.paw.network/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;

    # Rate Limiting
    limit_req_zone $binary_remote_addr zone=rpc_limit:10m rate=10r/s;
    limit_req zone=rpc_limit burst=20 nodelay;

    # Proxy Configuration
    location / {
        proxy_pass http://127.0.0.1:26657;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # WebSocket support
    location /websocket {
        proxy_pass http://127.0.0.1:26657/websocket;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name rpc.paw.network;
    return 301 https://$server_name$request_uri;
}
```

Create `/etc/nginx/sites-available/paw-grpc`:

```nginx
# gRPC Endpoint
server {
    listen 9090 ssl http2;
    server_name grpc.paw.network;

    # TLS Configuration
    ssl_certificate /etc/letsencrypt/live/grpc.paw.network/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/grpc.paw.network/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # gRPC Configuration
    location / {
        grpc_pass grpc://127.0.0.1:9090;
        grpc_set_header X-Real-IP $remote_addr;
        grpc_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # Timeouts
        grpc_read_timeout 300s;
        grpc_send_timeout 300s;
    }
}
```

Enable configurations:

```bash
sudo ln -s /etc/nginx/sites-available/paw-rpc /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/paw-grpc /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### mTLS for P2P Communication

For enhanced security between validator nodes, configure mutual TLS:

1. **Generate CA and certificates:**

```bash
./generate-mtls-certs.sh
```

2. **Configure P2P TLS in `config.toml`:**

```toml
#######################################################
###           P2P Configuration                   ###
#######################################################

[p2p]
laddr = "tcp://0.0.0.0:26656"

# Enable P2P TLS
p2p_tls_cert_file = "/etc/paw/tls/p2p-cert.pem"
p2p_tls_key_file = "/etc/paw/tls/p2p-key.pem"
p2p_tls_ca_file = "/etc/paw/tls/ca-cert.pem"

# Only accept connections from nodes with valid certificates
p2p_tls_require_client_cert = true
```

## Security Best Practices

### Certificate Management

1. **Use Strong Keys**
   - RSA 2048-bit minimum (4096-bit recommended)
   - Or ECDSA P-256/P-384

2. **Regular Rotation**
   - Rotate certificates every 90 days (automated with Let's Encrypt)
   - Rotate CA certificates annually for mTLS

3. **Secure Storage**
   - Private keys: 0600 permissions, root/service user only
   - Certificates: 0644 permissions
   - Store backups encrypted

4. **Monitoring**
   - Monitor certificate expiration dates
   - Alert 30 days before expiry
   - Track TLS handshake failures

### TLS Protocol Configuration

1. **Disable Weak Protocols**
   - Disable SSLv3, TLSv1.0, TLSv1.1
   - Use only TLSv1.2 and TLSv1.3

2. **Strong Cipher Suites**
   ```
   TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
   TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
   TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
   TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
   ```

3. **Enable Perfect Forward Secrecy**
   - Use ECDHE key exchange
   - Regenerate session keys regularly

### Firewall Configuration

```bash
# Allow only HTTPS traffic
sudo ufw allow 443/tcp
sudo ufw allow 9090/tcp

# Allow P2P TLS (if using mTLS)
sudo ufw allow from <trusted-validator-ip> to any port 26656 proto tcp

# Block direct access to unencrypted endpoints
sudo ufw deny 26657/tcp  # RPC
sudo ufw deny 1317/tcp   # REST API
```

## Testing

### Verify TLS Configuration

```bash
# Test RPC endpoint
curl -v https://rpc.paw.network/status

# Test certificate
openssl s_client -connect rpc.paw.network:443 -servername rpc.paw.network

# Test gRPC endpoint
grpcurl -v grpc.paw.network:9090 list

# Check certificate expiration
echo | openssl s_client -servername rpc.paw.network -connect rpc.paw.network:443 2>/dev/null | openssl x509 -noout -dates
```

### SSL Labs Testing

For public endpoints, test with SSL Labs:
- https://www.ssllabs.com/ssltest/

Target: A+ rating

## Troubleshooting

### Certificate Errors

**Problem**: "certificate signed by unknown authority"
```bash
# Check certificate chain
openssl s_client -connect rpc.paw.network:443 -showcerts

# Ensure fullchain.pem includes intermediate certificates
```

**Problem**: "certificate has expired"
```bash
# Check expiration
openssl x509 -in /etc/paw/tls/server.crt -noout -enddate

# Renew certificate
sudo certbot renew --force-renewal
```

### Connection Errors

**Problem**: "connection refused"
```bash
# Check if service is listening on TLS port
sudo netstat -tlnp | grep 443

# Check firewall
sudo ufw status
```

**Problem**: "TLS handshake failed"
```bash
# Check TLS configuration
openssl s_client -connect localhost:443 -tls1_2

# Review logs
journalctl -u pawd -f
```

## Additional Resources

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [NIST TLS Guidelines](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-52r2.pdf)
- [Cosmos SDK gRPC TLS](https://docs.cosmos.network/main/core/grpc_rest.html#grpc-server)

## Support

For issues or questions:
-  Issues: https://github.com/paw-chain/paw/issues
- Security Issues: security@paw.network
