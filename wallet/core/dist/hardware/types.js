"use strict";
/**
 * Hardware Wallet Type Definitions for PAW Chain
 * Supports Ledger and Trezor devices
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.DeviceConnectionStatus = exports.HardwareWalletType = void 0;
var HardwareWalletType;
(function (HardwareWalletType) {
    HardwareWalletType["LEDGER"] = "ledger";
    HardwareWalletType["TREZOR"] = "trezor";
})(HardwareWalletType || (exports.HardwareWalletType = HardwareWalletType = {}));
var DeviceConnectionStatus;
(function (DeviceConnectionStatus) {
    DeviceConnectionStatus["DISCONNECTED"] = "disconnected";
    DeviceConnectionStatus["CONNECTED"] = "connected";
    DeviceConnectionStatus["LOCKED"] = "locked";
    DeviceConnectionStatus["APP_NOT_OPEN"] = "app_not_open";
    DeviceConnectionStatus["BUSY"] = "busy";
})(DeviceConnectionStatus || (exports.DeviceConnectionStatus = DeviceConnectionStatus = {}));
//# sourceMappingURL=types.js.map