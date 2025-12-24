# MEV Risks and Mitigation Strategies

## What is MEV?

Maximal Extractable Value (MEV) refers to the maximum value that can be extracted from block production by reordering, including, or excluding transactions within a block. In the context of decentralized exchanges (DEXs), MEV manifests primarily through:

- **Front-running**: Validators or bots observe pending swap transactions and submit their own transactions first to profit from known price movements
- **Sandwich attacks**: Attackers place transactions before and after a target swap to extract value through price manipulation
- **Back-running**: Submitting transactions immediately after a target swap to capitalize on price changes

## How MEV Affects DEX Users

When you submit a swap transaction, it enters the mempool where validators and MEV searchers can observe it before inclusion in a block. This visibility allows sophisticated actors to:

1. **Front-run your swap** by executing their own swap first, moving the price against you
2. **Sandwich your swap** by buying before your transaction and selling after, extracting the price impact as profit
3. **Extract all available slippage** by manipulating the price to the exact edge of your slippage tolerance

**Example**: If you set 5% slippage on a large swap, an attacker might manipulate the pool to give you exactly 5% worse price, pocketing the difference as profit.

## Current Protections (Testnet Phase)

The PAW DEX implements several foundational protections:

### 1. Slippage Protection (`min_amount_out`)
Every swap requires a `min_amount_out` parameter that enforces a minimum acceptable output amount. If the actual output would be less due to price movement or manipulation, the transaction fails.

**Best Practice**: Set tight slippage (0.5-1% for liquid pairs, up to 3% for less liquid pairs) to limit MEV extraction.

### 2. Transaction Deadline
Swaps include a `deadline` timestamp. If your transaction sits in the mempool too long and executes after the deadline, it automatically fails, preventing execution at stale prices.

**Best Practice**: Set deadlines 30-120 seconds in the future for most swaps.

### 3. Price Impact Validation
The keeper validates price impact on every swap to prevent single transactions from dramatically moving the market.

### 4. Maximum Pool Drain Limits
Governance-controlled parameter `max_pool_drain_percent` (default: 30%) limits how much liquidity can be extracted in a single swap, preventing extreme manipulation.

### 5. Swap Size Validation
Large swaps relative to pool reserves trigger additional scrutiny and may be rejected if they would enable manipulation.

## Recommended User Strategies

During the testnet phase, users should employ these strategies to minimize MEV exposure:

1. **Use Tight Slippage Tolerances**
   - Liquid pairs (>$1M TVL): 0.5-1% slippage
   - Medium liquidity ($100K-$1M): 1-2% slippage
   - Low liquidity (<$100K): 2-3% slippage
   - Avoid slippage >5% except for extremely illiquid pairs

2. **Split Large Orders**
   - Instead of one large swap, execute multiple smaller swaps
   - Wait a few blocks between swaps to let price stabilize
   - Reduces visibility and price impact

3. **Monitor Mempool Congestion**
   - Higher congestion = more time for MEV extraction
   - Consider waiting for lower activity periods for large swaps

4. **Use Private Transaction Submission** (when available)
   - Some validators may offer private mempools
   - Transactions bypass public mempool, reducing MEV exposure
   - Check with validator operators for availability

5. **Set Appropriate Deadlines**
   - Short deadlines (30-60s) for time-sensitive swaps
   - Longer deadlines (2-5 minutes) acceptable for less urgent swaps
   - Never use deadlines >10 minutes

## Governance Parameters

Network governance controls these MEV-related parameters:

| Parameter | Current Default | Purpose |
|-----------|----------------|---------|
| `recommended_max_slippage` | 3% | Suggested maximum slippage for UI warnings |
| `max_pool_drain_percent` | 30% | Hard limit on single-swap pool drainage |
| `swap_fee` | 0.3% | Total swap fee (reduces MEV profitability) |

Users can check current parameter values via:
```bash
pawd query dex params
```

## Future Mainnet Protections

For mainnet launch, the following additional MEV protections are planned:

### 1. Commit-Reveal Swap Scheme (Implemented, Governance-Controlled)

A two-phase swap mechanism that hides swap details until after commitment:

**Phase 1 - Commit**: Submit a hash of your swap parameters without revealing them
```bash
pawd tx dex commit-swap <hash> --from <address>
```

**Phase 2 - Reveal**: After commit is included (10 blocks), reveal actual swap details
```bash
pawd tx dex reveal-swap <pool-id> <token-in> <token-out> <amount> <min-out> <deadline> <nonce> --from <address>
```

**Benefits**:
- Front-runners cannot see your swap details until after commit
- Sandwich attacks become significantly harder
- Optional: users can choose traditional instant swaps or commit-reveal

**Tradeoffs**:
- Requires two transactions (higher gas cost)
- ~60 second delay (10 blocks) between commit and reveal
- Complexity in UX and wallet integration

### 2. Encrypted Mempool Integration

Future integration with validators offering threshold encryption of pending transactions.

### 3. Batch Auctions

Periodic batch auctions where all swaps within a time window execute at a uniform clearing price, eliminating intra-batch MEV.

### 4. Oracle-Based Price Floors

Integration with external price oracles to validate that executed prices are within reasonable bounds of external markets.

## For Developers

When building applications on PAW DEX:

1. **Always enforce `min_amount_out`** - Never allow users to submit swaps without slippage protection
2. **Provide slippage estimation** - Calculate and display expected price impact
3. **Warn on high slippage** - Alert users when slippage exceeds `recommended_max_slippage` parameter
4. **Set reasonable deadlines** - Default to 2-minute deadlines, allow user customization
5. **Consider commit-reveal for large swaps** - Route swaps >$10K through commit-reveal when enabled
6. **Monitor for sandwich attacks** - Track failed swaps and unusual price movements

## Monitoring and Reporting

The PAW community encourages reporting suspected MEV exploitation:

- **GitHub Issues**: https://github.com/paw-chain/paw/issues
- **Discord**: #security channel
- **Validator Reports**: Contact validator operators directly

Validators engaging in predatory MEV practices may face slashing or exclusion from governance.

## References

- [Flashbots MEV Research](https://docs.flashbots.net/)
- [Ethereum MEV Overview](https://ethereum.org/en/developers/docs/mev/)
- [Cosmos SDK Anti-MEV Discussion](https://forum.cosmos.network/t/mev-in-cosmos/5000)
- [Osmosis DEX MEV Analysis](https://www.osmosis.zone/blog/mev-on-osmosis)

## Changelog

- 2024-12: Initial testnet documentation
- 2024-12: Added commit-reveal implementation for mainnet
