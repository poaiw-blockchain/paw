# Network Specifications

Technical specifications for PAW Blockchain networks.

## Mainnet

| Parameter | Value |
|-----------|-------|
| Chain ID | paw-mainnet-1 |
| Launch Date | 2025-01-15 |
| RPC Endpoint | https://rpc.paw.network |
| REST API | https://api.paw.network |
| gRPC | grpc.paw.network:9090 |
| Explorer | https://explorer.paw.network |

## Testnet

| Parameter | Value |
|-----------|-------|
| Chain ID | paw-testnet-1 |
| RPC Endpoint | https://rpc-testnet.paw.network |
| REST API | https://api-testnet.paw.network |
| Faucet | https://faucet.paw.network |

## Network Parameters

### Consensus

- Block Time: 4 seconds
- Block Size: 200KB max
- Gas Limit: 100M per block
- Validators: 100 active
- Consensus: Tendermint BFT

### Staking

- Unbonding Period: 21 days
- Max Validators: 100
- Minimum Self-Delegation: 1000 PAW
- Validator Jail: Downtime >5%, double-sign

### Governance

- Min Deposit: 1,000 PAW
- Deposit Period: 7 days
- Voting Period: 14 days
- Quorum: 40%
- Pass Threshold: 50%
- Veto Threshold: 33.4%

### DEX

- Trading Fee: 0.3%
- Pool Creation Deposit: 100 PAW
- Max Price Impact: 10%
- Min Liquidity: 1,000 USDC equivalent

---

**Previous:** [Tokenomics](/reference/tokenomics) | **Next:** [FAQ](/faq) →
