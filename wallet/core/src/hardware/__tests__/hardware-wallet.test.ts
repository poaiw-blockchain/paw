/**
 * Hardware Wallet Tests
 *
 * Note: These are unit tests with mocked hardware wallet interfaces.
 * Real device testing requires actual hardware and should be done manually.
 */

import {
  HardwareWalletFactory,
  HardwareWalletUtils,
  HardwareWalletManager,
  HardwareWalletType,
  DeviceConnectionStatus,
} from '../index';
import { toBech32 } from '@cosmjs/encoding';

const mockLedgerWallet = {
  type: HardwareWalletType.LEDGER,
  connect: jest.fn(),
  disconnect: jest.fn(),
  isConnected: jest.fn().mockResolvedValue(true),
  getDeviceInfo: jest.fn(),
};

const mockTrezorWallet = {
  type: HardwareWalletType.TREZOR,
  connect: jest.fn(),
  disconnect: jest.fn(),
  isConnected: jest.fn().mockReturnValue(true),
  getDeviceInfo: jest.fn(),
};

// Mock the hardware wallet modules
jest.mock('../ledger', () => ({
  createLedgerWallet: jest.fn(() => mockLedgerWallet),
  isLedgerSupported: jest.fn().mockResolvedValue(true),
  requestLedgerDevice: jest.fn().mockResolvedValue(undefined),
}));

jest.mock('../trezor', () => ({
  createTrezorWallet: jest.fn(() => mockTrezorWallet),
  isTrezorSupported: jest.fn().mockReturnValue(true),
}));

describe('HardwareWalletUtils', () => {
  describe('generatePaths', () => {
    it('should generate correct derivation paths', () => {
      const paths = HardwareWalletUtils.generatePaths(118, 5, 0);

      expect(paths).toHaveLength(5);
      expect(paths[0]).toBe("m/44'/118'/0'/0/0");
      expect(paths[4]).toBe("m/44'/118'/0'/0/4");
    });

    it('should handle different coin types', () => {
      const paths = HardwareWalletUtils.generatePaths(60, 3, 0);

      expect(paths[0]).toBe("m/44'/60'/0'/0/0");
      expect(paths[2]).toBe("m/44'/60'/0'/0/2");
    });
  });

  describe('buildDefaultPathMatrix', () => {
    it('should build default matrix up to max account index', () => {
      const paths = HardwareWalletUtils.buildDefaultPathMatrix(118, 4, 0);
      expect(paths).toHaveLength(5);
      expect(paths[0]).toBe("m/44'/118'/0'/0/0");
      expect(paths[4]).toBe("m/44'/118'/0'/0/4");
    });
  });

  describe('parsePath', () => {
    it('should parse valid derivation path', () => {
      const result = HardwareWalletUtils.parsePath("m/44'/118'/0'/0/0");

      expect(result).toEqual({
        coinType: 118,
        account: 0,
        change: 0,
        index: 0,
      });
    });

    it('should throw on invalid path', () => {
      expect(() => {
        HardwareWalletUtils.parsePath('invalid/path');
      }).toThrow('Invalid derivation path');
    });
  });

  describe('isValidPath', () => {
    it('should validate correct paths', () => {
      expect(HardwareWalletUtils.isValidPath("m/44'/118'/0'/0/0")).toBe(true);
      expect(HardwareWalletUtils.isValidPath("m/44'/60'/1'/0/5")).toBe(true);
    });

    it('should reject invalid paths', () => {
      expect(HardwareWalletUtils.isValidPath('invalid')).toBe(false);
      expect(HardwareWalletUtils.isValidPath('m/44/118/0/0/0')).toBe(false);
      expect(HardwareWalletUtils.isValidPath("m/44'/118'/0/0")).toBe(false);
    });
  });

  describe('getErrorMessage', () => {
    it('should return user-friendly error messages', () => {
      const error = new Error('Test error') as any;

      error.code = 'USER_REJECTED';
      expect(HardwareWalletUtils.getErrorMessage(error)).toBe(
        'Transaction was rejected on the device'
      );

      error.code = 'NOT_CONNECTED';
      expect(HardwareWalletUtils.getErrorMessage(error)).toBe(
        'Hardware wallet is not connected'
      );

      error.code = 'DEVICE_LOCKED';
      expect(HardwareWalletUtils.getErrorMessage(error)).toBe(
        'Please unlock your device'
      );

      error.code = 'APP_NOT_OPEN';
      expect(HardwareWalletUtils.getErrorMessage(error)).toBe(
        'Please open the Cosmos app on your device'
      );
    });

    it('should handle unknown errors', () => {
      const error = new Error('Custom error') as any;
      error.code = 'UNKNOWN';

      expect(HardwareWalletUtils.getErrorMessage(error)).toBe('Custom error');
    });
  });

  describe('bech32 and fee validation helpers', () => {
    it('should validate bech32 prefixes', () => {
      const good = toBech32('paw', new Uint8Array(20).fill(1));
      expect(() => HardwareWalletUtils.assertBech32Prefix(good, 'paw')).not.toThrow();
      expect(() => HardwareWalletUtils.assertBech32Prefix(good, 'cosmos')).toThrow(
        /Address prefix mismatch/
      );
    });

    it('should validate sign doc basics with allowed denom and chain-id', () => {
      expect(() =>
        HardwareWalletUtils.validateSignDocBasics(
          {
            chain_id: 'paw-testnet-1',
            fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
          },
          { enforceChainId: 'paw-testnet-1', allowedFeeDenoms: ['upaw'] }
        )
      ).not.toThrow();

      expect(() =>
        HardwareWalletUtils.validateSignDocBasics(
          {
            chain_id: 'other-chain',
            fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
          },
          { enforceChainId: 'paw-testnet-1', allowedFeeDenoms: ['upaw'] }
        )
      ).toThrow(/chain-id mismatch/);

      expect(() =>
        HardwareWalletUtils.validateSignDocBasics(
          {
            chain_id: 'paw-testnet-1',
            fee: { amount: [{ denom: 'uatom', amount: '2500' }], gas: '200000' },
          },
          { enforceChainId: 'paw-testnet-1', allowedFeeDenoms: ['upaw'] }
        )
      ).toThrow(/not permitted/);
    });
  });
});

