# Explorer Load Testing (k6)

Use this harness to validate API responsiveness and saturation limits for the explorer + API gateway stack (Flask app, nginx, indexer-backed endpoints).

## Prerequisites
- `k6` installed (verified with `k6 version`)
- Explorer stack running locally or at a reachable base URL (nginx/flask front door)

## Quick start (smoke)
```bash
BASE_URL=http://localhost:11080 k6 run explorer/loadtest/explorer-smoke.js
```

## Staged load
- Adjust VUs/stages via the `options` block in `explorer-smoke.js`.
- Override per-endpoint thresholds via `ENDPOINT_WEIGHTS` or `options.thresholds`.

## Suggested cadence
- Smoke: before every deploy (`vus:5`, `duration:30s`)
- Burst: ramp to production-expected RPS (`vus:50`, `duration:2m`, `stages`)
- Soak: `vus:10`, `duration:30m` during staging to catch leaks/timeouts
