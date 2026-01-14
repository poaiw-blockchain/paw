---
sidebar_position: 1
---

# What is PAW?

PAW is a Cosmos SDK-based Layer 1 blockchain designed for verifiable AI compute with integrated decentralized exchange (DEX) and oracle services.

## Key Features

### DEX Module
- **Automated Market Maker (AMM)**: Constant product formula (x * y = k) for token swaps
- **Liquidity Pools**: Create and provide liquidity to earn trading fees
- **Limit Orders**: Place buy/sell orders with expiration
- **Cross-chain DEX**: IBC integration for multi-chain swaps
- **Advanced Trading**: Slippage protection, multi-hop routing, portfolio management

### Compute Module
- **Job Escrow**: Secure payment system for compute jobs
- **Assignment**: Automated provider matching and job assignment
- **Verification Hooks**: Proof verification for completed work
- **IBC Port**: Cross-chain compute requests

### Oracle Module
- **Price Feeds**: Multi-source price aggregation
- **Voting System**: Decentralized price reporting
- **Slashing Integration**: Penalties for dishonest oracles
- **IBC Queries**: Cross-chain oracle data

## Architecture

PAW is built on:
- **Cosmos SDK v0.50**: Modern blockchain framework
- **CometBFT**: Byzantine Fault Tolerant consensus
- **IBC Protocol**: Inter-Blockchain Communication for cross-chain features
- **Go 1.23+**: High-performance implementation

## Network Specifications

- **Chain ID**: `paw-mvp-1` (testnet), `paw-1` (mainnet)
- **Denomination**: `upaw` (1 PAW = 1,000,000 upaw)
- **Bech32 Prefix**: `paw`
- **Block Time**: ~6 seconds
- **Consensus**: Tendermint (CometBFT)

## Use Cases

1. **Decentralized Trading**: Trade tokens with low fees and no intermediaries
2. **Cross-Chain Swaps**: Execute multi-hop swaps across Cosmos chains
3. **AI Compute Market**: Pay for verifiable AI/ML computation
4. **Oracle Services**: Access reliable price feeds for smart contracts
5. **Liquidity Provision**: Earn fees by providing liquidity to trading pairs

## Getting Started

Ready to start using PAW? Check out our guides:

- [Installation Guide](getting-started/installation.md) - Install the PAW daemon
- [Quick Start](getting-started/quick-start.md) - Run your first node
- [DEX Guide](developers/dex-integration.md) - Start trading on PAW DEX
- [Validator Setup](validators/setup.md) - Run a validator node

## Community

- **Discord**: https://discord.gg/DBHTc2QV
- **Twitter**: https://twitter.com/PAWNetwork
- **GitHub**: https://github.com/poaiw-blockchain/paw

## Next Steps

- Learn how to [install PAW](getting-started/installation.md)
- Explore the [Developer Guides](developers/overview.md)
- Set up a [Validator Node](validators/setup.md)
