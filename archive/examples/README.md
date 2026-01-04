# PAW Blockchain Code Examples

Comprehensive code examples for interacting with the PAW blockchain in multiple programming languages.

## Quick Start

Choose your preferred programming language and navigate to the corresponding directory:

- **JavaScript/TypeScript**: `javascript/` - Modern web and Node.js examples
- **Python**: `python/` - Python SDK examples
- **Go**: `go/` - Native Go examples using Cosmos SDK
- **Shell Scripts**: `scripts/` - CLI examples using curl and pawd

## Example Categories

### 1. Basic Examples
Get started with fundamental blockchain operations:
- Connecting to PAW network
- Creating and managing wallets
- Querying account balances
- Sending tokens
- Signing and broadcasting transactions
- Transaction status monitoring

### 2. DEX Examples
Decentralized exchange operations:
- Token swaps
- Adding liquidity to pools
- Removing liquidity
- Creating new trading pairs
- Flash loan operations
- Pool statistics and analysis

### 3. Staking Examples
Proof-of-stake operations:
- Delegating tokens to validators
- Undelegating tokens
- Redelegating between validators
- Claiming staking rewards
- Querying validator information
- Calculating APY/APR

### 4. Governance Examples
On-chain governance:
- Creating governance proposals
- Voting on proposals
- Depositing to proposals
- Querying proposal status
- Monitoring governance parameters

### 5. Advanced Examples
Advanced blockchain interactions:
- Multi-signature transactions
- Batch transaction processing
- WebSocket subscriptions
- Event listening and filtering
- Custom module interactions
- IBC transfers

## Prerequisites

### JavaScript/TypeScript
```bash
cd javascript
npm install
```

### Python
```bash
cd python
pip install -r requirements.txt
```

### Go
```bash
cd go
go mod download
```

### Shell Scripts
- curl
- jq
- pawd CLI

## Configuration

All examples use environment variables for configuration. Create a `.env` file in each language directory:

```env
# Network Configuration
PAW_RPC_ENDPOINT=https://rpc.paw-chain.network
PAW_REST_ENDPOINT=https://api.paw-chain.network
PAW_CHAIN_ID=paw-1

# Wallet Configuration (for testing only - never commit real keys)
MNEMONIC=your test mnemonic here
PRIVATE_KEY=your test private key here

# Optional
GAS_PRICE=0.025upaw
GAS_ADJUSTMENT=1.5
```

## Running Examples

### JavaScript
```bash
cd javascript/basic
node connect.js
```

### Python
```bash
cd python/basic
python connect.py
```

### Go
```bash
cd go/basic
go run connect.go
```

### Shell Scripts
```bash
cd scripts/basic
./connect.sh
```

## Testing

Run all example tests:

```bash
# Test all examples
npm test

# Test specific language
npm test -- --lang=javascript

# Test specific category
npm test -- --category=basic
```

## Documentation

Each example includes:
- Detailed code comments explaining each step
- README with usage instructions
- Sample input/output
- Error handling demonstrations
- Best practices

## Network Endpoints

### Mainnet
- RPC: `https://rpc.paw-chain.network`
- REST: `https://api.paw-chain.network`
- Chain ID: `paw-1`

### Testnet
- RPC: `https://rpc-testnet.paw-chain.network`
- REST: `https://api-testnet.paw-chain.network`
- Chain ID: `paw-testnet-1`

### Local Development
- RPC: `http://localhost:26657`
- REST: `http://localhost:1317`
- Chain ID: `paw-local`

## Support

- Documentation: https://docs.paw-chain.network
- Discord: https://discord.gg/DBHTc2QV
-  Issues: https://github.com/paw-chain/paw/issues

## Security

**WARNING**: The examples use test credentials for demonstration purposes. Never commit real private keys or mnemonics to version control.

## Contributing

Contributions are welcome! Please:
1. Follow the existing code style
2. Add comprehensive comments
3. Include error handling
4. Update the README
5. Add tests

## License

MIT License - see LICENSE file for details

## Example Index

### JavaScript/TypeScript

#### Basic
- [connect.js](javascript/basic/connect.js) - Connect to PAW network
- [create-wallet.js](javascript/basic/create-wallet.js) - Create new wallet
- [query-balance.js](javascript/basic/query-balance.js) - Query account balance
- [send-tokens.js](javascript/basic/send-tokens.js) - Send tokens
- [sign-transaction.js](javascript/basic/sign-transaction.js) - Sign transactions

#### DEX
- [swap-tokens.js](javascript/dex/swap-tokens.js) - Swap tokens
- [add-liquidity.js](javascript/dex/add-liquidity.js) - Add liquidity
- [remove-liquidity.js](javascript/dex/remove-liquidity.js) - Remove liquidity
- [create-pool.js](javascript/dex/create-pool.js) - Create trading pool
- [flash-loan.js](javascript/dex/flash-loan.js) - Execute flash loan

#### Staking
- [delegate.js](javascript/staking/delegate.js) - Delegate tokens
- [undelegate.js](javascript/staking/undelegate.js) - Undelegate tokens
- [redelegate.js](javascript/staking/redelegate.js) - Redelegate tokens
- [claim-rewards.js](javascript/staking/claim-rewards.js) - Claim rewards
- [query-validators.js](javascript/staking/query-validators.js) - Query validators

#### Governance
- [create-proposal.js](javascript/governance/create-proposal.js) - Create proposal
- [vote.js](javascript/governance/vote.js) - Vote on proposal
- [deposit.js](javascript/governance/deposit.js) - Deposit to proposal
- [query-proposals.js](javascript/governance/query-proposals.js) - Query proposals

#### Advanced
- [multisig.js](javascript/advanced/multisig.js) - Multi-sig transactions
- [batch-tx.js](javascript/advanced/batch-tx.js) - Batch transactions
- [websocket.js](javascript/advanced/websocket.js) - WebSocket subscriptions
- [events.js](javascript/advanced/events.js) - Event listening

### Python (Similar structure)
### Go (Similar structure)
### Shell Scripts (Similar structure)
