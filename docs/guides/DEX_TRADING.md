# PAW DEX Trading Guide

Hands-on reference for creating pools, managing liquidity, and executing swaps on the PAW AMM. Examples use the CLI (`pawd`) with the default keyring backend. Replace placeholder values with your own pool IDs and addresses.

## 1. Prerequisites
- `pawd` binary built from the same commit that produced the chain binary
- Funded account with sufficient `upaw` plus the tokens you plan to pair
- Access to a node RPC (local: `tcp://localhost:26657`)
- Keyring entry (e.g. `trader`) unlocked on the machine running the CLI

Check your balances first:
```bash
pawd query bank balances $(pawd keys show trader -a)
```

## 2. Create a Liquidity Pool
Pools require symmetric value in both tokens. The creator receives pool shares representing their liquidity.
```bash
pawd tx dex create-pool upaw 1000000uatom \
  --amount-in "1000000upaw,1000000uatom" \
  --from trader --chain-id paw-testnet-1 \
  --gas auto --gas-adjustment 1.5 --fees 5000upaw
```
Flags:
- `--amount-in` lists both tokens with the initial deposit (must match positional args)
- Use `--pool-type constant-product` (default) or specify alternative curves once enabled

After the transaction is included, query the pool:
```bash
pawd query dex pools
pawd query dex pool <pool-id>
```

## 3. Add Liquidity
```bash
pawd tx dex add-liquidity <pool-id> \
  --amount-in "500000upaw,250000uatom" \
  --min-shares 1 \
  --from trader --gas auto --fees 4000upaw
```
Tips:
- Deposit ratios must match the current pool price; the CLI auto-adjusts by refunding excess.
- `--min-shares` protects against slippage while minting pool shares.

### Remove Liquidity
```bash
pawd tx dex remove-liquidity <pool-id> \
  --shares 10000 \
  --min-amount-out "0upaw,0uatom" \
  --from trader
```
The `--min-amount-out` vector prevents receiving fewer tokens than expected.

## 4. Execute Swaps
Simulate first:
```bash
pawd query dex simulate-swap <pool-id> upaw 250000uatom
```
Then submit:
```bash
pawd tx dex swap <pool-id> upaw 250000uatom uatom \
  --min-amount-out 123456uatom \
  --slippage-tolerance 0.005 \
  --from trader --fees 5000upaw
```
Key parameters:
- `--slippage-tolerance` (0.5% above) rejects trades if price moves too far mid-block.
- `--min-amount-out` is enforced even if you omit `--slippage-tolerance`.

## 5. Fees & Rewards
- **Swap fee**: collected per trade and routed to the fee collector module, then distributed to staking validators per protocol params.
- **Liquidity incentives**: pools can emit additional rewards through governance. Monitor `pawd query dex pool-rewards <pool-id>` when available.
- **Gas fees**: pay in `upaw` when broadcasting any transaction.

## 6. Risk Management
- **Impermanent loss**: providing liquidity exposes you to divergence between token prices.
- **Pool drain protection**: protocol enforces a configurable max output per swap (currently 30%). Attempting to withdraw >30% of a reserve reverts.
- **Front-running/MEV**: always set slippage bounds and consider splitting large swaps to avoid sandwich attacks.
- **Smart contract upgrades**: watch governance proposals; upgraded pool logic may require re-adding liquidity.

## 7. Troubleshooting
| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| `insufficient funds` | Key lacks one of the tokens | `pawd tx bank send` from a funded wallet or faucet |
| `pool not found` | Wrong ID | List pools and confirm ID before retrying |
| Swap rejected with `max pool drain exceeded` | Trade tries to take > allowed percentage | Break into smaller swaps or wait for governance change |
| `slippage exceeded` | Price moved before inclusion | Increase tolerance slightly or re-broadcast with updated quote |

## 8. Helpful Queries
```bash
pawd query dex pools --limit 50
pawd query dex pool-by-tokens upaw uatom
pawd query dex liquidity <pool-id> $(pawd keys show trader -a)
pawd query dex params
```

Stay synced with the validator and governance channels to learn about new pools, liquidity mining campaigns, or parameter upgrades.
