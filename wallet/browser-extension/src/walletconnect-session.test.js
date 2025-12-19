import { describe, it, expect, vi } from 'vitest';
import { handleSessionProposal } from './walletconnect-session';

vi.mock('./popup', () => {
  const modal = { dataset: {}, style: { display: 'none' } };
  return {
    ensureWalletConnectModal: vi.fn(() => modal),
  };
});

describe('walletconnect session handler', () => {
  it('resolves approval when modal action set', async () => {
    const promise = handleSessionProposal({ origin: 'test' });
    const { ensureWalletConnectModal } = await import('./popup');
    const modal = ensureWalletConnectModal();
    setTimeout(() => {
      modal.dataset.action = 'approve';
    }, 10);
    const res = await promise;
    expect(res.approved).toBe(true);
  });
});
