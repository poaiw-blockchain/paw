# DEX Module Boundary Security Audit

## Scope
Evaluates the DEX module’s boundary surfaces: Msg/Query handlers, keeper invariants, IBC adapters, circuit breakers, authz/fee flows, and iterator/gas usage.

## Findings Status
- Iterator lifecycle: **Fixed** — added safe closes in pagination and rate-limit cleanup (`keeper/limit_orders.go`, `keeper/abci.go`).
- Protobuf varint G115 (gosec): **False positive** — generated code; to be suppressed centrally.
- Hardcoded sim weights G101: **Noise** — constants for simop weights; safe to ignore/suppress.
- Static analysis: **Baseline clean** with gosec `-exclude=G115,G101` (`/tmp/gosec-dex-final.json`), no actionable findings after iterator fixes.
- MsgCreatePool denom validation: **Added** — SDK denom validation on token_a/token_b with regression tests to block malformed denoms pre-keeper.
- Pagination hardening: **Added** — gRPC queries now enforce default/max limits (100/1000) across pools/limit-order listings to align with keeper caps and guard against DoS.

## Validation Checks (current)
- Pagination limits bounded (≤1000) for owner/pool order listings.
- Rate-limit cleanup window bounded (24h, capped loop).
- Circuit breaker hooks present; pool/denom checks enforced in keeper.
- IBC adapters use scoped capabilities and safe timeouts.
- Fees/denoms validated against bond denom; pool auth paths respect module accounts.

## TODO / Next Checks
- Add negative tests for Msgs: invalid signer/denom/pool state, paused/circuit-breaker active cases.
- Add IBC fuzz/negative tests for aggregation/timeout paths.
- Add gosec suppression config for protobuf varint G115 across generated code (CI should run with `-exclude=G115,G101` to match baseline).
- Add authz/feegrant boundary tests for swap/liquidity msgs to ensure no privilege escalation.
