# WalletConnect UI Integration Plan (Extension)

Objectives
- Present clear prompts for WC signing with hardware/software route selection.
- Surface dApp origin, requested address, chain-id, fee/denom/gas, message summary, and hardware status.

UI/UX Steps
1) Session Prompt
   - Show dApp origin, requested address, and connected hardware address (if any).
   - Require explicit user confirmation to pair; block non-allowlisted origins.
2) Signing Prompt
   - Display tx summary (type, amount, fee, gas, chain-id).
   - Show whether hardware will be used; allow switch to software only with confirmation.
   - Provide reject button; map hardware errors to user-friendly messages.
3) Result Handling
   - Show success hash or error; log WC request id + mode (hardware/software) for audit.
4) Background→Popup wiring
   - background.js listens for WC messages and forwards `{type: 'walletconnect-sign', request}` to popup
   - popup receives, renders prompt, calls `handleWalletConnectSignRequest`, returns signature/publicKey or error
5) Audit Log
   - Store last N sign events in local storage (id, origin, mode, chain-id, address, timestamp)
   - Provide UI buttons to view/clear audit log for operators

Technical Hooks
- Background script listens for WC messages and forwards to popup via `chrome.runtime.sendMessage`.
- Popup invokes `handleWalletConnectSignRequest` and returns signature/publicKey or error.
- Maintain in-memory WC session state (origin, address, mode) and persist minimal audit trail.
- Allowlist editable in popup; defaults to `https://trusted-dapp.example` (persisted in storage).

Testing
- Add mocked WC request tests to ensure prompts surface correct data.
- Manual run per `WALLETCONNECT_HW_RUNBOOK.md` with device + origin logging.
