# WalletConnect v2 + Hardware Passthrough Plan (Extension/Desktop)

Goals
- Let dApps request signatures that route through hardware (Ledger/Trezor) when connected, with explicit UX prompts and safe defaults.
- Preserve software fallback only when no hardware is present; never auto-downgrade without user consent.
- Enforce chain-id/fee/prefix guardrails on all WalletConnect-sourced requests.

Integration Steps
1) Session Handling
   - Expose hardware status (connected address/path/transport) in WalletConnect connect modal.
   - Require explicit confirmation when falling back to software.
   - Persist last hardware session metadata for audit (type, path, transport, timestamps).

2) Request Routing
   - For `cosmos_signAmino` / `cosmos_signDirect`, route to hardware signer when the dApp address matches connected hardware address; otherwise prompt user to switch account or reject.
   - Block unsupported modes (direct-sign on transports that cannot support it) with clear error.
   - Validate chain-id, fee denom (`upaw`), gas > 0, Bech32 prefix `paw` on all messages.

3) UX & Safety
   - Prompt shows dApp origin, requested address, fee/denom/gas, chain-id, and whether hardware will be used.
   - Allowlist/denylist per dApp origin; default denylist for known malicious/test origins.
   - Require confirmation when dApp asks to change chain-id or fee denom.

4) Error Mapping
   - Map hardware errors to user-friendly messages: user rejection, app not open, device locked, path invalid, chain-id mismatch, fee denom rejected.
   - Log error codes for regression (tie into `TEST_LOG_TEMPLATE.md`).

5) Testing
   - Mock transport for CI; fixture requests for amino/direct sign with valid/invalid fees, chain-id mismatch, prefix mismatch.
   - Manual device runbook (tie into `HARDWARE_REGRESSION_PLAN.md`): connect → WC dApp sign (send/stake/gov/IBC) → rejection paths.

6) Security
   - Disable WalletConnect signing when extension is in insecure context or when hardware attestation fails.
   - Rate-limit signing prompts; require recent user interaction for hardware operations.
