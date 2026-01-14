# PAW Network Infrastructure

This document provides transparency into the infrastructure supporting the PAW blockchain network. We believe in open communication with our community about how the network is operated and secured.

## Philosophy

PAW follows blockchain community best practices for infrastructure:

- **Dedicated bare-metal servers** - No shared virtualization or cloud instances for validator nodes
- **Geographic distribution** - Nodes distributed across multiple data centers and regions
- **Independent operation** - Each blockchain in our ecosystem runs on its own dedicated infrastructure
- **Transparency** - Open documentation of our infrastructure choices and security practices

## Current Testnet Infrastructure

### Validator Node

| Specification | Details |
|--------------|---------|
| **Server Type** | Dedicated bare-metal server |
| **Provider** | OVH (SoYouStart) |
| **Location** | Beauharnois, Quebec, Canada (BHS) |
| **CPU** | Intel Xeon E3-1270 v6 @ 3.80GHz (4 cores / 8 threads) |
| **Memory** | 64 GB DDR4 ECC |
| **Storage** | 2x 480GB SSD (RAID 1) |
| **Network** | 500 Mbps dedicated bandwidth |
| **IPv4** | Dedicated static IP |

### Why Bare-Metal?

We chose dedicated bare-metal servers over cloud instances for several reasons:

1. **Predictable Performance** - No noisy neighbor issues or resource contention
2. **Security** - Full control over the hardware stack with no hypervisor layer
3. **Cost Efficiency** - Better price-to-performance for long-running blockchain nodes
4. **Community Trust** - Bare-metal is the accepted standard for serious blockchain infrastructure

### Network Architecture

PAW testnet uses a **sentry node architecture** to protect validators from direct public exposure:

```
                              Internet
                                 │
                    ┌────────────┴────────────┐
                    │       Cloudflare        │
                    │    (DDoS Protection)    │
                    └────────────┬────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
       ┌──────▼──────┐    ┌──────▼──────┐   ┌──────▼──────┐
       │  Sentry 1   │    │  Sentry 2   │   │   Nginx     │
       │ paw-testnet │    │  services-  │   │  (RPC/API)  │
       │ P2P:12056   │    │   testnet   │   └──────┬──────┘
       └──────┬──────┘    │ P2P:12056   │          │
              │           └──────┬──────┘          │
              │                  │                 │
              └────────┬─────────┘                 │
                       │                          │
         ┌─────────────┼─────────────┐            │
         │     WireGuard VPN        │◄───────────┘
         │      (10.10.0.x)         │
         └─────────────┬─────────────┘
                       │
    ┌──────────────────┼──────────────────┐
    │                  │                  │
    ▼                  ▼                  ▼
┌─────────┐      ┌─────────┐      ┌─────────────────┐
│ Val 1&2 │      │ Val 3&4 │      │  Community      │
│paw-test │◄────►│services │      │  Validators     │
│   net   │ VPN  │ testnet │      │  (via sentries) │
└─────────┘      └─────────┘      └─────────────────┘
```

**Node Details:**
| Node | Server | P2P Port | RPC Port | Node ID |
|------|--------|----------|----------|---------|
| val1 | paw-testnet | 11656 | 11657 | `945dfd111e231525f722a32d24de0da28dade0e8` |
| val2 | paw-testnet | 11756 | 11757 | `35c1a40debd4a455a37a56cee7adbaaffb0778f8` |
| val3 | services | 11856 | 11857 | `a2b9ab78b0be7f006466131b44ede9a02fc140c4` |
| val4 | services | 11956 | 11957 | `f8187d5bafe58b78b00d73b0563b65ad8c0d5fda` |
| sentry1 | paw-testnet | 12056 | 12057 | `38510c172e324f25e6fe8d9938d713bcaed924af` |
| sentry2 | services | 12056 | 12057 | `ce6afbda0a4443139ad14d2b856cca586161f00d` |

