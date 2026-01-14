# PAW Browser Wallet & Miner Extension

This Chrome/Edge/Firefox-compatible extension exposes the mining and wallet-to-wallet trading APIs in a single popup.  
It is intentionally focused on API access (no KYC) and serves as a light WalletConnect-style pane for browser wallets.

## Features

- Configure the API host and wallet address.
- Start/stop the node miner so you can earn rewards directly from the popup.
- Browse current trade orders and matches.
- Submit new orders (sell or buy) that are routed through `POST /wallet-trades/orders`.
- Automatically refresh matches and broadcast events on settlement.

## Local Development & Packaging

1. Install dependencies:
   ```bash
   npm install
   ```
2. Run linting (includes auto-fix) to ensure store-ready JS:
   ```bash
   npm run lint
   ```
3. Build the extension assets into `dist/`:
   ```bash
   npm run build
   ```
4. Produce the distributable archive for store uploads:
   ```bash
   npm run package
   ```
   The zipped artifact is emitted as `wallet/browser-extension/extension.zip` and contains the minified JS/CSS/HTML plus the manifest.

## Installation (Unpacked)

1. In your Chromium-based or Firefox browser open the extensions page (e.g., `chrome://extensions`).
2. Enable developer mode (if required) and choose “Load unpacked”/“Load Temporary Add-on”.
3. Browse to the `dist/` folder that was generated via `npm run build` and load it as the unpacked extension.

For production store submissions follow the end-to-end steps in [`SUBMISSION_GUIDE.md`](./SUBMISSION_GUIDE.md), which captures the Chrome Web Store, Firefox Add-ons, and Microsoft Edge Add-ons Center workflows.

## Security Hardening

- Review the full audit + manual validation checklist in [`SECURITY_AUDIT.md`](./SECURITY_AUDIT.md).
- Run the automated checks (manifest validation + dependency scan) before packaging:
  ```bash
  npm run security:audit
  ```

## WalletConnect UX (Hardware-First)

- Hardware-first signing: Ledger (WebHID/WebUSB) with chain-id/fee/prefix guardrails and approval modal.
- Allowlist configurable in popup; audit log (last 10) stored in extension storage with origin/mode/chain/address.
- Session/sign results forwarded via runtime + content script; dApps can listen on `window`:
  ```js
  window.addEventListener('message', (e) => {
    if (e.data?.type === 'walletconnect-sign-result') {
      console.log('Signed', e.data.result);
    }
    if (e.data?.type === 'walletconnect-session-result') {
      console.log('Session approved?', e.data.result?.approved);
    }
  });
  ```
- DApps can actively request signing/session approval via the injected bridge helpers (event-based, id-correlated):
  ```js
  import { requestWalletConnectSign, requestWalletConnectSession } from './dapp-bridge';

  // Start a session proposal
  await requestWalletConnectSession({ chains: ['paw-mvp-1'], origin: window.location.origin });

  // Trigger a sign (Ledger-first, software fallback)
  const { result } = await requestWalletConnectSign({
    params: [{
      signerAddress: 'paw1...',
      origin: window.location.origin,
      signDoc: { chain_id: 'paw-mvp-1', fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' }, msgs: [], memo: '' }
    }]
  });
  console.log('signature', result.signature);
  ```

## API Notes

- Mining controls call `/mining/start`, `/mining/stop`, and `/mining/status`.
  -- Replace the default API host (`http://localhost:8545`) using the API Host field in the popup if your node runs elsewhere.
  -- The wallet registers a WalletConnect-style session (`/wallet-trades/register`) and signs each order payload with the session secret before posting; if you operate multiple nodes, configure `PAW_WALLET_TRADE_PEER_SECRET/PAW_WALLET_TRADE_PEERS` so they gossip orders via `/wallet-trades/gossip`.
  -- For enhanced security the extension now performs a WalletConnect-style ECDH handshake via `/wallet-trades/wc/handshake` and `/wallet-trades/wc/confirm`, deriving per-session secrets used for signing/encrypted trade payloads.
