import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { requestWalletConnectSign, requestWalletConnectSession } from './dapp-bridge';

const listeners = [];
if (typeof window === 'undefined') {
  // Minimal event shim for window
  // eslint-disable-next-line no-global-assign
  global.window = {
    addEventListener: (type, cb) => listeners.push({ type, cb }),
    removeEventListener: (type, cb) => {
      const idx = listeners.findIndex(l => l.type === type && l.cb === cb);
      if (idx >= 0) listeners.splice(idx, 1);
    },
    dispatchEvent: event => {
      listeners.filter(l => l.type === event.type).forEach(l => l.cb(event));
    },
    postMessage: payload => {
      const evt = { type: 'message', data: payload };
      listeners.filter(l => l.type === 'message').forEach(l => l.cb(evt));
    },
  };
}

describe('dapp bridge helpers', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('posts a sign request and resolves on matching result', async () => {
    const messages = [];
    vi.spyOn(window, 'postMessage').mockImplementation((payload) => {
      messages.push(payload);
    });

    const promise = requestWalletConnectSign({ params: [{ signerAddress: 'paw1abcd' }] }, { timeoutMs: 5000 });
    const sent = messages[0];

    // Simulate result from the extension
    window.dispatchEvent({
      type: 'message',
      data: { type: 'walletconnect-sign-result', id: sent.id, result: { signature: 'abc', publicKey: 'def' } },
    });

    const res = await promise;
    expect(res.result.signature).toBe('abc');
  });

  it('rejects when session result carries an error', async () => {
    const messages = [];
    vi.spyOn(window, 'postMessage').mockImplementation((payload) => messages.push(payload));

    const promise = requestWalletConnectSession({ chains: ['paw-testnet-1'] }, { timeoutMs: 5000 });
    const sent = messages[0];

    window.dispatchEvent({
      type: 'message',
      data: { type: 'walletconnect-session-result', id: sent.id, result: { approved: false, error: 'denied' } },
    });

    await expect(promise).rejects.toThrow(/denied/);
  });
});
