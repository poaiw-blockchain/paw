import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:11080';

const ENDPOINTS = [
  '/',
  '/health',
  '/health/ready',
  '/api/v1/blocks',
  '/api/v1/transactions',
  '/api/v1/stats',
  '/dex',
  '/oracle',
  '/compute',
  '/search?q=paw',
];

export const options = {
  vus: __ENV.VUS ? parseInt(__ENV.VUS, 10) : 10,
  duration: __ENV.DURATION || '30s',
  thresholds: {
    http_req_duration: ['p(95)<800'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  ENDPOINTS.forEach((path) => {
    const res = http.get(`${BASE_URL}${path}`);
    check(res, {
      'status is 2xx/3xx': (r) => r.status >= 200 && r.status < 400,
      'p95 under 800ms': (r) => r.timings.duration < 800,
    });
    sleep(0.5);
  });
}
