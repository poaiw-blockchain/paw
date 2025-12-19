# Hardware Services (Mobile)

Files:
- `bleTransport.ts`: Ledger BLE transport (react-native-ble-plx) implementing Ledger transport interface (scan/connect/exchange/close).
- `bleTransport.js`: JS shim that re-exports the TS transport for environments that still import `.js`.

Plans/Runbooks:
- `wallet/hardware/MOBILE_BLE_PLAN.md`
- `wallet/hardware/MOBILE_BLE_IMPL_NOTES.md`
- `wallet/hardware/HARDWARE_REGRESSION_PLAN.md`
- `wallet/hardware/TEST_LOG_TEMPLATE.md`
