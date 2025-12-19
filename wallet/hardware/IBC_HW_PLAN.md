# IBC + Hardware Signing Plan (Extension/Desktop)

Scope
- Route IBC transfer/signing through hardware (Ledger) in extension/desktop with the same guardrails (chain-id, fee denom, Bech32 prefix, account path).
- Target flows: `MsgTransfer` (ICS20), optional memo, timeout height/timestamp.

Tasks
- Build IBC transfer helper in extension using shared signing pipeline (hardware-first, software fallback).
- Validation: ensure source denom is `upaw`, timeout > current height, prefix `paw` on sender, channel/port present.
- UX: prompt shows channel/port/amount/fee/timeout; warn if memo present.
- WalletConnect: reuse hardware helper; reject unsupported chain-id or denom.
- Tests: add unit test for validation helpers; manual device run per regression plan (log channel/timeout/fee).
