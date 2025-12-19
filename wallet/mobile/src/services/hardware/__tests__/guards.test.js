const { assertBech32Prefix, validateFee, normalizePath } = require('../guards');
const { bech32 } = require('bech32');

describe('hardware guards (mobile)', () => {
  it('validates bech32 prefix', () => {
    const addr = bech32.encode('paw', bech32.toWords(new Uint8Array(20).fill(1)));
    expect(() => assertBech32Prefix(addr, 'paw')).not.toThrow();
    expect(() => assertBech32Prefix(addr, 'cosmos')).toThrow(/prefix mismatch/);
  });

  it('validates fee', () => {
    expect(() =>
      validateFee({ amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' })
    ).not.toThrow();
    expect(() =>
      validateFee({ amount: [{ denom: 'uatom', amount: '2500' }], gas: '200000' }, ['upaw'])
    ).toThrow(/not permitted/);
  });

  it('normalizes paths and enforces max account', () => {
    expect(normalizePath("m/44'/118'/0'/0/0")).toBe("44'/118'/0'/0/0");
    expect(() => normalizePath("m/44'/118'/5'/0/0", 4)).toThrow(/exceeds/);
  });
});
