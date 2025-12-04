# PAW DEX CLI Quick Reference

## Essential Commands

### Trading
```bash
# Quick swap (auto pool discovery)
pawd tx dex advanced quick-swap [token-in] [amount] [token-out] --from KEY

# Swap with slippage %
pawd tx dex advanced swap-with-slippage [pool-id] [token-in] [amt] [token-out] [slippage-%] --from KEY

# Basic swap
pawd tx dex swap [pool-id] [token-in] [amt-in] [token-out] [min-out] --from KEY
```

### Liquidity
```bash
# Create pool
pawd tx dex create-pool [token-a] [amt-a] [token-b] [amt-b] --from KEY

# Add liquidity (auto-balanced)
pawd tx dex advanced add-liquidity-balanced [pool-id] [total-value] --from KEY

# Remove liquidity
pawd tx dex remove-liquidity [pool-id] [shares] --from KEY
```

### Market Info
```bash
# Market overview
pawd query dex stats overview

# Top pools
pawd query dex stats top-pools --limit 10

# Current price
pawd query dex advanced price [token-in] [token-out]

# Simulate swap
pawd query dex simulate-swap [pool-id] [token-in] [token-out] [amount]
```

### Portfolio
```bash
# Your positions
pawd query dex advanced portfolio $(pawd keys show KEYNAME -a)

# User stats
pawd query dex stats user $(pawd keys show KEYNAME -a)

# Specific pool position
pawd query dex advanced lp-position [pool-id] [address]
```

### Discovery
```bash
# Find pool
pawd query dex pool-by-tokens [token-a] [token-b]

# Token info
pawd query dex stats token [denom]

# All pools
pawd query dex pools --limit 50
```

## Command Categories

**TX: Basic** → create-pool, add-liquidity, remove-liquidity, swap  
**TX: Advanced** → swap-with-slippage, quick-swap, batch-swap, add-liquidity-balanced  
**TX: Orders** → limit-order place/cancel/cancel-all (requires proto)  

**Query: Pools** → pool, pools, pool-by-tokens, liquidity  
**Query: Swaps** → simulate-swap, advanced/price, advanced/route  
**Query: Analytics** → advanced/pool-stats, advanced/portfolio, advanced/arbitrage  
**Query: Stats** → stats/overview, stats/top-pools, stats/user, stats/token  
**Query: Orders** → limit-order, order-book, orders-by-owner, orders-by-pool  

## Flags

**Common TX Flags:**  
`--from` - Signer key  
`--chain-id` - Chain identifier  
`--gas auto` - Auto gas estimation  
`--fees` - Transaction fees  

**Advanced TX Flags:**  
`--deadline` - Time limit (seconds)  
`--expiration` - Order expiration  
`--slippage` - Slippage tolerance %  

**Query Flags:**  
`--output json` - JSON output  
`--limit` - Page size  
`--offset` - Page offset  

## Examples by Use Case

### Provide Liquidity
```bash
# 1. Check existing pools
pawd query dex pools

# 2. Add to pool with auto-balance
pawd tx dex advanced add-liquidity-balanced 1 1000000 --from alice
```

### Execute Trade
```bash
# 1. Check price
pawd query dex advanced price upaw uusdt

# 2. Simulate
pawd query dex simulate-swap 1 upaw uusdt 1000000

# 3. Execute with slippage
pawd tx dex advanced swap-with-slippage 1 upaw 1000000 uusdt 0.5 --from alice
```

### Monitor Market
```bash
# Overview
pawd query dex stats overview

# Best pools
pawd query dex stats top-pools --sort-by apy

# Arbitrage opportunities
pawd query dex advanced arbitrage --min-profit 1.0
```

---
**Build:** `go build -o pawd ./cmd/pawd`  
**Docs:** `docs/CLI_DEX.md`  
**Help:** `pawd tx dex --help`
