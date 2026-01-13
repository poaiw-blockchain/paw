# PAW Testnet Sentry Node Architecture

## Overview

Sentry nodes protect validators from DDoS attacks by acting as a public-facing shield. Validators connect only to trusted sentry nodes via private network, never exposing their IP to the public internet.

```
                    Public Internet
                          |
            +-------------+-------------+
            |             |             |
       [Sentry 1]    [Sentry 2]    [Sentry N]
            |             |             |
            +------+------+------+------+
                   |   WireGuard VPN (10.10.0.x)
              [Validator]
```

## MVP Testnet Status

**Current Phase**: MVP Testnet (sentry nodes optional)

For MVP testnets, sentry nodes are not required but are recommended when:
- External validators join the network
- DDoS protection becomes necessary
- Preparing for mainnet launch

## Current Infrastructure

| Node | Server | IP | VPN IP | Role |
|------|--------|-----|--------|------|
| val1 | paw-testnet | 54.39.103.49 | 10.10.0.2 | Validator |
| val2 | paw-testnet | 54.39.103.49 | 10.10.0.2 | Validator |
| val3 | services-testnet | 139.99.149.160 | 10.10.0.4 | Validator |
| val4 | services-testnet | 139.99.149.160 | 10.10.0.4 | Validator |
| sentry1 | services-testnet | 139.99.149.160 | 10.10.0.4 | Sentry (active) |

## Sentry Node Details

- **Node ID**: `ce6afbda0a4443139ad14d2b856cca586161f00d`
- **P2P Address**: `139.99.149.160:12056`
- **Peer String**: `ce6afbda0a4443139ad14d2b856cca586161f00d@139.99.149.160:12056`

## Sentry Node Port Allocation

| Service | Port | Description |
|---------|------|-------------|
| P2P | 12056 | Peer-to-peer networking |
| RPC | 12057 | JSON-RPC (public) |
| gRPC | 12090 | gRPC (optional) |
| REST | 12017 | REST API (optional) |

## Quick Start

### Deploy Sentry Node

```bash
# On services-testnet
./deploy-sentry.sh

# Or manually
./init-sentry.sh
sudo cp pawd-sentry.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now pawd-sentry
```

### Convert Validator to Use Sentry

```bash
# After sentry is running, update validator config
./enable-sentry-mode.sh val1
```

## Configuration Files

| File | Purpose |
|------|---------|
| `config-sentry.toml` | CometBFT config for sentry node |
| `app-sentry.toml` | Application config for sentry node |
| `config-validator-sentry-mode.toml` | Validator config when using sentries |
| `pawd-sentry.service` | Systemd service file |
| `deploy-sentry.sh` | Automated deployment script |
| `enable-sentry-mode.sh` | Enable sentry mode on validators |

## Security Model

### Sentry Node Settings
- `pex = true` - Discovers peers from network
- `private_peer_ids = "<validator_ids>"` - Never gossip validator addresses
- `unconditional_peer_ids = "<validator_ids>"` - Always maintain validator connection
- `addr_book_strict = false` - Allow private IP connections

### Validator Settings (Sentry Mode)
- `pex = false` - No peer discovery (only connects to sentries)
- `persistent_peers = "<sentry_ids>"` - Connect only to trusted sentries
- P2P port bound to private IP only

## Monitoring

Check sentry health:
```bash
./scripts/testnet/health-sentry.sh
```

View sentry logs:
```bash
sudo journalctl -u pawd-sentry -f
```

## Rollback

To disable sentry mode and return to direct peering:
```bash
./disable-sentry-mode.sh val1
```
