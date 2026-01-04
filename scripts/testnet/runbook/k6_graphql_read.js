import http from 'k6/http';
import { check, sleep } from 'k6';

const graphql = __ENV.GRAPHQL_URL;
if (!graphql) {
  throw new Error('GRAPHQL_URL is required');
}

const profile = __ENV.PROFILE || 'baseline';
const isPeak = profile === 'peak';

const p95Default = isPeak ? 1500 : 800;
const p99Default = isPeak ? 5000 : 3000;
const errDefault = isPeak ? 0.01 : 0.002;

const p95 = __ENV.P95_MS ? parseInt(__ENV.P95_MS, 10) : p95Default;
const p99 = __ENV.P99_MS ? parseInt(__ENV.P99_MS, 10) : p99Default;
const errRate = __ENV.ERROR_RATE ? parseFloat(__ENV.ERROR_RATE) : errDefault;

export const options = {
  vus: __ENV.VUS ? parseInt(__ENV.VUS, 10) : isPeak ? 10 : 5,
  duration: __ENV.DURATION || (isPeak ? '5m' : '2m'),
  rps: __ENV.RPS ? parseInt(__ENV.RPS, 10) : undefined,
  thresholds: {
    http_req_failed: [`rate<${errRate}`],
    http_req_duration: [`p(95)<${p95}`, `p(99)<${p99}`],
  },
};

export default function () {
  const payload = JSON.stringify({ query: '{__typename}' });
  const res = http.post(graphql, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'graphql 200': (r) => r.status === 200 });
  sleep(1);
}

export function handleSummary(data) {
  if (__ENV.CALIBRATE !== '1') {
    return {};
  }
  const p95Val = Math.round(data.metrics.http_req_duration['p(95)']);
  const p99Val = Math.round(data.metrics.http_req_duration['p(99)']);
  const suggestedP95 = Math.round(p95Val * 2);
  const suggestedP99 = Math.round(p99Val * 3.5);
  const err = data.metrics.http_req_failed ? data.metrics.http_req_failed.rate : 0;
  const suggestedErr = Math.max(0.001, Math.min(0.01, err * 2 || 0.002));
  const out = [
    '',
    'Calibration summary:',
    `Observed p95=${p95Val}ms p99=${p99Val}ms error_rate=${(err * 100).toFixed(2)}%`,
    `Suggested thresholds: P95_MS=${suggestedP95} P99_MS=${suggestedP99} ERROR_RATE=${suggestedErr}`,
    '',
  ].join('\n');
  return { stdout: out };
}
