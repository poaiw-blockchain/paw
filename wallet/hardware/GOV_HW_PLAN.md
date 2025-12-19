# Governance Hardware Signing Plan

Scope
- Route governance voting (`MsgVote`, `MsgVoteWeighted`) through hardware (Ledger) in extension/desktop with full guardrails.

Tasks
- Add gov vote helper in extension using shared hardware-first signing pipeline.
- Validation: ensure address prefix `paw`, chain-id match, fee denom `upaw`, gas > 0, proposal id present, option set valid.
- UX: prompt shows proposal id, option, fee, chain-id; warn on weighted votes; allow confirm/reject.
- WalletConnect: reuse hardware helper; block if address mismatch or chain-id/fee invalid.
- Tests: unit test validation; manual device log via `TEST_LOG_TEMPLATE.md`.
