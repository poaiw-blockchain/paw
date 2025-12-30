import { describe, it, expect, vi, beforeEach } from 'vitest';
import { bech32 } from 'bech32';

// Use vi.hoisted to ensure the mock is created before hoisting
const { mockSignAmino } = vi.hoisted(() => ({
  mockSignAmino: vi.fn(),
}));

vi.mock('./hardware/ledger.js', () => ({
  signAmino: mockSignAmino,
}));

import {
  normalizeAddress,
  shouldUseHardware,
  signAminoRequest,
} from './walletconnect-hw.js';

describe('walletconnect hardware helpers', () => {
const hwAddress = bech32.encode('paw', bech32.toWords(new Uint8Array(20).fill(1)));
const hw = { address: hwAddress, transport: 'webhid' };

  beforeEach(() => {
    vi.clearAllMocks();
    mockSignAmino.mockResolvedValue({
      signature: new Uint8Array([1, 2, 3]),
      publicKey: new Uint8Array([4, 5, 6]),
      transportType: 'webhid',
    });
  });

  it('normalizes bech32 address', () => {
    expect(() => normalizeAddress(hw.address)).not.toThrow();
    expect(() => normalizeAddress('invalid')).toThrow();
  });

  it('detects hardware matching address', () => {
    expect(shouldUseHardware(hw.address, hw)).toBe(true);
    expect(shouldUseHardware('paw1deadbeefdeadbeefdeadbeefdeadbeefp9l4', hw)).toBe(false);
  });

  it('signs amino request with guardrails', async () => {
    const signDoc = {
      chain_id: 'paw-testnet-1',
      account_number: '1',
      sequence: '2',
      fee: { amount: [{ denom: 'upaw', amount: '2500' }], gas: '200000' },
      msgs: [{ '@type': '/cosmos.bank.v1beta1.MsgSend', from_address: hw.address, to_address: hw.address, amount: [{ denom: 'upaw', amount: '1' }] }],
      memo: '',
    };

    const res = await signAminoRequest({
      signDoc,
      address: hw.address,
      chainId: 'paw-testnet-1',
      hardwareState: hw,
    });

    expect(res.mode).toBe('hardware');
    expect(res.signature.length).toBeGreaterThan(0);
    expect(res.publicKey.length).toBeGreaterThan(0);
  });

  it('rejects chain-id mismatch', async () => {
    const signDoc = {
      chain_id: 'wrong-chain',
      fee: { amount: [{ denom: 'upaw', amount: '1' }], gas: '1' },
      msgs: [],
    };
    await expect(
      signAminoRequest({ signDoc, address: hw.address, chainId: 'paw-testnet-1', hardwareState: hw })
    ).rejects.toThrow(/Chain-id mismatch/);
  });
});
