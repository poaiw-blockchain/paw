import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { requestWalletConnectSign, requestWalletConnectSession } from './dapp-bridge';

// Helper to dispatch a message event (jsdom requires actual Event objects)
function dispatchMessage(data) {
  const event = new MessageEvent('message', { data });
  window.dispatchEvent(event);
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
    dispatchMessage({ type: 'walletconnect-sign-result', id: sent.id, result: { signature: 'abc', publicKey: 'def' } });

    const res = await promise;
    expect(res.result.signature).toBe('abc');
  });

  it('rejects when session result carries an error', async () => {
    const messages = [];
    vi.spyOn(window, 'postMessage').mockImplementation((payload) => messages.push(payload));

    const promise = requestWalletConnectSession({ chains: ['paw-testnet-1'] }, { timeoutMs: 5000 });
    const sent = messages[0];

    dispatchMessage({ type: 'walletconnect-session-result', id: sent.id, result: { approved: false, error: 'denied' } });

    await expect(promise).rejects.toThrow(/denied/);
  });
});
