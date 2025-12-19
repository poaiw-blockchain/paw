# WalletConnect v2 Hardware Runbook (Extension/Desktop)

## Purpose
- Validate hardware-first signing via WalletConnect v2 in the browser extension (and later desktop).
- Ensure guardrails (chain-id, fee denom, Bech32 prefix, account path) are enforced before device prompts.

## Pre-Flight
- Build extension: `npm run build` (or `npm run watch` during dev).
- Hardware: Ledger Nano S+/X with Cosmos app open; WebHID/WebUSB permitted.
- Config: Set RPC/REST host in popup; connect Ledger via popup control (account 0-4).
- WalletConnect test dApp: use local WC v2 test harness or trusted staging dApp.

## Flow
1) Pairing
   - Start WC session from dApp; extension shows connect prompt with hardware status (expected address/path).
   - Confirm session; deny if origin not allowlisted.
2) Signing (Send/Stake/Gov/IBC)
   - When dApp requests `cosmos_signAmino`, extension checks:
     - Address matches connected hardware address.
     - chain_id matches configured network (reject otherwise).
     - fee denom in `upaw`, gas > 0.
     - Bech32 prefix `paw` on all message addresses.
   - If all pass, route to Ledger via `walletconnect-hw` helper; otherwise show explicit error or fallback prompt (only if user consents).
3) Rejection/Error Paths
   - Verify user rejection surfaces `USER_REJECTED`.
   - App-not-open/device-locked produce clear errors and do not fallback silently.

## Logging
- Use `wallet/hardware/TEST_LOG_TEMPLATE.md` per session:
  - Device/app/fw, transport, path, chain-id, flow, result, errors.
- Capture dApp origin and WC request payload for audit.

## Expected Results
- Hardware-signed tx returned to dApp; broadcast succeeds.
- Mismatched chain-id/fee/prefix/path are blocked with actionable messages.
- Software fallback only after explicit user consent when no hardware connected.
- Audit log records origin/mode/chain/address for the last N events in extension storage; allowlist is enforced and editable via popup.
