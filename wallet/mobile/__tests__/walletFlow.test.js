/**
 * End-to-end wallet flow tests (mocked services)
 * Exercises creation → balance → history → staking/dex discovery → cleanup.
 */

import WalletService from '../src/services/WalletService';
import KeyStore from '../src/services/KeyStore';
import PawAPI from '../src/services/PawAPI';
import BiometricAuth from '../src/services/BiometricAuth';

jest.mock('../src/services/KeyStore');
jest.mock('../src/services/PawAPI');
jest.mock('../src/services/BiometricAuth');

describe('Wallet end-to-end flows', () => {
  const address = 'paw1end2endtestaddressxyz';

  beforeEach(() => {
    jest.clearAllMocks();
    BiometricAuth.checkAvailability.mockResolvedValue({available: true});
    BiometricAuth.createKeys.mockResolvedValue(true);
  });

  it('creates wallet, retrieves balances/txs/pools/staking data, and cleans up', async () => {
    // Arrange mocks for creation/storage
    KeyStore.storeWallet.mockResolvedValue(true);
    KeyStore.storeMetadata.mockResolvedValue(true);
    KeyStore.getAddress.mockResolvedValue(address);
    KeyStore.getName.mockResolvedValue('E2E Wallet');
    KeyStore.storeTransactions.mockResolvedValue(true);
    KeyStore.clearAll.mockResolvedValue(true);

    // API mocks for downstream flows
    PawAPI.getBalance.mockResolvedValue({
      balances: [{denom: 'upaw', amount: '1234567'}],
    });
    const mockTxs = [{hash: 'ABC123', body: {messages: []}}];
    PawAPI.getTransactionsByAddress.mockResolvedValue(mockTxs);
    PawAPI.getDexPools.mockResolvedValue([{id: '1', tokenA: 'upaw', tokenB: 'uusdc'}]);
    PawAPI.getValidators.mockResolvedValue([{operator_address: 'val1', description: {moniker: 'Validator One'}}]);
    PawAPI.getDelegations.mockResolvedValue([{delegation: {validator_address: 'val1', shares: '100.0'}}]);
    PawAPI.getRewards.mockResolvedValue({total: [{denom: 'upaw', amount: '42'}]});

    // Act: full flow
    const wallet = await WalletService.createWallet({
      walletName: 'E2E Wallet',
      password: 'StrongPass123',
      useBiometric: true,
    });

    const balance = await WalletService.getBalance();
    const txs = await WalletService.getTransactions(10);
    const pools = await PawAPI.getDexPools();
    const validators = await PawAPI.getValidators();
    const delegations = await PawAPI.getDelegations(address);
    const rewards = await PawAPI.getRewards(address);
    const deleted = await WalletService.deleteWallet();

    // Assert: creation + storage
    expect(wallet.address).toMatch(/^paw1/);
    expect(KeyStore.storeWallet).toHaveBeenCalled();
    // Capture stored metadata to assert biometric flag without relying on generated address.
    const storedMetaCall = KeyStore.storeMetadata.mock.calls[0]?.[0];
    expect(storedMetaCall).toMatchObject({name: 'E2E Wallet', biometricEnabled: true});
    expect(BiometricAuth.checkAvailability).toHaveBeenCalled();
    expect(BiometricAuth.createKeys).toHaveBeenCalled();

    // Assert: balance + history + caching
    expect(balance.amount).toBe(1234567);
    expect(balance.formatted).toBe('1.234567');
    expect(txs).toEqual(mockTxs);
    expect(KeyStore.storeTransactions).toHaveBeenCalledWith(mockTxs);

    // Assert: ancillary data calls
    expect(pools).toHaveLength(1);
    expect(validators[0].description.moniker).toBe('Validator One');
    expect(delegations[0].delegation.validator_address).toBe('val1');
    expect(rewards.total[0].amount).toBe('42');

    // Assert: cleanup
    expect(deleted).toBe(true);
    expect(KeyStore.clearAll).toHaveBeenCalled();
  });
});
