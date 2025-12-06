import { DexService } from '../../src/services/dex';

const metadataByDenom: Record<string, any> = {
  upaw: {
    symbol: 'PAW',
    display: 'paw',
    denom_units: [
      { denom: 'upaw', exponent: 0 },
      { denom: 'paw', exponent: 6 },
    ],
  },
  uatom: {
    symbol: 'ATOM',
    display: 'atom',
    denom_units: [
      { denom: 'uatom', exponent: 0 },
      { denom: 'atom', exponent: 6 },
    ],
  },
};

const mockApiServiceInstance = {
  getEndpoint: jest.fn(),
  getDenomMetadata: jest.fn(),
};

const mockKeystoreInstance = {
  unlockWallet: jest.fn(),
};

const mockRpcClientInstance = {
  getPools: jest.fn(),
  simulateSwap: jest.fn(),
  getAllPrices: jest.fn(),
};

const mockWalletInstance = {
  createFromMnemonic: jest.fn(),
  swap: jest.fn(),
};

const mockPAWRPCClientCtor = jest.fn();
const mockPAWWalletCtor = jest.fn();

jest.mock('../../src/services/api', () => ({
  ApiService: jest.fn(() => mockApiServiceInstance),
}));

jest.mock('../../src/services/keystore', () => ({
  KeystoreService: jest.fn(() => mockKeystoreInstance),
}));

jest.mock(
  '@paw-chain/wallet-core',
  () => ({
    PAWRPCClient: function mockPAWRPCClient(this: any, ...args: any[]) {
      mockPAWRPCClientCtor(...args);
      return mockRpcClientInstance;
    },
    PAWWallet: function mockPAWWallet(this: any, ...args: any[]) {
      mockPAWWalletCtor(...args);
      return mockWalletInstance;
    },
  }),
  { virtual: true }
);

describe('DexService', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    mockApiServiceInstance.getEndpoint.mockResolvedValue('http://localhost:1317');
    mockApiServiceInstance.getDenomMetadata.mockImplementation(async (denom: string) => metadataByDenom[denom]);

    mockKeystoreInstance.unlockWallet.mockResolvedValue({
      address: 'paw1testaddress',
      publicKey: 'pubkey',
      privateKey: 'test mnemonic',
    });

    mockRpcClientInstance.getPools.mockResolvedValue([
      {
        id: 1,
        tokenA: 'upaw',
        tokenB: 'uatom',
        reserveA: '500000000000',
        reserveB: '250000000000',
      },
    ]);
    mockRpcClientInstance.simulateSwap.mockResolvedValue({ amountOut: '4900000' });
    mockRpcClientInstance.getAllPrices.mockResolvedValue([]);

    mockWalletInstance.createFromMnemonic.mockResolvedValue(undefined);
    mockWalletInstance.swap.mockResolvedValue({
      code: 0,
      transactionHash: 'TX_HASH',
    });
  });

  function createService() {
    return new DexService();
  }

  it('quotes swaps using pool simulation and denom metadata', async () => {
    const service = createService();
    const quote = await service.quoteSwap({
      tokenIn: 'upaw',
      tokenOut: 'uatom',
      amountIn: '10',
      slippagePercent: 0.5,
    });

    expect(mockRpcClientInstance.simulateSwap).toHaveBeenCalledWith(1, 'upaw', 'uatom', '10000000');
    expect(quote).toMatchObject({
      poolId: 1,
      tokenIn: expect.objectContaining({ denom: 'upaw', symbol: 'PAW' }),
      tokenOut: expect.objectContaining({ denom: 'uatom', symbol: 'ATOM' }),
      normalizedAmountIn: '10000000',
      expectedAmountOut: '4.9',
      minAmountOutBase: '4875500',
    });
    expect(Number(quote.priceImpactPercent)).toBeGreaterThanOrEqual(0);
    expect(quote.executionPrice).toBe('0.49');
  });

  it('executes swaps end-to-end with decrypted wallet and RPC config', async () => {
    const service = createService();
    const result = await service.executeSwap({
      tokenIn: 'upaw',
      tokenOut: 'uatom',
      amountIn: '10',
      slippagePercent: 0.5,
      password: 'super-secret',
      memo: 'routing',
    });

    expect(mockKeystoreInstance.unlockWallet).toHaveBeenCalledWith('super-secret');
    expect(mockWalletInstance.createFromMnemonic).toHaveBeenCalledWith('test mnemonic');
    expect(mockWalletInstance.swap).toHaveBeenCalledWith(
      1,
      'upaw',
      'uatom',
      '10000000',
      '4875500',
      { memo: 'routing' }
    );
    expect(mockPAWWalletCtor).toHaveBeenCalledWith({
      rpcConfig: {
        restUrl: 'http://localhost:1317',
        rpcUrl: 'http://localhost:26657',
      },
    });
    expect(result).toEqual({
      code: 0,
      transactionHash: 'TX_HASH',
    });
  });
});
