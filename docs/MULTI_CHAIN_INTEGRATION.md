# Multi-Chain Integration Guide

## Overview

PAW is the DEX hub for interoperability with external chains such as Aura and XAI:
- **Native IBC** with full transfer and DEX packet support
- **Cross-chain swaps** via PAW DEX
- **Unified wallet** experience across compatible chains

## Scope (Cross-Chain Only)

This guide documents cross-chain compatibility. PAW’s repo does not include other chains’ code or infra. The shared multi-chain wallet lives in `~/blockchain-projects/shared/wallet/multi-chain-wallet/`.

## Chain Architecture

| Chain | Type | Coin Type | Prefix | Features |
|-------|------|-----------|--------|----------|
| Aura | Cosmos SDK | 118 | `aura` | IBC, CosmWasm |
| PAW | Cosmos SDK | 118 | `paw` | IBC, DEX |
| XAI | EVM-compatible | 22593 | `xai` | AI Trading |

## PAW as DEX Hub

PAW provides cross-chain liquidity:

```
┌─────────────────────────────────────────┐
│              PAW DEX Hub                │
├─────────────────────────────────────────┤
│  AURA/PAW Pool ←→ PAW/XAI Pool         │
│       ↑                  ↑              │
│      IBC              Axelar            │
│       ↓                  ↓              │
│    Aura Chain        XAI Chain          │
└─────────────────────────────────────────┘
```

## IBC Integration

### Supported IBC Ports

| Port | Purpose |
|------|---------|
| `transfer` | Token transfers |
| `dex` | Cross-chain swaps |

### IBC Channel Configuration

PAW's Hermes relayer config includes Aura chain:

```toml
[[chains]]
id = 'aura-local-4'
account_prefix = 'aura'
gas_price = { price = 0.025, denom = 'uaura' }
```

### Channel Authorization

PAW uses whitelist-based channel authorization (SEC-10):
- `app/ibcutil/channel_authorization.go`
- Only authorized channels can send packets
- Governance can update authorized list

## Unified Wallet

The shared wallet library (`~/blockchain-projects/shared/wallet/multi-chain-wallet/`) enables:
- Single mnemonic for all chains
- Linked addresses for Aura and PAW
- IBC transfer message building

### Address Linking

Aura and PAW share coin type 118:
```typescript
const wallet = await MultiChainWallet.fromMnemonic(mnemonic);
const linked = await wallet.getLinkedCosmosAddresses();
// { aura: 'aura1abc...', paw: 'paw1abc...' }
// Same public key, different prefixes
```

## DEX IBC Packets

PAW DEX supports cross-chain packets:

| Packet Type | Purpose |
|-------------|---------|
| `query_pools` | Query remote liquidity |
| `execute_swap` | Execute swap on remote chain |
| `cross_chain_swap` | Multi-hop swap |
| `pool_update` | Sync pool state |

## Testing

```bash
cd ~/blockchain-projects/shared/wallet/multi-chain-wallet
npm test
```

All tests verify multi-chain wallet functionality including IBC message building.
