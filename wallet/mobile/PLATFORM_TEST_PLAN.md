# PAW Mobile Wallet – Platform-Specific Test Plan (iOS & Android)

This plan defines the device matrix, test scenarios, instrumentation steps, and acceptance criteria for pre-store QA. Run the full matrix before every release candidate. Record results in the appended run log.

## Device & OS Matrix
- **iOS**: iPhone 15 Pro (iOS 18.x), iPhone 14 (iOS 17.x), iPhone SE 3rd gen (iOS 17.x, small screen)
- **Android**: Pixel 8 (Android 15), Pixel 6a (Android 14), Samsung Galaxy S21 (Android 13), low-end device (Android 12, 2GB RAM)
- **Orientation**: Portrait + Landscape sanity
- **Network profiles**: WiFi, LTE (simulated throttling 3G/edge via Android dev options or macOS Network Link Conditioner), offline/airplane for storage flows

## Pre-Run Setup
1) `npm ci` (or `npm install`), ensure metro bundler not running from a stale cache.
2) iOS: `cd ios && pod install && cd ..`; Android: accept licenses `yes | sdkmanager --licenses`.
3) Configure API endpoint in `.env.testnet` or via dev settings: `API_URL=https://api.paw-testnet.io`, `RPC_URL=https://rpc1.paw-testnet.io`, `CHAIN_ID=paw-testnet-1`.
4) Enable push notification permissions on devices (required for regression of notification flow).
5) Clear app storage before each scenario run: uninstall/reinstall or Settings → Apps → Storage → Clear.

## Core Test Scenarios (run on every device)
1) **Cold start & biometrics**  
   - Launch from terminated state; ensure splash → unlock flow <3s.  
   - Enroll biometrics; toggle biometric auth on/off; verify lock screen respects device settings.
2) **Wallet lifecycle**  
   - Create wallet (24-word); require confirmation; verify mnemonic never displayed after creation.  
   - Import existing 12/24-word; reject invalid/typo; ensure address matches known test vector.
3) **Send/Receive**  
   - Send PAW to faucet address; confirm fee preview, gas estimate; biometric confirmation works.  
   - Receive via QR; scan from another device; verify balance increment and transaction detail.
4) **Transaction history & pagination**  
   - Pull-to-refresh, infinite scroll; ensure timestamps and amounts match RPC history; offline mode displays cached list with “offline” badge.
5) **DEX interaction**  
   - Execute swap on paw-testnet-1 against a liquid pool; slippage guard enforced; error surfaces for stale quotes.
6) **Staking**  
   - Delegate/undelegate/redelegate to testnet validator; claim rewards; verify updated balances.
7) **Push notifications**  
   - Trigger incoming transfer; ensure notification received while app backgrounded/terminated; deep-link opens transaction detail.
8) **Security & storage**  
   - Background/foreground: confirm app auto-locks per settings; clipboard clear after 60s; no mnemonic in recent apps snapshot.  
   - Root/jailbreak detection sanity: ensure warning surfaces on rooted Android / jailbroken iOS (if emulated).
9) **Offline & error handling**  
   - Airplane mode: app loads cached balances; critical actions disabled with clear messaging.  
   - Corrupt API base URL: user sees actionable error and retry path.
10) **Localization & accessibility (spot checks)**  
    - Dynamic font size respects OS setting; VoiceOver/TalkBack reads primary controls; buttons have accessibility labels.

## Performance & Stability Gates
- Cold start < 3s on modern devices, < 5s on low-end.
- Memory: no growth >150MB after 20 min heavy usage (monitor via Xcode Instruments / Android Studio Profiler).
- CPU spikes recover within 5s after heavy screens (DEX charting, history load).
- No crashes/ANRs in logcat/Xcode device logs.

## Automation Hooks
- Run Jest/unit suite: `npm test`.
- Run detox-lite smoke (if configured) or manual scripted flows; capture video for iOS via `xcrun simctl io booted recordVideo`.
- Push notification mock: use `wallet/mobile/__tests__/PushNotifications.test.js` to validate channel/permission logic before device runs.

## Run Log Template
Record each device execution in `wallet/mobile/TEST_RUN_LOG.md` (create if absent):
```
Date | Device | OS | Network Profile | Scenarios Run | Issues | Result | Tester
```

## Exit Criteria
- All core scenarios pass on each OS family (one modern + one low-end device).  
- No P0/P1 open; P2 mitigations documented; performance gates satisfied.  
- App binaries built in release mode for both platforms and signed with store keystores/profiles.
