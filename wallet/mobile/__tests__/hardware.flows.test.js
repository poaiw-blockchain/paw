jest.mock('../src/services/PawAPI', () => ({
  getNodeInfo: jest.fn().mockResolvedValue({ default_node_info: { network: 'paw-mvp-1' } }),
  getAccount: jest.fn().mockResolvedValue({
    base_account: { account_number: '7', sequence: '9' },
  }),
}));

jest.mock('../src/services/hardware/guards', () => ({
  assertBech32Prefix: jest.fn(),
  validateFee: jest.fn(),
  normalizePath: jest.fn((p) => p),
}));

jest.mock('../src/services/LedgerHardwareSigner', () => ({
  signWithLedger: jest.fn().mockResolvedValue({
    signature: new Uint8Array([1, 2, 3]),
    publicKey: new Uint8Array([4, 5]),
    path: "m/44'/118'/0'/0/0",
  }),
}));

const PawAPI = require('../src/services/PawAPI');
const { signWithLedger } = require('../src/services/LedgerHardwareSigner');
const {
  signLedgerSend,
  signLedgerDelegate,
  signLedgerVote,
  signLedgerIbcTransfer,
} = require('../src/services/hardware/flows');

describe('hardware flow helpers', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('signs a send flow with live account metadata', async () => {
    const res = await signLedgerSend({
      fromAddress: 'paw1fromaddressxxxxxxxxxxxxxxxxxxxxx0jlzjq',
      toAddress: 'paw1toaddressxxxxxxxxxxxxxxxxxxxxxxxf6p48r',
      amount: '1000000',
      memo: 'hi',
    });
    expect(res.signature).toBeDefined();
    expect(signWithLedger).toHaveBeenCalledTimes(1);
    const signDoc = signWithLedger.mock.calls[0][0];
    expect(signDoc.chain_id).toBe('paw-mvp-1');
    expect(signDoc.msgs[0].type).toBe('cosmos-sdk/MsgSend');
    expect(signDoc.msgs[0].value.to_address).toContain('paw1toaddress');
  });

  it('signs a delegation flow', async () => {
    await signLedgerDelegate({
      delegatorAddress: 'paw1delegatorxxxxxxxxxxxxxxxxxxxxxxx7st0cm',
      validatorAddress: 'paw1validatorxxxxxxxxxxxxxxxxxxxxxxx6xjw6u',
      amount: '2500000',
    });
    const signDoc = signWithLedger.mock.calls[0][0];
    expect(signDoc.msgs[0].type).toBe('cosmos-sdk/MsgDelegate');
    expect(signDoc.msgs[0].value.validator_address).toContain('validator');
  });

  it('signs a governance vote flow', async () => {
    await signLedgerVote({
      voter: 'paw1voterxxxxxxxxxxxxxxxxxxxxxxxxxxxxxa25ykk',
      proposalId: '99',
      option: 'VOTE_OPTION_NO',
    });
    const signDoc = signWithLedger.mock.calls[0][0];
    expect(signDoc.msgs[0].type).toBe('cosmos-sdk/MsgVote');
    expect(signDoc.msgs[0].value.option).toBe('VOTE_OPTION_NO');
  });

  it('signs an IBC transfer flow', async () => {
    await signLedgerIbcTransfer({
      sender: 'paw1senderxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx0f6n5',
      receiver: 'paw1receiverxxxxxxxxxxxxxxxxxxxxxxxxxxy5g7vj',
      sourceChannel: 'channel-7',
      amount: '5000000',
    });
    const signDoc = signWithLedger.mock.calls[0][0];
    expect(signDoc.msgs[0].type).toBe('cosmos-sdk/MsgTransfer');
    expect(signDoc.msgs[0].value.source_channel).toBe('channel-7');
  });
});
