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

```
                    ┌─────────────────┐
                    │   Cloudflare    │
                    │   (DDoS/CDN)    │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │     Nginx       │
                    │  (Reverse Proxy)│
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
┌───────▼───────┐   ┌───────▼───────┐   ┌───────▼───────┐
│   RPC (26657) │   │  REST (1317)  │   │  gRPC (9090)  │
└───────────────┘   └───────────────┘   └───────────────┘
```

### Public Endpoints

| Service | URL | Port |
|---------|-----|------|
| RPC | https://testnet-rpc.poaiw.org | 443 |
| REST API | https://testnet-api.poaiw.org | 443 |
| gRPC | testnet-rpc.poaiw.org | 9090 |
| Explorer | https://testnet-explorer.poaiw.org | 443 |
| Faucet | https://testnet-faucet.poaiw.org | 443 |
| Monitoring | https://monitoring.poaiw.org | 443 |
| Snapshots | https://snapshots.poaiw.org | 443 |

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
- **Sentry node architecture** to protect validator nodes from direct exposure
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
- Discord: [Join our community](https://discord.gg/paw)
- GitHub: [poaiw-blockchain/paw](https://github.com/poaiw-blockchain/paw)

## Changelog

| Date | Change |
|------|--------|
| 2025-01-01 | Initial testnet infrastructure deployed |

---

*Last updated: January 2025*
