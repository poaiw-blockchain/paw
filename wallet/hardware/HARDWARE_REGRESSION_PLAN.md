# Hardware Wallet Regression Plan (Ledger / Trezor)

## Scope
- Devices: Ledger Nano S Plus / Nano X (WebHID/WebUSB/HID), Trezor One / Model T (Trezor Connect).
- Surfaces: Desktop (Electron, HID/WebHID), Browser Extension (WebHID/WebUSB), Core SDK (Node/web harness), future Mobile (BLE shim via react-native-ble-plx).
- Flows: Address discovery (paths 0-4), send, staking (delegate/undelegate/redelegate), governance vote, IBC transfer, message signing (auth), rejection/timeout/error codes.

## Guardrails to Verify
- Chain-id enforcement matches target network.
- Fee denom restricted to `upaw`; gas > 0.
- Bech32 prefix `paw` enforced on all msg addresses.
- Max account index respected (0-4).
- Device attestation (manufacturer/model) warnings on mismatch.
- Direct-sign blocked for hardware transports not supporting it (Ledger/Trezor via current APIs).

## Test Matrix (per device × surface)
1) **Connect & Enumerate**: Connect, fetch address/path, ensure prefix/gap limit, verify device info.
2) **Send Tx**: Build MsgSend (fee 2500upaw, gas 200000), sign+simulate broadcast; verify signature length >0.
3) **Staking**: Delegate then undelegate, confirm addresses prefixed `paw`.
4) **Governance**: Vote Yes on sample proposal, confirm chain-id guard rejects mismatch.
5) **IBC Transfer**: Transfer minimal amount over configured channel; ensure fee denom guard passes and gas set.
6) **Message Sign**: Sign auth string; confirm rejection surfaces `USER_REJECTED`.
7) **Negative Cases**: Invalid fee denom, zero gas, prefix mismatch, over-max account index, chain-id mismatch.

## Execution Notes
- Desktop: `LedgerService` transport `hid` vs `webhid`, account indices 0-4; require RPC chain-id check before signing.
- Extension: Use `hardware/ledger.js` helpers; WebHID preferred, fallback WebUSB; run Vitest `npm test -- --run src/hardware/ledger.test.js` before manual.
- Core SDK: Run `npm test -- --runTestsByPath src/hardware/__tests__/hardware-wallet.test.ts` for guardrails.
- Record physical runs in `wallet/hardware/test-log-<date>.md` (device, surface, path, result, errors, fw/app versions).

## Mobile (BLE)
- BLE transport (react-native-ble-plx) enforces biometric gate on pairing and signing; same path/fee/chain-id/prefix guardrails.
- Flows wired into UI quick actions (delegate/vote/IBC) and Send screen; uses helpers in `wallet/mobile/src/services/hardware/flows.ts`.
- Automated coverage: `npm test -- --runTestsByPath __tests__/hardware.flows.test.js`.
- Manual: run send/delegate/vote/IBC with a real Ledger; log results in `wallet/hardware/test-log-<date>.md` using `TEST_LOG_TEMPLATE.md`.