describe('HardwareWalletFactory', () => {
  describe('create', () => {
    it('should create Ledger wallet', () => {
      const wallet = HardwareWalletFactory.create(HardwareWalletType.LEDGER);
      expect(wallet.type).toBe(HardwareWalletType.LEDGER);
    });

    it('should create Trezor wallet', () => {
      const wallet = HardwareWalletFactory.create(HardwareWalletType.TREZOR);
      expect(wallet.type).toBe(HardwareWalletType.TREZOR);
    });

    it('should throw on unsupported type', () => {
      expect(() => {
        HardwareWalletFactory.create('unsupported' as any);
      }).toThrow('Unsupported hardware wallet type');
    });
  });

  describe('getSupportedWallets', () => {
    it('should return supported wallet types', async () => {
      const supported = await HardwareWalletFactory.getSupportedWallets();

      expect(Array.isArray(supported)).toBe(true);
      expect(supported).toContain(HardwareWalletType.LEDGER);
      expect(supported).toContain(HardwareWalletType.TREZOR);
    });
  });
});

describe('HardwareWalletManager', () => {
  let manager: HardwareWalletManager;

  beforeEach(() => {
    manager = new HardwareWalletManager();
  });

  afterEach(async () => {
    await manager.disconnectAll();
  });

  describe('wallet management', () => {
    it('should add and retrieve wallets', async () => {
      // Mock wallet connect
      const mockConnect = jest.fn().mockResolvedValue({
        type: HardwareWalletType.LEDGER,
        model: 'Ledger Nano S',
        version: '2.0.0',
        deviceId: 'test-device-123',
        status: DeviceConnectionStatus.CONNECTED,
      });

      // Mock the factory
      jest.spyOn(HardwareWalletFactory, 'create').mockReturnValue({
        type: HardwareWalletType.LEDGER,
        connect: mockConnect,
        disconnect: jest.fn(),
      } as any);

      const id = await manager.addWallet(HardwareWalletType.LEDGER);

      expect(id).toBeTruthy();
      expect(manager.getWallet(id)).toBeDefined();
    });

    it('should remove wallets', async () => {
      const mockConnect = jest.fn().mockResolvedValue({
        type: HardwareWalletType.LEDGER,
        model: 'Ledger Nano S',
        version: '2.0.0',
        status: DeviceConnectionStatus.CONNECTED,
      });

      const mockDisconnect = jest.fn();

      jest.spyOn(HardwareWalletFactory, 'create').mockReturnValue({
        type: HardwareWalletType.LEDGER,
        connect: mockConnect,
        disconnect: mockDisconnect,
      } as any);

      const id = await manager.addWallet(HardwareWalletType.LEDGER);
      await manager.removeWallet(id);

      expect(mockDisconnect).toHaveBeenCalled();
      expect(manager.getWallet(id)).toBeUndefined();
    });

    it('should disconnect all wallets', async () => {
      const disconnects: jest.Mock[] = [];
      let deviceCounter = 0;
      const mockConnect = jest.fn().mockImplementation(() =>
        Promise.resolve({
          type: HardwareWalletType.LEDGER,
          model: 'Ledger Nano S',
          version: '2.0.0',
          deviceId: `device-${deviceCounter++}`,
          status: DeviceConnectionStatus.CONNECTED,
        })
      );

      jest.spyOn(HardwareWalletFactory, 'create').mockImplementation(() => {
        const mockDisconnect = jest.fn();
        disconnects.push(mockDisconnect);
        return {
          type: HardwareWalletType.LEDGER,
          connect: mockConnect,
          disconnect: mockDisconnect,
        } as any;
      });

      await manager.addWallet(HardwareWalletType.LEDGER);
      await manager.addWallet(HardwareWalletType.LEDGER);

      await manager.disconnectAll();

      expect(disconnects.every((fn) => fn.mock.calls.length === 1)).toBe(true);
      expect(manager.getAllWallets().size).toBe(0);
    });
  });
});