**Key Security Features:**
- Validators are hidden behind sentry nodes (no public P2P ports exposed)
- All public RPC/REST/gRPC traffic routes through sentries
- Sentry nodes use `private_peer_ids` to never gossip validator addresses
- VPN-only communication between validators (10.10.0.x network)

### Public Endpoints

| Service | URL | Port |
|---------|-----|------|
| RPC | https://testnet-rpc.poaiw.org | 443 |
| REST API | https://testnet-api.poaiw.org | 443 |
| gRPC | testnet-rpc.poaiw.org | 9091 |
| Explorer | https://testnet-explorer.poaiw.org | 443 |
| Faucet | https://testnet-faucet.poaiw.org | 443 |
| Monitoring | https://monitoring.poaiw.org | 443 |
| Snapshots | https://snapshots.poaiw.org | 443 |

Note: REST endpoints may be degraded; see `docs/TESTNET_ENDPOINTS.md` for current status.

### Security Measures

- **Firewall** - UFW with strict ingress rules; only required ports exposed
- **SSH** - Key-based authentication only, no password login
- **TLS** - All public endpoints secured with Let's Encrypt certificates
- **Updates** - Automated security updates via unattended-upgrades
- **Monitoring** - 24/7 monitoring via Grafana with alerting

## Cosmos SDK Infrastructure

As a Cosmos SDK-based blockchain, PAW follows the standard Tendermint/CometBFT architecture:

### Node Types

| Type | Purpose | Exposure |
|------|---------|----------|
| **Validator** | Block production and consensus | Private (sentry-protected) |
| **Sentry** | Public-facing relay nodes | Public P2P |
| **Archive** | Full historical state | Public RPC/API |

### IBC Relayer Infrastructure

For Inter-Blockchain Communication (IBC), we maintain:

- Dedicated relayer nodes for cross-chain message passing
- Connections to Cosmos Hub and other IBC-enabled chains
- Redundant relayer instances for high availability

## Mainnet Plans

For mainnet launch, we plan to expand infrastructure with:

- **Multiple validator nodes** across different geographic regions (NA, EU, Asia)
- **Multiple sentry nodes** for geographic redundancy (already implemented on testnet)
- **Multiple hosting providers** to avoid single provider dependency
- **Hardware Security Modules (HSM)** for validator key protection
- **Independent snapshot providers** for quick node bootstrapping
- **Dedicated IBC relayer infrastructure** with geographic redundancy

## Running Your Own Node

We encourage community members to run their own nodes. See our documentation:

- [Validator Quick Start](docs/VALIDATOR_QUICK_START.md)
- [Validator Hardware Requirements](docs/VALIDATOR_HARDWARE_REQUIREMENTS.md)
- [Validator Operator Guide](docs/VALIDATOR_OPERATOR_GUIDE.md)
- [Sentry Architecture](docs/SENTRY_ARCHITECTURE.md)

### Minimum Requirements for Validators

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 4 cores | 8+ cores |
| Memory | 16 GB | 32+ GB |
| Storage | 500 GB SSD | 1 TB NVMe |
| Network | 100 Mbps | 1 Gbps |

### Compute Provider Requirements

For running AI compute workloads on PAW:

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 8 cores | 16+ cores |
| Memory | 32 GB | 64+ GB |
| GPU | Optional | NVIDIA RTX 3080+ |
| Storage | 1 TB NVMe | 2+ TB NVMe |
| Network | 1 Gbps | 10 Gbps |

## Contact

For infrastructure-related inquiries:
- Email: info@poaiw.org
- Discord: [Join our community](https://discord.gg/DBHTc2QV)
- GitHub: [poaiw-blockchain/paw](https://github.com/poaiw-blockchain/paw)

## Changelog

| Date | Change |
|------|--------|
| 2026-01-13 | Deployed sentry node architecture for validator protection |
| 2025-01-01 | Initial testnet infrastructure deployed |

---

*Last updated: January 2026*
