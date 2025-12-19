# Mobile BLE Implementation Notes (Ledger)

Transport
- Use `react-native-ble-plx` to discover Ledger devices advertising the Cosmos app service UUID.
- Mirror HID/WebHID APDUs; wrap into a `TransportBLE` class matching LedgerJS transport interface.
- Add connect timeout (60s) and disconnection cleanup; expose manufacturer/model for attestation.

Guardrails
- Reuse core guard helpers (chain-id/fee/prefix/path max) before signing.
- Require biometric unlock before BLE session starts; block on rooted/jailbreak detection.
- Restrict account indices to 0-4; deny non-`upaw` fees; enforce Bech32 `paw`.

Integration Steps
- Add BLE transport module under `wallet/mobile/src/services/hardware/bleTransport.js`.
- Wire into a `LedgerServiceMobile` that implements getAddress/signAmino using BLE transport.
- UI: connect button, account index selector, status/transport indicator.
- Error mapping: user rejection, app-not-open, device-lock, timeout, prefix/fee/chain-id failures.

Testing
- Mock transport for Jest; simulate connect, address fetch, sign success/reject.
- Manual logs via `TEST_LOG_TEMPLATE.md` (device, fw/app, path, flow, error).
- Pair with existing guardrails and regression plan.
