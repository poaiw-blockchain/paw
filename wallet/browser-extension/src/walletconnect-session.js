import { ensureWalletConnectModal } from './popup';

export async function handleSessionProposal(request) {
  const summary = `WC Session Proposal:\nOrigin: ${request?.origin || 'unknown'}\nChain(s): ${request?.chains?.join(', ') || 'unknown'}`;
  const modal = ensureWalletConnectModal(summary);

  return new Promise(resolve => {
    const watcher = setInterval(() => {
      if (modal.dataset.action === 'approve') {
        clearInterval(watcher);
        resolve({ approved: true });
      } else if (modal.dataset.action === 'reject') {
        clearInterval(watcher);
        resolve({ approved: false, error: 'User rejected session' });
      }
    }, 100);
  });
}
