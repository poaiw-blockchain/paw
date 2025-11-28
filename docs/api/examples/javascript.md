# JavaScript/TypeScript API Examples

Complete JavaScript and TypeScript examples for interacting with the PAW Blockchain API.

## Table of Contents
- [Setup](#setup)
- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [DEX Module](#dex-module)
- [Oracle Module](#oracle-module)
- [Compute Module](#compute-module)
- [Bank Module](#bank-module)
- [Staking Module](#staking-module)
- [TypeScript Examples](#typescript-examples)
- [Error Handling](#error-handling)

## Setup

### Node.js

```javascript
const fetch = require('node-fetch');

const API_URL = process.env.API_URL || 'http://localhost:1317';
const MY_ADDRESS = process.env.MY_ADDRESS || 'paw1abc123...';
```

### Browser

```javascript
const API_URL = 'http://localhost:1317';
const MY_ADDRESS = 'paw1abc123...';
```

## Installation

```bash
# Using npm
npm install @cosmjs/stargate @cosmjs/proto-signing

# Using yarn
yarn add @cosmjs/stargate @cosmjs/proto-signing
```

## Basic Usage

### API Client Class

```javascript
class PAWClient {
  constructor(baseURL) {
    this.baseURL = baseURL;
  }

  async get(endpoint) {
    const response = await fetch(`${this.baseURL}${endpoint}`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${await response.text()}`);
    }
    return response.json();
  }

  async post(endpoint, data) {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${await response.text()}`);
    }
    return response.json();
  }
}

// Initialize client
const client = new PAWClient(API_URL);
```

## DEX Module

### List All Pools

```javascript
async function listPools() {
  const response = await client.get('/paw/dex/v1/pools');
  console.log('Pools:', response.pools);
  return response.pools;
}

// With pagination
async function listPoolsPaginated(limit = 10, offset = 0) {
  const response = await client.get(
    `/paw/dex/v1/pools?pagination.limit=${limit}&pagination.offset=${offset}`
  );
  return response;
}
```

### Get Pool by ID

```javascript
async function getPool(poolId) {
  const response = await client.get(`/paw/dex/v1/pools/${poolId}`);
  console.log('Pool:', response.pool);
  return response.pool;
}
```

### Get Pool Price

```javascript
async function getPoolPrice(poolId) {
  const response = await client.get(`/paw/dex/v1/pools/${poolId}/price`);
  console.log(`Price: ${response.price}`);
  return response;
}
```

### Estimate Swap

```javascript
async function estimateSwap(poolId, tokenIn, amountIn) {
  const response = await client.post('/paw/dex/v1/estimate_swap', {
    pool_id: poolId,
    token_in: tokenIn,
    amount_in: amountIn.toString(),
  });

  console.log('Estimated output:', response.amount_out);
  console.log('Price impact:', response.price_impact);
  console.log('Fee:', response.fee);

  return response;
}

// Example usage
estimateSwap(1, 'uapaw', 1000000);
```

### Create Pool

```javascript
async function createPool(creator, tokenA, tokenB, amountA, amountB) {
  const response = await client.post('/paw/dex/v1/create_pool', {
    creator,
    token_a: tokenA,
    token_b: tokenB,
    amount_a: amountA.toString(),
    amount_b: amountB.toString(),
  });

  return response;
}
```

### Swap Tokens

```javascript
async function swap(sender, poolId, tokenIn, amountIn, minAmountOut) {
  const response = await client.post('/paw/dex/v1/swap', {
    sender,
    pool_id: poolId,
    token_in: tokenIn,
    amount_in: amountIn.toString(),
    min_amount_out: minAmountOut.toString(),
  });

  console.log('Swap transaction hash:', response.txhash);
  return response;
}
```

### Add Liquidity

```javascript
async function addLiquidity(sender, poolId, amountA, amountB, minShares) {
  const response = await client.post('/paw/dex/v1/add_liquidity', {
    sender,
    pool_id: poolId,
    amount_a: amountA.toString(),
    amount_b: amountB.toString(),
    min_shares: minShares.toString(),
  });

  return response;
}
```

## Oracle Module

### List Price Feeds

```javascript
async function listPriceFeeds() {
  const response = await client.get('/paw/oracle/v1/prices');
  console.log('Price Feeds:', response.price_feeds);
  return response.price_feeds;
}
```

### Get Specific Price

```javascript
async function getPrice(asset) {
  // URL encode the asset (e.g., "BTC/USD" -> "BTC%2FUSD")
  const encodedAsset = encodeURIComponent(asset);
  const response = await client.get(`/paw/oracle/v1/prices/${encodedAsset}`);

  console.log(`${asset} Price: ${response.price_feed.price}`);
  return response.price_feed;
}

// Example
getPrice('BTC/USD');
```

### Monitor Price Updates

```javascript
async function monitorPrices(assets, intervalMs = 5000) {
  setInterval(async () => {
    for (const asset of assets) {
      try {
        const priceFeed = await getPrice(asset);
        console.log(`[${new Date().toISOString()}] ${asset}: ${priceFeed.price}`);
      } catch (error) {
        console.error(`Error fetching ${asset}:`, error.message);
      }
    }
  }, intervalMs);
}

// Monitor BTC and ETH prices every 5 seconds
monitorPrices(['BTC/USD', 'ETH/USD'], 5000);
```

## Compute Module

### List Tasks

```javascript
async function listTasks(status = null, requester = null) {
  let endpoint = '/paw/compute/v1/tasks';
  const params = [];

  if (status) params.push(`status=${status}`);
  if (requester) params.push(`requester=${requester}`);

  if (params.length > 0) {
    endpoint += '?' + params.join('&');
  }

  const response = await client.get(endpoint);
  return response.tasks;
}

// Examples
listTasks(); // All tasks
listTasks('completed'); // Only completed tasks
listTasks(null, MY_ADDRESS); // My tasks
```

### Get Task by ID

```javascript
async function getTask(taskId) {
  const response = await client.get(`/paw/compute/v1/tasks/${taskId}`);
  return response.task;
}
```

### Submit Compute Task

```javascript
async function submitTask(requester, taskType, taskData, fee) {
  const response = await client.post('/paw/compute/v1/submit_task', {
    requester,
    task_type: taskType,
    task_data: taskData,
    fee,
  });

  console.log('Task submitted:', response.txhash);
  return response;
}

// Example: Submit API call task
submitTask(
  MY_ADDRESS,
  'api_call',
  {
    url: 'https://api.github.com/data',
    method: 'GET',
    headers: {
      'Authorization': 'Bearer token'
    }
  },
  {
    denom: 'uapaw',
    amount: '100000'
  }
);
```

### List Providers

```javascript
async function listProviders() {
  const response = await client.get('/paw/compute/v1/providers');
  return response.providers;
}
```

## Bank Module

### Get Balance

```javascript
async function getBalance(address) {
  const response = await client.get(`/cosmos/bank/v1beta1/balances/${address}`);
  console.log('Balances:', response.balances);
  return response.balances;
}

// Get specific denomination
async function getBalanceByDenom(address, denom) {
  const balances = await getBalance(address);
  const balance = balances.find(b => b.denom === denom);
  return balance ? balance.amount : '0';
}

// Example
const pawBalance = await getBalanceByDenom(MY_ADDRESS, 'uapaw');
console.log(`PAW Balance: ${pawBalance / 1000000} PAW`);
```

### Send Tokens

```javascript
async function sendTokens(fromAddress, toAddress, amount, denom = 'uapaw') {
  const response = await client.post('/cosmos/bank/v1beta1/send', {
    from_address: fromAddress,
    to_address: toAddress,
    amount: [{
      denom,
      amount: amount.toString(),
    }],
  });

  return response;
}
```

## Staking Module

### List Validators

```javascript
async function listValidators(status = null) {
  let endpoint = '/cosmos/staking/v1beta1/validators';
  if (status) {
    endpoint += `?status=${status}`;
  }

  const response = await client.get(endpoint);
  return response.validators;
}

// Get only bonded validators
const activeValidators = await listValidators('BOND_STATUS_BONDED');
```

### Get Validator Details

```javascript
async function getValidator(validatorAddress) {
  const response = await client.get(
    `/cosmos/staking/v1beta1/validators/${validatorAddress}`
  );
  return response.validator;
}
```

### Get Delegations

```javascript
async function getDelegations(delegatorAddress) {
  const response = await client.get(
    `/cosmos/staking/v1beta1/delegations/${delegatorAddress}`
  );
  return response.delegation_responses;
}

// Calculate total staked
async function getTotalStaked(address) {
  const delegations = await getDelegations(address);
  const total = delegations.reduce((sum, del) => {
    return sum + parseInt(del.balance.amount);
  }, 0);
  return total;
}
```

### Delegate Tokens

```javascript
async function delegate(delegatorAddress, validatorAddress, amount, denom = 'uapaw') {
  const response = await client.post('/cosmos/staking/v1beta1/delegate', {
    delegator_address: delegatorAddress,
    validator_address: validatorAddress,
    amount: {
      denom,
      amount: amount.toString(),
    },
  });

  return response;
}
```

## TypeScript Examples

### Type Definitions

```typescript
interface Pool {
  id: number;
  token_a: string;
  token_b: string;
  reserve_a: string;
  reserve_b: string;
  total_shares: string;
  fee_rate: string;
  created_at: string;
}

interface PriceFeed {
  asset: string;
  price: string;
  source: string;
  updated_at: string;
  validators_voted: number;
  total_validators: number;
}

interface Coin {
  denom: string;
  amount: string;
}

interface Validator {
  operator_address: string;
  consensus_pubkey: any;
  jailed: boolean;
  status: string;
  tokens: string;
  delegator_shares: string;
  description: {
    moniker: string;
    identity: string;
    website: string;
    details: string;
  };
  commission: {
    rate: string;
    max_rate: string;
    max_change_rate: string;
  };
}
```

### Typed Client

```typescript
class PAWTypedClient {
  constructor(private baseURL: string) {}

  async get<T>(endpoint: string): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${await response.text()}`);
    }
    return response.json();
  }

  async post<T>(endpoint: string, data: any): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${await response.text()}`);
    }
    return response.json();
  }

  // DEX methods
  async getPools(): Promise<{ pools: Pool[] }> {
    return this.get<{ pools: Pool[] }>('/paw/dex/v1/pools');
  }

  async getPool(id: number): Promise<{ pool: Pool }> {
    return this.get<{ pool: Pool }>(`/paw/dex/v1/pools/${id}`);
  }

  // Oracle methods
  async getPriceFeeds(): Promise<{ price_feeds: PriceFeed[] }> {
    return this.get<{ price_feeds: PriceFeed[] }>('/paw/oracle/v1/prices');
  }

  async getPrice(asset: string): Promise<{ price_feed: PriceFeed }> {
    const encoded = encodeURIComponent(asset);
    return this.get<{ price_feed: PriceFeed }>(`/paw/oracle/v1/prices/${encoded}`);
  }

  // Staking methods
  async getValidators(): Promise<{ validators: Validator[] }> {
    return this.get<{ validators: Validator[] }>('/cosmos/staking/v1beta1/validators');
  }
}

// Usage
const typedClient = new PAWTypedClient('http://localhost:1317');
const { pools } = await typedClient.getPools();
console.log('Pools:', pools);
```

## Error Handling

### Retry Logic

```javascript
async function fetchWithRetry(fn, maxRetries = 3, delay = 1000) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      console.log(`Retry ${i + 1}/${maxRetries} after ${delay}ms...`);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
}

// Usage
const pools = await fetchWithRetry(() => client.get('/paw/dex/v1/pools'));
```

### Rate Limit Handling

```javascript
class RateLimitedClient {
  constructor(baseURL, requestsPerMinute = 100) {
    this.baseURL = baseURL;
    this.requestsPerMinute = requestsPerMinute;
    this.queue = [];
    this.processing = false;
  }

  async request(fn) {
    return new Promise((resolve, reject) => {
      this.queue.push({ fn, resolve, reject });
      this.processQueue();
    });
  }

  async processQueue() {
    if (this.processing || this.queue.length === 0) return;

    this.processing = true;
    const { fn, resolve, reject } = this.queue.shift();

    try {
      const result = await fn();
      resolve(result);
    } catch (error) {
      reject(error);
    }

    // Wait before next request
    const delay = 60000 / this.requestsPerMinute;
    setTimeout(() => {
      this.processing = false;
      this.processQueue();
    }, delay);
  }

  async get(endpoint) {
    return this.request(async () => {
      const response = await fetch(`${this.baseURL}${endpoint}`);
      return response.json();
    });
  }
}
```

## Advanced Examples

### WebSocket Integration

```javascript
class PAWWebSocketClient {
  constructor(wsURL) {
    this.wsURL = wsURL;
    this.ws = null;
    this.subscriptions = new Map();
  }

  connect() {
    this.ws = new WebSocket(this.wsURL);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
    };

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const handlers = this.subscriptions.get(data.type) || [];
      handlers.forEach(handler => handler(data));
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket closed');
      // Reconnect after 5 seconds
      setTimeout(() => this.connect(), 5000);
    };
  }

  subscribe(eventType, handler) {
    if (!this.subscriptions.has(eventType)) {
      this.subscriptions.set(eventType, []);
    }
    this.subscriptions.get(eventType).push(handler);

    // Send subscription request
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        method: 'subscribe',
        params: [eventType]
      }));
    }
  }
}

// Usage
const wsClient = new PAWWebSocketClient('ws://localhost:26657/websocket');
wsClient.connect();
wsClient.subscribe('new_block', (data) => {
  console.log('New block:', data);
});
```

## See Also

- [cURL Examples](./curl.md)
- [Python Examples](./python.md)
- [Go Examples](./go.md)
- [Authentication Guide](../guides/authentication.md)
- [WebSocket Guide](../guides/websockets.md)
