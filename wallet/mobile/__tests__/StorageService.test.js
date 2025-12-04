/**
 * Storage Service Tests
 */

import StorageService from '../src/services/StorageService';
import AsyncStorage from '@react-native-async-storage/async-storage';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  setItem: jest.fn(),
  getItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
  getAllKeys: jest.fn(),
}));

describe('StorageService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('setItem and getItem', () => {
    it('should store and retrieve items', async () => {
      const key = '@TEST:key';
      const value = {test: 'data'};

      AsyncStorage.setItem.mockResolvedValue();
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(value));

      await StorageService.setItem(key, value);
      const result = await StorageService.getItem(key);

      expect(AsyncStorage.setItem).toHaveBeenCalledWith(key, JSON.stringify(value));
      expect(result).toEqual(value);
    });

    it('should return default value when item not found', async () => {
      AsyncStorage.getItem.mockResolvedValue(null);

      const result = await StorageService.getItem('@TEST:missing', 'default');

      expect(result).toBe('default');
    });
  });

  describe('removeItem', () => {
    it('should remove item successfully', async () => {
      AsyncStorage.removeItem.mockResolvedValue();

      const result = await StorageService.removeItem('@TEST:key');

      expect(result).toBe(true);
      expect(AsyncStorage.removeItem).toHaveBeenCalledWith('@TEST:key');
    });
  });

  describe('Theme settings', () => {
    it('should get and set theme', async () => {
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify('light'));
      AsyncStorage.setItem.mockResolvedValue();

      await StorageService.setTheme('light');
      const theme = await StorageService.getTheme();

      expect(theme).toBe('light');
    });

    it('should return default theme', async () => {
      AsyncStorage.getItem.mockResolvedValue(null);

      const theme = await StorageService.getTheme();

      expect(theme).toBe('dark');
    });
  });

  describe('Network settings', () => {
    it('should store and retrieve network config', async () => {
      const network = {
        name: 'testnet',
        rpcUrl: 'http://testnet:1317',
        chainId: 'paw-testnet',
      };

      AsyncStorage.setItem.mockResolvedValue();
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(network));

      await StorageService.setNetwork(network);
      const result = await StorageService.getNetwork();

      expect(result).toEqual(network);
    });
  });

  describe('Address Book', () => {
    it('should add address to address book', async () => {
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify([]));
      AsyncStorage.setItem.mockResolvedValue();

      const entry = await StorageService.addAddress({
        name: 'Test User',
        address: 'paw1test',
        note: 'Test note',
      });

      expect(entry.name).toBe('Test User');
      expect(entry.address).toBe('paw1test');
      expect(entry.id).toBeDefined();
    });

    it('should remove address from address book', async () => {
      const addressBook = [
        {id: '1', name: 'User 1', address: 'paw1test1'},
        {id: '2', name: 'User 2', address: 'paw1test2'},
      ];

      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(addressBook));
      AsyncStorage.setItem.mockResolvedValue();

      await StorageService.removeAddress('1');

      const setCall = AsyncStorage.setItem.mock.calls[0];
      const savedData = JSON.parse(setCall[1]);
      expect(savedData.length).toBe(1);
      expect(savedData[0].id).toBe('2');
    });

    it('should update address in address book', async () => {
      const addressBook = [{id: '1', name: 'User 1', address: 'paw1test1'}];

      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(addressBook));
      AsyncStorage.setItem.mockResolvedValue();

      const updated = await StorageService.updateAddress('1', {name: 'Updated User'});

      expect(updated.name).toBe('Updated User');
    });
  });

  describe('Recent Addresses', () => {
    it('should add recent address', async () => {
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify([]));
      AsyncStorage.setItem.mockResolvedValue();

      await StorageService.addRecentAddress('paw1test');

      const setCall = AsyncStorage.setItem.mock.calls[0];
      const savedData = JSON.parse(setCall[1]);
      expect(savedData[0]).toBe('paw1test');
    });

    it('should limit recent addresses to 10', async () => {
      const existing = Array(10).fill(null).map((_, i) => `paw1test${i}`);
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(existing));
      AsyncStorage.setItem.mockResolvedValue();

      await StorageService.addRecentAddress('paw1new');

      const setCall = AsyncStorage.setItem.mock.calls[0];
      const savedData = JSON.parse(setCall[1]);
      expect(savedData.length).toBe(10);
      expect(savedData[0]).toBe('paw1new');
    });
  });

  describe('Price Alerts', () => {
    it('should add price alert', async () => {
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify([]));
      AsyncStorage.setItem.mockResolvedValue();

      const alert = await StorageService.addPriceAlert({
        type: 'above',
        price: 100,
      });

      expect(alert.type).toBe('above');
      expect(alert.price).toBe(100);
      expect(alert.enabled).toBe(true);
    });

    it('should toggle price alert', async () => {
      const alerts = [{id: '1', type: 'above', price: 100, enabled: true}];
      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(alerts));
      AsyncStorage.setItem.mockResolvedValue();

      const toggled = await StorageService.togglePriceAlert('1');

      expect(toggled.enabled).toBe(false);
    });
  });
});
