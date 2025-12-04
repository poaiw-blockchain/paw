/**
 * PawAPI service tests
 */

import PawAPI from '../src/services/PawAPI';
import axios from 'axios';

jest.mock('axios');

describe('PawAPI Service', () => {
  let mockAxiosInstance;

  beforeEach(() => {
    mockAxiosInstance = {
      get: jest.fn(),
      post: jest.fn(),
      defaults: {
        baseURL: 'http://localhost:1317',
      },
    };
    axios.create.mockReturnValue(mockAxiosInstance);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Configuration', () => {
    test('should set base URL', () => {
      const newUrl = 'http://example.com:1317';
      PawAPI.setBaseURL(newUrl);
      expect(PawAPI.baseURL).toBe(newUrl);
    });
  });

  describe('Node Info', () => {
    test('should get node info', async () => {
      const mockNodeInfo = {
        node_info: {
          network: 'paw-1',
          version: '1.0.0',
        },
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockNodeInfo});

      const result = await PawAPI.getNodeInfo();
      expect(result).toEqual(mockNodeInfo);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith(
        '/cosmos/base/tendermint/v1beta1/node_info',
      );
    });

    test('should handle node info error', async () => {
      mockAxiosInstance.get.mockRejectedValue(new Error('Network error'));

      await expect(PawAPI.getNodeInfo()).rejects.toThrow();
    });
  });

  describe('Balance', () => {
    test('should get account balance', async () => {
      const address = 'paw1test';
      const mockBalance = {
        balances: [
          {denom: 'upaw', amount: '1000000'},
        ],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockBalance});

      const result = await PawAPI.getBalance(address);
      expect(result).toEqual(mockBalance);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith(
        `/cosmos/bank/v1beta1/balances/${address}`,
      );
    });

    test('should handle balance error', async () => {
      mockAxiosInstance.get.mockRejectedValue({
        response: {
          data: {message: 'Account not found'},
        },
      });

      await expect(PawAPI.getBalance('invalid')).rejects.toThrow('API Error');
    });
  });

  describe('Account Info', () => {
    test('should get account info', async () => {
      const address = 'paw1test';
      const mockAccount = {
        account: {
          '@type': '/cosmos.auth.v1beta1.BaseAccount',
          address,
          sequence: '5',
          account_number: '10',
        },
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockAccount});

      const result = await PawAPI.getAccount(address);
      expect(result).toEqual(mockAccount.account);
    });
  });

  describe('Transactions', () => {
    test('should get transaction by hash', async () => {
      const hash = 'ABC123';
      const mockTx = {
        tx: {},
        tx_response: {
          txhash: hash,
          height: '100',
        },
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockTx});

      const result = await PawAPI.getTransaction(hash);
      expect(result).toEqual(mockTx);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith(
        `/cosmos/tx/v1beta1/txs/${hash}`,
      );
    });

    test('should get transactions by address', async () => {
      const address = 'paw1test';
      const mockTxs = {
        txs: [
          {txhash: 'ABC123'},
          {txhash: 'DEF456'},
        ],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockTxs});

      const result = await PawAPI.getTransactionsByAddress(address, 10);
      expect(result).toEqual(mockTxs.txs);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith(
        '/cosmos/tx/v1beta1/txs',
        expect.objectContaining({
          params: expect.objectContaining({
            events: `message.sender='${address}'`,
          }),
        }),
      );
    });

    test('should return empty array when no transactions', async () => {
      mockAxiosInstance.get.mockResolvedValue({data: {}});

      const result = await PawAPI.getTransactionsByAddress('paw1test');
      expect(result).toEqual([]);
    });
  });

  describe('DEX', () => {
    test('should get DEX pools', async () => {
      const mockPools = {
        pools: [
          {id: '1', token_a: 'upaw', token_b: 'uatom'},
        ],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockPools});

      const result = await PawAPI.getDexPools();
      expect(result).toEqual(mockPools.pools);
    });

    test('should get specific pool', async () => {
      const poolId = '1';
      const mockPool = {
        pool: {id: poolId, token_a: 'upaw', token_b: 'uatom'},
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockPool});

      const result = await PawAPI.getPool(poolId);
      expect(result).toEqual(mockPool.pool);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith(
        `/paw/dex/v1/pools/${poolId}`,
      );
    });
  });

  describe('Oracle', () => {
    test('should get oracle prices', async () => {
      const mockPrices = {
        prices: [
          {symbol: 'PAW/USD', price: '1.50'},
        ],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockPrices});

      const result = await PawAPI.getOraclePrices();
      expect(result).toEqual(mockPrices.prices);
    });
  });

  describe('Staking', () => {
    test('should get validators', async () => {
      const mockValidators = {
        validators: [
          {operator_address: 'pawvaloper1test'},
        ],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockValidators});

      const result = await PawAPI.getValidators();
      expect(result).toEqual(mockValidators.validators);
    });

    test('should get delegations', async () => {
      const address = 'paw1test';
      const mockDelegations = {
        delegation_responses: [
          {delegation: {delegator_address: address}},
        ],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockDelegations});

      const result = await PawAPI.getDelegations(address);
      expect(result).toEqual(mockDelegations.delegation_responses);
    });

    test('should get rewards', async () => {
      const address = 'paw1test';
      const mockRewards = {
        rewards: [
          {validator_address: 'pawvaloper1test', reward: [{denom: 'upaw', amount: '100'}]},
        ],
        total: [{denom: 'upaw', amount: '100'}],
      };

      mockAxiosInstance.get.mockResolvedValue({data: mockRewards});

      const result = await PawAPI.getRewards(address);
      expect(result).toEqual(mockRewards);
    });
  });

  describe('Error Handling', () => {
    test('should handle network error', async () => {
      const networkError = new Error('Network error');
      networkError.request = {};
      delete networkError.response; // Ensure only request property exists
      mockAxiosInstance.get.mockRejectedValue(networkError);

      await expect(PawAPI.getNodeInfo()).rejects.toThrow();
    });

    test('should handle API error with message', async () => {
      const apiError = new Error('API Error');
      apiError.response = {
        data: {message: 'Custom error message'},
      };
      delete apiError.request; // Ensure only response property exists
      mockAxiosInstance.get.mockRejectedValue(apiError);

      await expect(PawAPI.getNodeInfo()).rejects.toThrow();
    });

    test('should handle generic error', async () => {
      const genericError = new Error('Something went wrong');
      delete genericError.response;
      delete genericError.request;
      mockAxiosInstance.get.mockRejectedValue(genericError);

      await expect(PawAPI.getNodeInfo()).rejects.toThrow('Request error');
    });
  });
});
