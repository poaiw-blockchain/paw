jest.mock('../src/services/hardware', () => {
  return {
    LedgerServiceMobile: jest.fn().mockImplementation(() => ({
      connect: jest.fn().mockResolvedValue({ address: 'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnyq6ts', path: "44'/118'/0'/0/0", publicKey: new Uint8Array([9, 9]) }),
      signAmino: jest.fn().mockResolvedValue({ signature: new Uint8Array([1, 2, 3]) }),
      disconnect: jest.fn().mockResolvedValue(undefined),
    })),
  };
});

const { signWithLedger } = require('../src/services/LedgerHardwareSigner');
const { LedgerServiceMobile } = require('../src/services/hardware');

describe('LedgerHardwareSigner', () => {
  it('connects and signs with Ledger', async () => {
    const signDoc = {
      chain_id: 'paw-mvp-1',
      account_number: '1',
      sequence: '1',
      fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
      msgs: [{ type: 'cosmos-sdk/MsgSend', value: { from_address: 'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnyq6ts', to_address: 'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnyq6ts', amount: [{ denom: 'upaw', amount: '1' }] } }],
      memo: '',
    };

    const res = await signWithLedger(signDoc);
    expect(res.signature).toBeDefined();
    expect(res.publicKey).toBeDefined();
    expect(LedgerServiceMobile).toHaveBeenCalled();
  });

  it('rejects when signer address mismatches', async () => {
    const signDoc = {
      chain_id: 'paw-mvp-1',
      account_number: '1',
      sequence: '1',
      fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
      msgs: [{ type: 'cosmos-sdk/MsgSend', value: { from_address: 'paw1deadbeefdeadbeefdeadbeefdeadbeefp9l4', to_address: 'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnyq6ts', amount: [{ denom: 'upaw', amount: '1' }] } }],
      memo: '',
    };

    await expect(signWithLedger(signDoc)).rejects.toThrow(/does not match signer/);
  });
});
