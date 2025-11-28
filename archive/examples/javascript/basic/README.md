# JavaScript Basic Examples

This directory contains basic blockchain interaction examples in JavaScript.

## Prerequisites

```bash
npm install
```

## Environment Setup

Create a `.env` file:

```env
PAW_RPC_ENDPOINT=http://localhost:26657
PAW_REST_ENDPOINT=http://localhost:1317
PAW_CHAIN_ID=paw-local
MNEMONIC="your test mnemonic here"
PAW_ADDRESS=paw1...
```

## Examples

### 1. Connect to Network (`connect.js`)

Connect to the PAW blockchain and retrieve network information.

```bash
node connect.js
```

**What it demonstrates:**
- Connecting to RPC endpoint
- Querying chain ID
- Getting current block height
- Retrieving block information
- Calculating average block time

**Sample Output:**
```
Connecting to PAW Network...
RPC Endpoint: http://localhost:26657

✓ Successfully connected to PAW network

Chain ID: paw-local
Current Block Height: 12345
...
```

### 2. Create Wallet (`create-wallet.js`)

Create a new wallet or import an existing one.

```bash
# Create new wallet
node create-wallet.js

# Import existing wallet
node create-wallet.js import "your mnemonic phrase here"

# Validate mnemonic
node create-wallet.js validate "your mnemonic phrase here"
```

**What it demonstrates:**
- Generating BIP39 mnemonic phrases
- Creating wallets from mnemonics
- Deriving addresses using BIP44
- Validating mnemonic phrases
- Secure key management

**Security Warning:**
Never commit real mnemonics to version control!

### 3. Query Balance (`query-balance.js`)

Query account balances.

```bash
# Query specific address
node query-balance.js paw1abc...xyz

# Query using PAW_ADDRESS from .env
node query-balance.js

# Query specific denomination
node query-balance.js paw1abc...xyz upaw
```

**What it demonstrates:**
- Querying all token balances
- Querying specific denomination
- Formatting token amounts
- Checking account status

### 4. Send Tokens (`send-tokens.js`)

Send tokens to another address.

```bash
node send-tokens.js <recipient> <amount> [denom] [memo]

# Example
node send-tokens.js paw1xyz...abc 1000000 upaw "Hello PAW"
```

**What it demonstrates:**
- Creating and signing transactions
- Broadcasting transactions
- Balance validation
- Gas estimation
- Transaction confirmation

**Requirements:**
- MNEMONIC must be set in .env
- Sufficient balance for amount + fees

## Error Handling

All examples include comprehensive error handling:

- Connection errors
- Invalid addresses
- Insufficient balance
- Network timeouts
- Transaction failures

## Testing

Run tests for basic examples:

```bash
npm test -- --testPathPattern=basic
```

## Common Issues

### Connection Refused
- Ensure the node is running
- Check the RPC endpoint URL
- Verify firewall settings

### Invalid Mnemonic
- Check for typos in the mnemonic
- Ensure 24 words (or 12 for some wallets)
- Validate using `create-wallet.js validate`

### Insufficient Balance
- Check balance with `query-balance.js`
- Request tokens from faucet
- Ensure enough for amount + fees

## Next Steps

After mastering basic examples, explore:
- [DEX Examples](../dex/) - Token swapping and liquidity
- [Staking Examples](../staking/) - Delegating and rewards
- [Governance Examples](../governance/) - Proposals and voting
- [Advanced Examples](../advanced/) - WebSockets and batch operations
