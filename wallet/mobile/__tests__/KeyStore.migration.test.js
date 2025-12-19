import CryptoJS from 'crypto-js';
import * as Keychain from 'react-native-keychain';
import KeyStore from '../src/services/KeyStore';
import {encrypt} from '../src/utils/crypto';

describe('KeyStore encryption migration', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  test('re-encrypts legacy payloads when retrieved', async () => {
    const legacyPrivate = CryptoJS.AES.encrypt('priv', 'pass').toString();
    const legacyMnemonic = CryptoJS.AES.encrypt('mnemonic', 'pass').toString();
    Keychain.getGenericPassword.mockResolvedValue({
      username: 'paw_wallet',
      password: JSON.stringify({
        privateKey: legacyPrivate,
        mnemonic: legacyMnemonic,
      }),
    });

    const wallet = await KeyStore.retrieveWallet('pass');

    expect(wallet.privateKey).toBe('priv');
    expect(wallet.mnemonic).toBe('mnemonic');
    expect(Keychain.setGenericPassword).toHaveBeenCalledTimes(1);
    const [username, storedPayload] = Keychain.setGenericPassword.mock.calls[0];
    expect(username).toBe('paw_wallet');
    const stored = JSON.parse(storedPayload);
    expect(stored.privateKey).toContain('ct');
    expect(stored.mnemonic).toContain('ct');
  });

  test('keeps hardened payloads unchanged', async () => {
    const hardenedPriv = encrypt('priv', 'pass');
    const hardenedMnemonic = encrypt('mnemonic', 'pass');
    Keychain.getGenericPassword.mockResolvedValue({
      username: 'paw_wallet',
      password: JSON.stringify({
        privateKey: hardenedPriv,
        mnemonic: hardenedMnemonic,
      }),
    });

    await KeyStore.retrieveWallet('pass');

    expect(Keychain.setGenericPassword).not.toHaveBeenCalled();
  });
});
