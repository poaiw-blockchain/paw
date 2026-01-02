# Mobile Wallet Onboarding (iOS & Android)

Step-by-step onboarding for end-users and QA, including platform compatibility, permissions, and download channels (store/TestFlight + direct builds).

## Download Links
- **Android (Play Beta/Internal)**: `https://play.google.com/store/apps/details?id=io.paw.wallet` (use Play internal/beta track while testnet is active).
- **Android direct sideload**: build via `npm run android:release` (artifact: `android/app/build/outputs/apk/release/app-release.apk`).
- **iOS TestFlight**: public invite link once uploaded: `https://testflight.apple.com/join/pawwallet-beta` (update if App Store Connect assigns a new code).
- **iOS sideload (QA only)**: `npm run ios:archive` then install via Xcode Devices/Simulator.

## Compatibility Matrix
| Device | OS | Biometrics | BLE (Ledger) | Status |
|--------|----|------------|--------------|--------|
| iPhone 12/13/14 | iOS 16–18 | Face ID | Yes | ✅ Full support |
| iPhone 8/SE (2nd gen) | iOS 16–17 | Touch ID | Yes | ✅ Full support |
| iPhone 7 and older | iOS 15 or lower | Touch ID | Limited | ⚠️ BLE unstable; prefer software signing |
| Pixel 6/7 | Android 13–14 | Fingerprint | Yes | ✅ Full support |
| Samsung S22/S23 | Android 13–14 | Fingerprint | Yes | ✅ Full support |
| Budget Android (no BLE) | Android 12–13 | Fingerprint (varies) | No | ⚠️ Hardware signing unavailable; software keys only |

## Required Permissions
- **Camera**: QR scanning for receive/onboarding; grant on first scan.
- **Bluetooth**: Required for Ledger BLE signing; enable and keep Ledger app open during pairing.
- **Biometrics**: Used for wallet unlock and signing confirmation; enable in Settings → Security.
- **Notifications**: Optional; required for staking/bridge alerts.

## First-Run Flow
1. Install from the link above (Play/TestFlight) and launch the app.
2. Choose network: select **paw-testnet-1** (default). Mainnet will surface as **paw-mainnet-1** on release.
3. Create or import wallet:
   - New wallet: generate 24-word mnemonic → confirm words → set passphrase.
   - Import: paste mnemonic → optional passphrase.
4. Enable security:
   - Turn on biometrics (Face ID/Touch ID/fingerprint).
   - Set auto-lock to 30 seconds or lower for shared devices.
5. (Optional) Pair Ledger:
   - Open Cosmos app on Ledger, enable Bluetooth.
   - In Settings → Hardware Wallet → “Pair Ledger (BLE)”.
   - Approve pairing request; BLE prompt will appear before signing.
6. Request funds:
   - Use faucet: `https://faucet.paw-testnet.io` or CLI `./scripts/faucet.sh --check https://rpc1.paw-testnet.io <address> 1000000upaw`.
7. Run a smoke action:
   - Send a 1upaw transfer to yourself.
   - View balance and history to confirm RPC connectivity.

## Troubleshooting
- **Stuck at “Connecting”**: confirm RPC in Settings → Network is `https://api.paw-testnet.io` (or your light RPC), toggle airplane mode, then reopen.
- **BLE pairing fails**: reboot Ledger, ensure Bluetooth is enabled system-wide, and stay within 2m of the device. Remove stale pairing in OS settings and retry.
- **Biometric prompt missing**: re-enable biometrics inside OS settings, then toggle the in-app Security switch off/on.
- **Low disk devices**: enable “Light mode (state sync)” in Settings → Advanced; this matches the pruning/state-sync profile documented in `docs/guides/onboarding/LIGHT_CLIENT_PROFILE.md`.

## Support & Reporting
- Status page: `https://status.paw-testnet.io` (RPC/REST/gRPC/Explorer/Faucet health).
- Submit issues: GitHub `paw-chain/paw` (tag `mobile`).
- Security reports: security@paw.network.
