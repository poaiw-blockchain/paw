import { describe, it, expect, vi, beforeEach } from 'vitest';
import { toBech32 } from '@cosmjs/encoding';
import {
  normalizePath,
  assertBech32Prefix,
  validateSignDocBasics,
  getLedgerAddress,
  signAmino,
} from './ledger';

const mockTransport = () => {
  const t = {
    setExchangeTimeout: vi.fn(),
    close: vi.fn().mockResolvedValue(undefined),
    device: { manufacturerName: 'Ledger', productName: 'Nano X' },
  };
  return t;
};

const mockApp = () => {
  const addr = toBech32('paw', new Uint8Array(20).fill(1));
  return {
    getAddress: vi.fn().mockResolvedValue({
      address: addr,
      publicKey: '02'.repeat(33),
    }),
    sign: vi.fn().mockResolvedValue({ signature: Buffer.from('deadbeef', 'hex').toString('base64') }),
  };
};

vi.mock('@ledgerhq/hw-transport-webhid', () => ({
  default: {
    create: vi.fn(async () => mockTransport()),
    isSupported: vi.fn(async () => true),
  },
}));

vi.mock('@ledgerhq/hw-transport-webusb', () => ({
  default: {
    create: vi.fn(async () => mockTransport()),
    isSupported: vi.fn(async () => true),
  },
}));

vi.mock('@ledgerhq/hw-app-cosmos', () => ({
  default: vi.fn().mockImplementation(() => mockApp()),
}));

describe('ledger hardware helpers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('normalizes valid paths and rejects out-of-range accounts', () => {
    expect(normalizePath("m/44'/118'/2'/0/0")).toBe("44'/118'/2'/0/0");
    expect(() => normalizePath("m/44'/118'/5'/0/0")).toThrow(/Account index exceeds/);
    expect(() => normalizePath('m/44/118/0/0/0')).toThrow(/must be hardened/);
  });

  it('validates bech32 prefixes', () => {
    const addr = toBech32('paw', new Uint8Array(20).fill(2));
    expect(() => assertBech32Prefix(addr, 'paw')).not.toThrow();
    expect(() => assertBech32Prefix(addr, 'cosmos')).toThrow(/prefix mismatch/);
  });

  it('validates sign doc basics', () => {
    expect(() =>
      validateSignDocBasics(
        {
          chain_id: 'paw-testnet-1',
          fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
        },
        { enforceChainId: 'paw-testnet-1', allowedFeeDenoms: ['upaw'] }
      )
    ).not.toThrow();

    expect(() =>
      validateSignDocBasics(
        { chain_id: 'wrong', fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' } },
        { enforceChainId: 'paw-testnet-1', allowedFeeDenoms: ['upaw'] }
      )
    ).toThrow(/chain-id mismatch/);
  });

  it('connects via WebHID/WebUSB to fetch address', async () => {
    const res = await getLedgerAddress();
    expect(res.address).toMatch(/^paw1/);
    expect(res.publicKey.length).toBeGreaterThan(0);
  });

  it('signs amino doc with guardrails', async () => {
    const signDoc = {
      chain_id: 'paw-testnet-1',
      account_number: '1',
      sequence: '1',
      fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
      msgs: [],
      memo: '',
    };
    const res = await signAmino({ signDoc, enforceChainId: 'paw-testnet-1' });
    expect(res.signature.length).toBeGreaterThan(0);
    expect(res.publicKey.length).toBeGreaterThan(0);
  });
});
