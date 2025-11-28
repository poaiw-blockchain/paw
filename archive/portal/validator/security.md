# Validator Security

Comprehensive security guide for PAW validators.

## Key Security

### Protect Private Keys

- Use hardware security modules (HSM)
- Enable key encryption at rest
- Use Tendermint KMS for remote signing
- Never expose priv_validator_key.json
- Store encrypted backups offline

### HSM Integration

```bash
# Install Tendermint KMS
wget https://github.com/iqlusioninc/tmkms/releases/download/v0.12.2/tmkms-linux-amd64
chmod +x tmkms-linux-amd64
sudo mv tmkms-linux-amd64 /usr/local/bin/tmkms

# Initialize KMS
tmkms init ~/tmkms
```

## Network Security

### Firewall Configuration

```bash
# Lock down ports
sudo ufw default deny incoming
sudo ufw allow 22/tcp  # SSH only from known IPs
sudo ufw allow from YOUR_IP to any port 22
sudo ufw allow 26656/tcp  # P2P
sudo ufw enable
```

### Sentry Node Architecture

```
       Internet
           |
    [Load Balancer]
        /    \
   [Sentry] [Sentry]
        \    /
      [Validator]
       (private)
```

### DDoS Protection

- Use Cloudflare or similar CDN
- Implement rate limiting
- Deploy sentry nodes
- Hide validator IP

## SSH Hardening

```bash
# Disable password auth
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

# Use SSH keys only
ssh-keygen -t ed25519

# Disable root login
sudo sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config

# Restart SSH
sudo systemctl restart sshd
```

## System Hardening

```bash
# Auto security updates
sudo apt install unattended-upgrades
sudo dpkg-reconfigure --priority=low unattended-upgrades

# Fail2ban for SSH
sudo apt install fail2ban
sudo systemctl enable fail2ban
```

## Monitoring & Alerts

- Set up uptime monitoring
- Configure alerting (Discord, Telegram, email)
- Monitor disk space, CPU, memory
- Track signing performance

---

**Previous:** [Operations](/validator/operations) | **Next:** [Monitoring](/validator/monitoring) →
