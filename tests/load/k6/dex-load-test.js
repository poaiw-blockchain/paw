// K6 Load Test for PAW Chain DEX Module
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const swapSuccessRate = new Rate('swap_success_rate');
const swapDuration = new Trend('swap_duration');
const poolCreationRate = new Rate('pool_creation_success_rate');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 50 },   // Ramp-up to 50 users over 2 minutes
    { duration: '5m', target: 50 },   // Stay at 50 users for 5 minutes
    { duration: '2m', target: 100 },  // Ramp-up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users for 5 minutes
    { duration: '2m', target: 0 },    // Ramp-down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should complete within 2s
    http_req_failed: ['rate<0.05'],    // Error rate should be less than 5%
    swap_success_rate: ['rate>0.95'],  // 95%+ swap success rate
  },
};

// Environment configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:1317';
const LCD_ENDPOINT = `${BASE_URL}`;

// Test data
const POOL_PAIRS = [
  { tokenA: 'upaw', tokenB: 'uusdt' },
  { tokenA: 'upaw', tokenB: 'uatom' },
  { tokenA: 'uusdt', tokenB: 'uatom' },
];

/**
 * Query pool information
 */
function queryPool(poolId) {
  const url = `${LCD_ENDPOINT}/paw/dex/pool/${poolId}`;
  const response = http.get(url);

  check(response, {
    'pool query status is 200': (r) => r.status === 200,
    'pool query response has data': (r) => r.json('pool') !== undefined,
  });

  return response;
}

/**
 * Query all pools
 */
function queryAllPools() {
  const url = `${LCD_ENDPOINT}/paw/dex/pools`;
  const response = http.get(url);

  check(response, {
    'pools query status is 200': (r) => r.status === 200,
    'pools query response is array': (r) => Array.isArray(r.json('pools')),
  });

  return response;
}

/**
 * Simulate swap price calculation
 */
function simulateSwap(poolId, tokenIn, tokenOut, amountIn) {
  const url = `${LCD_ENDPOINT}/paw/dex/simulate-swap`;
  const payload = JSON.stringify({
    pool_id: poolId,
    token_in: tokenIn,
    token_out: tokenOut,
    amount_in: amountIn,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const startTime = Date.now();
  const response = http.post(url, payload, params);
  const duration = Date.now() - startTime;

  const success = check(response, {
    'swap simulation status is 200': (r) => r.status === 200,
    'swap simulation has amount_out': (r) => r.json('amount_out') !== undefined,
  });

  swapSuccessRate.add(success);
  swapDuration.add(duration);

  return response;
}

/**
 * Query pool liquidity
 */
function queryLiquidity(poolId, address) {
  const url = `${LCD_ENDPOINT}/paw/dex/liquidity/${poolId}/${address}`;
  const response = http.get(url);

  check(response, {
    'liquidity query status is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  return response;
}

/**
 * Query pool statistics
 */
function queryPoolStats(poolId) {
  const url = `${LCD_ENDPOINT}/paw/dex/pool/${poolId}/stats`;
  const response = http.get(url);

  check(response, {
    'pool stats query status is 200': (r) => r.status === 200,
  });

  return response;
}

/**
 * Main test scenario
 */
export default function () {
  // Scenario 1: Query all pools
  queryAllPools();
  sleep(1);

  // Scenario 2: Query specific pool (randomly select from 1-10)
  const poolId = Math.floor(Math.random() * 10) + 1;
  queryPool(poolId);
  sleep(1);

  // Scenario 3: Simulate swap
  const pair = POOL_PAIRS[Math.floor(Math.random() * POOL_PAIRS.length)];
  const amountIn = Math.floor(Math.random() * 1000000) + 100000; // Random amount between 100k-1.1M
  simulateSwap(poolId, pair.tokenA, pair.tokenB, amountIn.toString());
  sleep(1);

  // Scenario 4: Query liquidity
  const dummyAddress = 'paw1' + Math.random().toString(36).substring(2, 15);
  queryLiquidity(poolId, dummyAddress);
  sleep(1);

  // Scenario 5: Query pool statistics
  queryPoolStats(poolId);
  sleep(2);
}

/**
 * Setup function - runs once before test
 */
export function setup() {
  console.log('Starting DEX load test...');
  console.log(`Target: ${BASE_URL}`);

  // Health check
  const healthResponse = http.get(`${LCD_ENDPOINT}/cosmos/base/tendermint/v1beta1/node_info`);
  if (healthResponse.status !== 200) {
    throw new Error('Node health check failed');
  }

  console.log('Health check passed!');
}

/**
 * Teardown function - runs once after test
 */
export function teardown(data) {
  console.log('DEX load test completed!');
}
