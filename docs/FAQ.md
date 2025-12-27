# PAW Blockchain FAQ

## General

### What is PAW?
PAW is a Cosmos SDK Layer-1 blockchain purpose-built for verifiable AI compute coordination. It provides on-chain job escrow, worker assignment, result verification, and dispute resolution alongside native liquidity through an integrated DEX.

### What modules does PAW have?
PAW has three core modules:
- **DEX**: Liquidity pools, token swaps, fee collection
- **Compute**: AI job submission, provider selection, result verification
- **Oracle**: Price feeds and data aggregation from validators

### How is PAW different from other Cosmos chains?
PAW integrates compute verification natively rather than relying on external solutions. The DEX provides on-chain liquidity for compute markets, and the oracle ensures accurate pricing for job costs and settlements.

---

## DEX Module

### How do swaps work?
The DEX uses constant-product AMM pools. Swaps execute atomically with slippage protection via `--min-amount-out` and `--slippage-tolerance` flags. Pool drain protection limits single swaps to 30% of reserves.

### What are the fees?
- **Swap fee**: Configurable per-pool (default 0.3%), distributed to liquidity providers
- **Gas fee**: Paid in uPAW (minimum 0.001upaw per gas unit)

### How does the circuit breaker work?
When price deviation exceeds thresholds, the circuit breaker pauses a pool for a configurable duration (default 1 hour). This prevents cascading failures during extreme volatility. Paused pools reject swaps until the timer expires or governance intervenes.

### What is commit-reveal?
An optional MEV protection mechanism. Users first commit a hash of their swap, wait a delay period (default 10 blocks), then reveal and execute. This prevents frontrunning by hiding trade details until execution is imminent.

---

## Oracle Module

### How are prices aggregated?
Validators submit price votes during designated voting windows. The oracle aggregates using weighted median based on validator stake. Outliers beyond configurable deviation thresholds are excluded.

### What prevents oracle manipulation?
- **Weighted median**: Requires controlling 33%+ stake to manipulate
- **Outlier detection**: Extreme values are rejected
- **Slashing**: Dishonest reporters lose stake and reputation
- **Voting windows**: Limited submission periods prevent timing attacks

### How often are prices updated?
Price updates occur each block (~4 seconds) during active voting windows. Finalized prices are available after the aggregation completes at EndBlock.

---

## Compute Module

### How do I submit a compute job?
```bash
pawd tx compute submit-job \
  --input-hash <sha256> \
  --max-gas 1000000 \
  --reward 100000upaw \
  --from mykey
```
The job enters escrow and awaits provider assignment.

### How are providers selected?
Provider selection uses stake-weighted random selection with reputation scoring. Providers must:
- Meet minimum stake requirements
- Have registered signing keys
- Not be jailed or slashed recently

### What happens if a provider fails?
Failed jobs are reassigned to backup providers. The original provider:
- Loses reputation score
- May be slashed for repeated failures
- Gets jailed if below minimum reputation

---

## Development

### How do I run a local node?
```bash
# Initialize
pawd init mynode --chain-id paw-localnet-1

# Create and fund validator key
pawd keys add validator --keyring-backend test
pawd genesis add-genesis-account validator 1000000000000upaw --keyring-backend test
pawd genesis gentx validator 500000000000upaw --chain-id paw-localnet-1 --keyring-backend test
pawd genesis collect-gentxs

# Start
pawd start
```

### How do I connect my dApp?
- **RPC**: `http://localhost:26657` (CometBFT)
- **REST**: `http://localhost:1317` (Cosmos SDK API)
- **gRPC**: `localhost:9090`

Use CosmJS or the PAW SDK for JavaScript/TypeScript integration.

### Where can I get testnet tokens?
- Faucet: https://faucet.testnet.paw.network
- Discord: #testnet-faucet channel
- CLI (if you have funded account):
  ```bash
  pawd tx bank send funded-key <your-address> 10000000upaw --fees 1000upaw
  ```

---

## Quick Reference

| Resource | Location |
|----------|----------|
| API Docs | `docs/api/API_REFERENCE.md` |
| CLI Guide | `docs/CLI_DEX.md` |
| Validator Guide | `docs/guides/VALIDATOR_QUICKSTART.md` |
| SDK Guide | `docs/SDK_DEVELOPER_GUIDE.md` |
| Troubleshooting | `docs/TROUBLESHOOTING.md` |
