import { PawClient } from '../src/client';
import { PawWallet } from '../src/wallet';

// Mock the Tendermint client
jest.mock('@cosmjs/tendermint-rpc');
jest.mock('@cosmjs/stargate');

describe('PawClient', () => {
  const config = {
    rpcEndpoint: 'http://localhost:26657',
    restEndpoint: 'http://localhost:1317',
    chainId: 'paw-testnet-1'
  };

  describe('constructor', () => {
    it('should create client with config', () => {
      const client = new PawClient(config);
      expect(client).toBeDefined();
      expect(client.bank).toBeDefined();
      expect(client.dex).toBeDefined();
      expect(client.staking).toBeDefined();
      expect(client.governance).toBeDefined();
    });

    it('should use default prefix and gas settings', () => {
      const client = new PawClient(config);
      const clientConfig = client.getConfig();

      expect(clientConfig.prefix).toBe('paw');
      expect(clientConfig.gasPrice).toBe('0.025upaw');
      expect(clientConfig.gasAdjustment).toBe(1.5);
    });

    it('should allow custom prefix and gas settings', () => {
      const customConfig = {
        ...config,
        prefix: 'custom',
        gasPrice: '0.05upaw',
        gasAdjustment: 2.0
      };

      const client = new PawClient(customConfig);
      const clientConfig = client.getConfig();

      expect(clientConfig.prefix).toBe('custom');
      expect(clientConfig.gasPrice).toBe('0.05upaw');
      expect(clientConfig.gasAdjustment).toBe(2.0);
    });
  });

  describe('getConfig', () => {
    it('should return the configuration', () => {
      const client = new PawClient(config);
      const returnedConfig = client.getConfig();

      expect(returnedConfig.rpcEndpoint).toBe(config.rpcEndpoint);
      expect(returnedConfig.chainId).toBe(config.chainId);
    });
  });

  describe('isConnected', () => {
    it('should return false when not connected', () => {
      const client = new PawClient(config);
      expect(client.isConnected()).toBe(false);
    });
  });

  describe('canSign', () => {
    it('should return false when wallet not connected', () => {
      const client = new PawClient(config);
      expect(client.canSign()).toBe(false);
    });
  });

  describe('error handling', () => {
    it('should throw error when accessing client before connection', () => {
      const client = new PawClient(config);
      expect(() => client.getClient()).toThrow('Client not connected');
    });

    it('should throw error when accessing signing client before wallet connection', () => {
      const client = new PawClient(config);
      expect(() => client.getSigningClient()).toThrow('Signing client not connected');
    });

    it('should throw error when accessing tx builder before wallet connection', () => {
      const client = new PawClient(config);
      expect(() => client.getTxBuilder()).toThrow('Transaction builder not available');
    });
  });
});
