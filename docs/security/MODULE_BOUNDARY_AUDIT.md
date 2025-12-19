# Module Boundary Audit Status

Progress log for the roadmap "Module Boundary Security Audit" stream. Keep entries concise (<50 lines).

## Completed (Dec 18, 2025)
- Ran interface/input validation sweep across DEX/Compute/Oracle Msgs and IBC packet structs; confirmed address/amount checks and pagination bounds on limit-order iterators (keeper caps at 1k).
- Hardened DEX MsgCreatePool with SDK denom validation on token_a/token_b to block malformed denoms before keeper execution; added regression tests.
- Verified DEX swap deadline guard lives in msg server (blocks stale tx) and keeper-side slippage/flash-loan protections remain intact.
- Confirmed scoped capability usage: compute/dex/oracle keepers initialized with scoped keepers, bind ports in genesis, and channel/port capabilities are reclaimed on restart; module accounts use gov authority and maccPerms block unauthorized sends via bank blocked list.
- Enforced query pagination caps in DEX gRPC handlers (default 100, max 1000) to align with keeper limits and reduce resource exhaustion risk.
- Added pagination caps in Compute/Oracle gRPC handlers (default 100, max 1000) with regression tests to ensure nil/zero limits are bounded; compute covers providers/requests/disputes/evidence/slash/appeals, oracle covers prices/validators.
- IBC allowlists and replay guards: DEX/Oracle use shared packet validator (channel allowlist + nonce/timestamp validation), Compute checks allowlist before parsing and validates nonce/timestamp via keeper; channel open paths enforce expected port/version/ordering (ordered for compute, unordered for dex/oracle) and claim capabilities. Compute caps ack size to 1MB for timeouts.
- Begin/End blocker audit: no O(n) unbounded loops—DEX TWAP update now no-op; DEX end block does bounded iterations (expired orders, matching, circuit breaker recovery, rate-limit cleanup) with error logging instead of panics; Compute blocks reputation updates to 100-block cadence and cleans nonces with per-block window; Oracle amortizes outlier cleanup (50 pairs/block, 100-block cycle) and cleans submissions/slash windows with error logging.
- Resource/gas bounds: Compute enforces evidence size via params (default 10MB) and validation helpers cap command/env/output lengths; escrow paths lock funds via bank transfers with refunds on failure. DEX keeps security parameters hard-coded (MaxPriceDeviation/MaxSwapSizePercent) rather than gov-controlled; economic flows rely on bank module permissions (module mints/burns only). Further proposal/payload sizing remains to be reviewed.
- Governance payload sizing: Compute governance param update CLI accepts max_evidence_size, quorum, slash, etc., but ValidateBasic defers to param validation; no explicit proposal payload size caps observed—recommend enforcing size bounds for text/proposal memos in gov handlers (future work).
- Governance param bounds: MsgUpdateGovernanceParams now validates dispute/appeal params, enforces non-zero/≤50MB max evidence size, and caps quorum/threshold/slash/appeal percentages to [0,1]; tests added.
- Economic flow trace: DEX protocol fees collected under module account and distributed via bank keeper; limit orders escrow/refunds via module account with expiry/timeout cleanup; compute escrows lock funds via module account with atomic refund on failure, release/timeout guarded by keeper flows; mint/burn restricted by maccPerms.
- Payload hardening: MsgSubmitEvidence now rejects payloads above 50MB hard limit at ValidateBasic to bound tx size before hitting keeper param checks.
- Oracle input caps: MsgSubmitPrice now limits asset length (≤128 chars) to prevent oversized identifiers.
- Added negative coverage for compute governance params (quorum >1) and evidence hard cap to guard new validation paths.
- Added fuzz-style payload guard: dispute reasons beyond cap now rejected in tests to bound proposal payload size.
- Added extra fuzz guards for oracle asset length and compute evidence description length to keep payload sizes bounded.
- Observability: added validation-failure events in DEX/Compute/Oracle IBC handlers (unauthorized channel/parse/validate errors) and compute test covering unauthorized IBC packet emits the event.
 - Shared IBC validator now emits unified `ibc_packet_validation_failed` events for auth/data/nonce failures; coverage added in shared ibc tests.
- ICS polish: Compute OnChanOpenTry now validates port ID; shared validator emits events on auth/data/nonce failures; memo cap decorator added (256 bytes).
- ICS compliance sweep: shared channel validator now enforces port parity on `OnChanOpenTry` (affecting DEX/Oracle) to match ICS-004 expectations; checklist captured in `docs/security/ICS_COMPLIANCE_CHECKLIST.md`.
- Access control hardening: Compute dispute/appeal/governance and Oracle param messages now require the governance module authority at ValidateBasic, blocking non-governance senders pre-ante.
- Observability metrics: packet validation failures now increment telemetry counters (port/channel/reason labels) for Prometheus/Grafana dashboards.
- Economic reconciliation: DEX protocol fees accrue in module accounts with no arbitrary mint/burn; compute escrows lock funds with refund/timeout paths; oracle price feeds gate DEX/compute via circuit-breaker fallbacks when deviations or stale data are detected. Slashing/rewards remain under staking/slashing module invariants.
- Observability assets: Added dedicated IBC boundary dashboard (`monitoring/grafana/dashboards/ibc-boundary.json`) to visualize validation failure rates by reason and top offending channels using the new counter.
- Alerting/runbook: Prometheus rule (`monitoring/grafana/alerts/ibc-boundary-alerts.yml`) alarms on 5m increases in validation failures per port/channel/reason; triage steps in `monitoring/runbooks/ibc-boundary-triage.md`.

## Outstanding Focus
- Payload/gas sizing: enforce proposal payload/memo size caps where applicable and ensure gas ceilings on heavy paths; recheck evidence/proposal limits for non-compute modules.
- Observability/tests: fuzz and chaos suites are green (`go test ./tests/fuzz/...`, `go test ./tests/chaos/...`); remaining optional work is targeted negative/fuzz additions for boundary failures and extra metrics/events on circuit-breaker/IBC errors.
