# Authentication Guide

## Overview

PAW Blockchain API uses transaction signing for write operations. Read operations are generally public and don't require authentication.

## Authentication Methods

### 1. Public Endpoints (No Auth Required)

All GET requests for querying blockchain state are public:

```bash
# No authentication needed
curl http://localhost:1317/paw/dex/v1/pools
curl http://localhost:1317/cosmos/bank/v1beta1/balances/paw1abc...
```

### 2. Transaction Signing (Required for Writes)

All state-changing operations require signed transactions:

- Creating pools
- Swapping tokens
- Delegating stake
- Voting on proposals
- Sending tokens

## Transaction Signing Process

### Using CosmJS (JavaScript/TypeScript)

```javascript
import { SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';

const mnemonic = "your 24-word mnemonic here";
const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic);
const [firstAccount] = await wallet.getAccounts();

const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657",
  wallet
);

// Send transaction
const result = await client.sendTokens(
  firstAccount.address,
  "paw1recipient...",
  [{ denom: "uapaw", amount: "1000000" }],
  "auto"
);
```

### Using CLI

```bash
# Sign and broadcast transaction
pawd tx bank send \
  paw1sender... \
  paw1recipient... \
  1000000uapaw \
  --from my-wallet \
  --chain-id paw-mainnet-1 \
  --node http://localhost:26657
```

## API Keys (Optional)

For rate limit increases, you can register for an API key:

```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:1317/paw/dex/v1/pools
```

## Security Best Practices

1. **Never expose private keys** in client-side code
2. **Use environment variables** for sensitive data
3. **Validate all inputs** before signing
4. **Check transaction fees** before broadcasting
5. **Use testnet** for development and testing

## Error Handling

```javascript
try {
  const result = await client.sendTokens(...);
} catch (error) {
  if (error.code === 401) {
    console.error("Authentication failed");
  } else if (error.code === 403) {
    console.error("Insufficient permissions");
  }
}
```

## See Also

- [WebSocket Guide](./websockets.md)
- [Rate Limiting](./rate-limiting.md)
- [Error Codes](./errors.md)
