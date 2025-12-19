# Mobile Hardware (BLE) Plan

Goal: enable Ledger signing on mobile via BLE (react-native-ble-plx) with the same guardrails as desktop/extension.

Steps
- Transport layer: build BLE transport shim that mirrors Ledger HID/WebHID APIs; gate behind feature flag; enforce app open + timeout handling.
- Path/guardrails: reuse path max (0-4), chain-id/fee/Bech32 checks, and device attestation (manufacturer/model).
- Flows: address fetch, MsgSend, staking (delegate/undelegate/redelegate), governance vote, IBC transfer, message sign (auth).
- Error handling: user rejection, timeout, app-not-open, device-lock; map to UI-friendly messages.
- Testing: add mock transport for Jest; manual runbook logs device/app/fw versions, path, flow result, errors.
- Security: disable when rooted/jailbroken flag present; require biometric unlock before initiating BLE session; clear session state on disconnect.
