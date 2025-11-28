# DEX & Wallet Integration Plan

This document explains how the imported `external/crypto/exchange-frontend` and `browser-wallet-extension` connect to PAW's runtime, along with how the Fernet wallet storage integrates into that experience.

## Frontend wiring

1. Update `exchange-frontend/app.js`'s `API_BASE_URL` to the PAW endpoint (e.g., `https://api.paw.network`).
2. Expose the light-node endpoints by adding a small `config/api.js` module and export it into `app.js` so navigation, Grid data, and WebSocket feeds can be rerouted without manual edits.
3. Use the Fernet wallet helper whenever the frontend signs a transaction or stores metadata (store the password/encryption seed in the mobile wallet's secure storage and hand the serialized payload to `wallet/fernet_storage.encrypt_wallet`).
4. Extend the `browser-wallet-extension` to call `/light-client/checkpoint` and `/light-client/tx-proof` before submitting orders, verifying spontaneity, and caching proofs.

## Wallet UX

- The mobile wallet uses the Fernet helper to store encrypted drafts, receipts, and session states. The UI should prompt for a password, derive the Fernet key with the stored salt, and decrypt via `decrypt_wallet`; the same flows are used for signing with WalletConnect QR flows.
- Light nodes fetch headers via the `/light-client/headers` endpoint described in `external/crypto/docs/LIGHT_CLIENT.md`. Integrate that data into both the extension and the trade frontend so they can show fresh state and validate proofs without downloading blocks.

## Atomic swaps

- Use the frontend's existing buy/sell functions to call the new `POST /atomic-swap/prepare` and `/atomic-swap/commit` endpoints that the PAW controller exposes.
- The extension can act as a bridge for air-gapped devices; it reads QR drafts from `MOBILE_BRIDGE.md` flows, uses Fernet to keep the drafts private, and submits signed payloads over the extension's WebSocket connection once ready.

## Deployment pointers

- Build/deploy the frontend assets via any static host (S3, Vercel). Point the WebSocket and REST hosts to the controller node defined under `infra/node-config.yaml`.
- Run the extension locally by loading `external/crypto/browser-wallet-extension` as an unpacked extension in Chromium.

## Validation

- Smoke test swap flows end-to-end with the controller's sample accounts (comes from `infra/node-config.yaml`).
- Verify wallet storage by running the Fernet helper script (see `wallet/fernet_storage.py` tests, or extend them) and ensuring XOR fallback works on older files if necessary.
