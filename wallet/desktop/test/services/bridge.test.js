import { BridgeService } from '../../src/services/bridge';
import { GasPrice, SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';

jest.mock('@cosmjs/stargate', () => {
  const sendIbcTokens = jest.fn(() => Promise.resolve({ transactionHash: 'ABC123' }));
  return {
    GasPrice: { fromString: jest.fn() },
    SigningStargateClient: {
      connectWithSigner: jest.fn(() =>
        Promise.resolve({
          sendIbcTokens,
        })
      ),
    },
    __esModule: true,
  };
});

jest.mock('@cosmjs/proto-signing', () => ({
  DirectSecp256k1HdWallet: {
    fromMnemonic: jest.fn(() =>
      Promise.resolve({
        getAccounts: jest.fn(() => Promise.resolve([{ address: 'paw1addr' }])),
      })
    ),
  },
  __esModule: true,
}));

jest.mock('../../src/services/keystore', () => ({
  KeystoreService: jest.fn().mockImplementation(() => ({
    unlockWallet: jest.fn(() => Promise.resolve({ privateKey: 'seed mnemonic' })),
  })),
}));

jest.mock('../../src/services/api', () => ({
  ApiService: jest.fn().mockImplementation(() => ({
    getEndpoint: jest.fn(() => Promise.resolve('http://localhost:1317')),
  })),
}));

describe('BridgeService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('builds and broadcasts IBC transfer with defaults', async () => {
    const service = new BridgeService();
    const res = await service.bridgeTokens({
      password: 'password123',
      amount: '10',
      denom: 'upaw',
      destAddress: 'cosmos1destination',
    });

    expect(res.transactionHash).toBe('ABC123');
    expect(SigningStargateClient.connectWithSigner).toHaveBeenCalledTimes(1);
    const client = await SigningStargateClient.connectWithSigner.mock.results[0].value;
    expect(client.sendIbcTokens).toHaveBeenCalledTimes(1);
    const args = client.sendIbcTokens.mock.calls[0];
    expect(args[0]).toBe('paw1addr');
    expect(args[1]).toBe('cosmos1destination');
    expect(args[3]).toBe('transfer');
    expect(args[4]).toBe('channel-0');
    expect(typeof args[6]).toBe('bigint');
  });

  it('rejects when password is missing', async () => {
    const service = new BridgeService();
    await expect(
      service.bridgeTokens({
        password: '',
        amount: '1',
        denom: 'upaw',
        destAddress: 'cosmos1destination',
      })
    ).rejects.toThrow(/Password/);
  });
});