describe('Path Derivation', () => {
  it('should generate consistent paths for account discovery', () => {
    const paths = HardwareWalletUtils.generatePaths(118, 10, 0);

    // Verify all paths follow BIP44 standard
    paths.forEach((path, index) => {
      const parsed = HardwareWalletUtils.parsePath(path);
      expect(parsed.coinType).toBe(118);
      expect(parsed.account).toBe(0);
      expect(parsed.change).toBe(0);
      expect(parsed.index).toBe(index);
    });
  });

  it('should handle multiple accounts', () => {
    const account0Paths = HardwareWalletUtils.generatePaths(118, 3, 0);
    const account1Paths = HardwareWalletUtils.generatePaths(118, 3, 1);

    const parsed0 = HardwareWalletUtils.parsePath(account0Paths[0]);
    const parsed1 = HardwareWalletUtils.parsePath(account1Paths[0]);

    expect(parsed0.account).toBe(0);
    expect(parsed1.account).toBe(1);
  });
});

describe('Error Handling', () => {
  it('should provide helpful error messages for common issues', () => {
    const errorCodes = [
      'USER_REJECTED',
      'NOT_CONNECTED',
      'DEVICE_LOCKED',
      'APP_NOT_OPEN',
      'INVALID_DATA',
      'INVALID_PATH',
    ];

    errorCodes.forEach(code => {
      const error = new Error() as any;
      error.code = code;
      const message = HardwareWalletUtils.getErrorMessage(error);

      expect(message).toBeTruthy();
      expect(message.length).toBeGreaterThan(0);
    });
  });
});
