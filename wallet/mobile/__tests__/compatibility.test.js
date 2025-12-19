/**
 * Cross-wallet compatibility vectors to ensure deterministic key/address derivation.
 * These fixtures must remain stable to interoperate with other PAW wallets.
 */

import {
  derivePrivateKeyFromMnemonic,
  getPublicKey,
  deriveAddress,
} from '../src/utils/crypto';

describe('Compatibility Vectors', () => {
  const mnemonic =
    'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
  const expected = {
    privateKey:
      '5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc1',
    publicKey:
      '029058af2e7b6f0dc54d96925b80868515bf87f3158e95afce81927b3b772d5b24',
    address: 'paw1h9p5k7s4hyt3jds5xh27sksnpw2uzsjgkqxecm',
  };

  it('derives expected keys and address from reference mnemonic', () => {
    const priv = derivePrivateKeyFromMnemonic(mnemonic);
    expect(priv).toBe(expected.privateKey);

    const pub = getPublicKey(priv);
    expect(pub).toBe(expected.publicKey);

    const addr = deriveAddress(pub);
    expect(addr).toBe(expected.address);
  });
});
