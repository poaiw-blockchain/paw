/**
 * API Service Tests
 */

import { ApiService } from '../src/services/api';
import axios from 'axios';

jest.mock('axios');

describe('ApiService', () => {
  let apiService;

  beforeEach(() => {
    apiService = new ApiService();
    jest.clearAllMocks();
  });

  describe('Balance', () => {
    test('should fetch account balance', async () => {
      const mockBalance = {
        balances: [
          { denom: 'upaw', amount: '1000000' }
        ]
      };

      axios.get.mockResolvedValue({ data: mockBalance });

      const balance = await apiService.getBalance('paw1test');

      expect(axios.get).toHaveBeenCalled();
      expect(balance).toEqual(mockBalance);
    });

    test('should handle balance fetch error', async () => {
      axios.get.mockRejectedValue(new Error('Network error'));

      await expect(
        apiService.getBalance('paw1test')
      ).rejects.toThrow();
    });
  });

  describe('Account', () => {
    test('should fetch account information', async () => {
      const mockAccount = {
        account: {
          address: 'paw1test',
          account_number: '1',
          sequence: '0'
        }
      };

      axios.get.mockResolvedValue({ data: mockAccount });

      const account = await apiService.getAccount('paw1test');

      expect(axios.get).toHaveBeenCalled();
      expect(account).toEqual(mockAccount.account);
    });
  });

  describe('Transactions', () => {
    test('should fetch transaction history', async () => {
      const mockTxs = {
        txs: [
          { txhash: 'hash1', height: '100' },
          { txhash: 'hash2', height: '101' }
        ]
      };

      axios.get.mockResolvedValue({ data: mockTxs });

      const txs = await apiService.getTransactions('paw1test');

      expect(axios.get).toHaveBeenCalled();
      expect(txs).toEqual(mockTxs.txs);
    });

    test('should return empty array on error', async () => {
      axios.get.mockRejectedValue(new Error('Not found'));

      const txs = await apiService.getTransactions('paw1test');

      expect(txs).toEqual([]);
    });
  });

  describe('Node Information', () => {
    test('should fetch node info', async () => {
      const mockNodeInfo = {
        node_info: {
          network: 'paw-testnet',
          version: '1.0.0'
        }
      };

      axios.get.mockResolvedValue({ data: mockNodeInfo });

      const nodeInfo = await apiService.getNodeInfo();

      expect(axios.get).toHaveBeenCalled();
      expect(nodeInfo).toEqual(mockNodeInfo.node_info);
    });
  });

  describe('Validators', () => {
    test('should fetch validator list', async () => {
      const mockValidators = {
        validators: [
          { operator_address: 'pawvaloper1test', status: 'BOND_STATUS_BONDED' }
        ]
      };

      axios.get.mockResolvedValue({ data: mockValidators });

      const validators = await apiService.getValidators();

      expect(axios.get).toHaveBeenCalled();
      expect(validators).toEqual(mockValidators.validators);
    });
  });

  describe('DEX Pools', () => {
    test('should fetch DEX pools', async () => {
      const mockPools = {
        pools: [
          { id: '1', token_a: 'upaw', token_b: 'uatom' }
        ]
      };

      axios.get.mockResolvedValue({ data: mockPools });

      const pools = await apiService.getDexPools();

      expect(axios.get).toHaveBeenCalled();
      expect(pools).toEqual(mockPools.pools);
    });

    test('should return empty array if DEX not available', async () => {
      axios.get.mockRejectedValue(new Error('Not found'));

      const pools = await apiService.getDexPools();

      expect(pools).toEqual([]);
    });
  });

  describe('Oracle Prices', () => {
    test('should fetch oracle prices', async () => {
      const mockPrices = {
        prices: [
          { symbol: 'PAW/USD', price: '1.5' }
        ]
      };

      axios.get.mockResolvedValue({ data: mockPrices });

      const prices = await apiService.getOraclePrices();

      expect(axios.get).toHaveBeenCalled();
      expect(prices).toEqual(mockPrices.prices);
    });

    test('should return empty array if oracle not available', async () => {
      axios.get.mockRejectedValue(new Error('Not found'));

      const prices = await apiService.getOraclePrices();

      expect(prices).toEqual([]);
    });
  });
});
