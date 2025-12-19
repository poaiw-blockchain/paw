# PAW Browser Extension Security Audit

## Scope & Methodology

- **Codebase**: `wallet/browser-extension` (popup UI, background worker, packaging scripts)
- **Revision**: `paw-testnet` mainline, audited locally on 2025-12-15
- **Entry Points Reviewed**
  - UI surface (`popup.html`, `styles.css`, `src/popup.js`)
  - Background surface (`src/background.js`) that persists RPC hosts
  - Build/packaging artifacts (`build.js`, `manifest.json`, `dist/`)
  - Release automation (`package.json`, `tools/security-audit.js`)
- **Methodology**
  1. Static review of DOM writes, storage APIs, fetch usage, wallet key flows.
  2. Manifest/permission verification and CSP hardening.
  3. Threat modeling covering wallet secrets, RPC interactions, AI helper secrets.
  4. Automated lint via `npm run security:audit`.
  5. Manual test script exercising privileged flows (wallet import/export, trade submit, AI helper, RPC host override).

## Attack Surface Review

| Surface | Threat | Mitigation |
| --- | --- | --- |
| Popup DOM rendering | DOM-based XSS via order book / RPC error strings | Added template literal sanitizer (`escapeHtml` + `html` tagged template) to every `innerHTML` write (`src/popup.js`) so remote content is encoded before insertion. |
| RPC query construction | Query injection via crafted wallet address | Encode the `message.sender='<addr>'` clause before issuing tx searches (`src/popup.js`). |
| Manifest | Over-privileged APIs, script injection | CSP pinned to `script-src 'self'; object-src 'none'; base-uri 'self'` and permissions restricted to `storage`/`alarms` with localhost host permissions only (`manifest.json`). |
| Local storage | Wallet private keys / AI keys leakage | Existing flows keep secrets in `chrome.storage.local`. Audit verified delete/export confirmations and added documentation for operational hygiene. |
| Supply Chain | Dependency CVEs | `npm run security:audit` runs custom source checks plus `npm audit --audit-level=moderate`. |
| Background host setter | Malicious host persistence | Background worker is limited to `chrome.storage` writes and surfaced hosts are now validated through audits; user still confirms before overrides. |

## Findings & Remediations

1. **DOM Injection Vectors (Fixed)**
   - *Risk*: RPC responses and error messages were interpolated into the DOM with raw template literals.
   - *Fix*: Added `escapeHtml` + `html` tagged templates to sanitize every interpolated value before `innerHTML` assignment (`src/popup.js`). Error fallbacks now use defensive defaults instead of leaking stack traces.
2. **Transaction Search Query Injection (Fixed)**
   - *Risk*: `message.sender='${address}'` was interpolated directly, allowing malformed URLs.
   - *Fix*: Event string is URL encoded before fetching (`src/popup.js`), matching Cosmos REST parser expectations.
3. **Manifest CSP Gap (Fixed)**
   - *Risk*: Lack of `content_security_policy` allowed the default permissive MV3 CSP.
   - *Fix*: Manifest now enforces a strict CSP forbidding `eval`, remote scripts, and object embeds (`manifest.json`).
4. **Missing Automated Security Checks (Fixed)**
   - *Risk*: No extension-specific lint existed beyond `npm audit`.
   - *Fix*: Added `tools/security-audit.js` and wired it into `npm run security:audit` to enforce manifest hygiene and scan for dynamic code execution APIs.

No remaining P1/P2 issues were identified in this pass. See recommendations below for future hardening.

## Automated Audit

Run the combined static + dependency audit:

```bash
cd wallet/browser-extension
npm run security:audit
```

Execution steps:
1. `tools/security-audit.js` validates the CSP, permissions, host scopes, and source code for `eval`, `new Function`, `document.write`, and string-based timers.
2. `npm audit --audit-level=moderate` flags actionable dependency CVEs.

CI integration recommendation: add `security:audit` to the wallet extension pipeline to block regressions before packaging new store builds.

## Manual Validation Checklist

1. **Wallet Secret Handling**
   - Import and export a wallet, ensuring warning dialogs and storage clear operations behave as expected.
   - Trigger `Delete Wallet` to confirm secrets are wiped and status messaging updates accordingly.
2. **RPC Host Override**
   - Change the API host from `http://localhost:1317` to a custom value and confirm both REST (`1317`) and RPC (`26657`) endpoints update.
3. **Trade & History UI**
   - Trigger pool/price refresh with mocked API data containing HTML characters (`<`, `>`, quotes) to confirm output renders escaped text, not markup.
   - Attempt to fetch trade history with a malicious address string (e.g., `paw1';alert(1)//`) and verify the encoded query executes without UI corruption.
4. **AI Assistant Secrets**
   - Run the AI assistant in temporary/session modes, validate that keys are deleted or persisted according to the selected strategy.
5. **Packaging**
   - `npm run build && npm run package` then inspect `dist/manifest.json` to ensure CSP and permissions align with the audit.

## Recommendations & Next Steps

- Add parity E2E tests (Playwright) to assert DOM sanitization of API-driven widgets.
- Expand `tools/security-audit.js` to parse `dist/` bundles for CSP drift before publishing to Chrome/Firefox stores.
- When public sentry hosts exist, replace localhost host permissions with HTTPS endpoints + certificate pinning references.
- Coordinate with centralized security to hook extension releases into the broader bug bounty and chaos-testing plans (see roadmap section 9).
