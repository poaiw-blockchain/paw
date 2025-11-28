// K6 Load Test for PAW Chain Compute Module
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const requestQuerySuccessRate = new Rate('request_query_success_rate');
const requestQueryDuration = new Trend('request_query_duration');
const providerQuerySuccessRate = new Rate('provider_query_success_rate');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 10 },   // Ramp-up to 10 users
    { duration: '3m', target: 10 },   // Stay at 10 users
    { duration: '1m', target: 25 },   // Ramp-up to 25 users
    { duration: '3m', target: 25 },   // Stay at 25 users
    { duration: '1m', target: 0 },    // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.05'],
    request_query_success_rate: ['rate>0.95'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:1317';
const LCD_ENDPOINT = `${BASE_URL}`;

/**
 * Query all compute providers
 */
function queryProviders() {
  const url = `${LCD_ENDPOINT}/paw/compute/providers`;
  const response = http.get(url);

  const success = check(response, {
    'providers query status is 200': (r) => r.status === 200,
    'providers response is array': (r) => Array.isArray(r.json('providers')),
  });

  providerQuerySuccessRate.add(success);

  return response;
}

/**
 * Query specific provider
 */
function queryProvider(address) {
  const url = `${LCD_ENDPOINT}/paw/compute/provider/${address}`;
  const response = http.get(url);

  check(response, {
    'provider query status is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  return response;
}

/**
 * Query compute requests
 */
function queryRequests() {
  const url = `${LCD_ENDPOINT}/paw/compute/requests`;
  const response = http.get(url);

  check(response, {
    'requests query status is 200': (r) => r.status === 200,
  });

  return response;
}

/**
 * Query specific compute request
 */
function queryRequest(requestId) {
  const url = `${LCD_ENDPOINT}/paw/compute/request/${requestId}`;
  const startTime = Date.now();
  const response = http.get(url);
  const duration = Date.now() - startTime;

  const success = check(response, {
    'request query status is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  requestQuerySuccessRate.add(success);
  requestQueryDuration.add(duration);

  return response;
}

/**
 * Query module parameters
 */
function queryParams() {
  const url = `${LCD_ENDPOINT}/paw/compute/params`;
  const response = http.get(url);

  check(response, {
    'params query status is 200': (r) => r.status === 200,
    'params has data': (r) => r.json('params') !== undefined,
  });

  return response;
}

/**
 * Main test scenario
 */
export default function () {
  // Scenario 1: Query providers
  queryProviders();
  sleep(1);

  // Scenario 2: Query specific provider (with dummy address)
  const providerAddr = 'paw1' + Math.random().toString(36).substring(2, 15);
  queryProvider(providerAddr);
  sleep(1);

  // Scenario 3: Query all requests
  if (Math.random() < 0.5) {  // 50% of the time
    queryRequests();
    sleep(1);
  }

  // Scenario 4: Query specific request
  const requestId = Math.floor(Math.random() * 1000) + 1;
  queryRequest(requestId);
  sleep(1);

  // Scenario 5: Query module params (less frequent)
  if (Math.random() < 0.2) {  // 20% of the time
    queryParams();
    sleep(1);
  }

  sleep(2);
}

export function setup() {
  console.log('Starting Compute module load test...');
  console.log(`Target: ${BASE_URL}`);

  // Health check
  const healthResponse = http.get(`${LCD_ENDPOINT}/cosmos/base/tendermint/v1beta1/node_info`);
  if (healthResponse.status !== 200) {
    throw new Error('Node health check failed');
  }

  console.log('Health check passed!');
}

export function teardown(data) {
  console.log('Compute load test completed!');
}
