// K6 Load Test for PAW Chain Oracle Module
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const priceQuerySuccessRate = new Rate('price_query_success_rate');
const priceQueryDuration = new Trend('price_query_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 20 },   // Ramp-up to 20 users
    { duration: '3m', target: 20 },   // Stay at 20 users
    { duration: '1m', target: 50 },   // Ramp-up to 50 users
    { duration: '3m', target: 50 },   // Stay at 50 users
    { duration: '1m', target: 0 },    // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1500'],
    http_req_failed: ['rate<0.05'],
    price_query_success_rate: ['rate>0.95'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:1317';
const LCD_ENDPOINT = `${BASE_URL}`;

// Assets to query
const ASSETS = [
  'BTC/USD',
  'ETH/USD',
  'ATOM/USD',
  'OSMO/USD',
  'PAW/USD',
];

/**
 * Query price for a specific asset
 */
function queryPrice(asset) {
  const url = `${LCD_ENDPOINT}/paw/oracle/price/${asset}`;
  const startTime = Date.now();
  const response = http.get(url);
  const duration = Date.now() - startTime;

  const success = check(response, {
    'price query status is 200': (r) => r.status === 200,
    'price query has price data': (r) => r.json('price') !== undefined,
  });

  priceQuerySuccessRate.add(success);
  priceQueryDuration.add(duration);

  return response;
}

/**
 * Query all prices
 */
function queryAllPrices() {
  const url = `${LCD_ENDPOINT}/paw/oracle/prices`;
  const response = http.get(url);

  check(response, {
    'all prices query status is 200': (r) => r.status === 200,
    'all prices response is array': (r) => Array.isArray(r.json('prices')),
  });

  return response;
}

/**
 * Query oracle validators
 */
function queryOracles() {
  const url = `${LCD_ENDPOINT}/paw/oracle/oracles`;
  const response = http.get(url);

  check(response, {
    'oracles query status is 200': (r) => r.status === 200,
  });

  return response;
}

/**
 * Query price feed for asset (historical data)
 */
function queryPriceFeed(asset) {
  const url = `${LCD_ENDPOINT}/paw/oracle/feed/${asset}`;
  const response = http.get(url);

  check(response, {
    'price feed query status is 200': (r) => r.status === 200,
  });

  return response;
}

/**
 * Main test scenario
 */
export default function () {
  // Scenario 1: Query random asset price
  const asset = ASSETS[Math.floor(Math.random() * ASSETS.length)];
  queryPrice(asset);
  sleep(1);

  // Scenario 2: Query all prices (less frequent)
  if (Math.random() < 0.3) {  // 30% of the time
    queryAllPrices();
    sleep(1);
  }

  // Scenario 3: Query oracles
  if (Math.random() < 0.2) {  // 20% of the time
    queryOracles();
    sleep(1);
  }

  // Scenario 4: Query price feed
  queryPriceFeed(asset);
  sleep(2);
}

export function setup() {
  console.log('Starting Oracle load test...');
  console.log(`Target: ${BASE_URL}`);

  // Health check
  const healthResponse = http.get(`${LCD_ENDPOINT}/cosmos/base/tendermint/v1beta1/node_info`);
  if (healthResponse.status !== 200) {
    throw new Error('Node health check failed');
  }

  console.log('Health check passed!');
}

export function teardown(data) {
  console.log('Oracle load test completed!');
}
