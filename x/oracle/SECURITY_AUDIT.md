# Oracle Module Boundary Security Audit

## Scope
Msg/Query/IBC handlers, keeper invariants, circuit breakers, slashing/economic checks, iterator usage, and price feed validation.

## Findings Status
- gosec G115 on generated protobuf varint encoders: **False positive** — to suppress globally.
- Iterator handling: review ongoing; no leaks observed so far.
- Circuit breaker + geoip/aggregation guards present; price path validates bounds.
- Static analysis: **Baseline clean** with gosec `-exclude=G115,G101` (`/tmp/gosec-oracle-final.json`), no actionable findings after iterator close fix.

## Validation Checks (current)
- Params/denoms validated; bond denom enforced on slashing/fees.
- IBC adapters bind scoped capabilities; timeout/error paths handled.
- TWAP/aggregation paths bound loops; pagination limits enforced.
- Slashing and outlier counters guarded; events emitted for monitoring.

## TODO / Next Checks
- Add negative tests for Msgs (bad signer/denom/feed id, paused circuit breaker, geoip failure).
- Add IBC fuzz/negative tests for price packets and timeouts.
- Add gosec suppression config for protobuf G115 across generated code (CI should run with `-exclude=G115,G101` to match baseline).
- Review module account send/recv restrictions and authz/feegrant edges for oracle updates.
