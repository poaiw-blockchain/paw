// Simple DApp bridge that listens for WC result events and exposes helpers to request sign/session flows.

const listeners = [];
let requestCounter = 0;
const PENDING = new Map();

export function onWalletConnectResult(cb) {
  listeners.push(cb);
}

function postAndWait(type, request, timeoutMs = 15000) {
  const id = request?.id || `wc-${Date.now()}-${requestCounter++}`;

  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      PENDING.delete(id);
      window.removeEventListener('message', handler);
      reject(new Error(`${type} timed out`));
    }, timeoutMs);

    const handler = event => {
      const data = event.data;
      if (!data || typeof data !== 'object') return;
      if ((data.type === 'walletconnect-sign-result' || data.type === 'walletconnect-session-result') && data.id === id) {
        clearTimeout(timer);
        window.removeEventListener('message', handler);
        PENDING.delete(id);
        if (data.result?.error) {
          reject(new Error(data.result.error));
          return;
        }
        resolve(data);
      }
    };

    window.addEventListener('message', handler);
    PENDING.set(id, handler);

    window.postMessage(
      {
        type,
        id,
        request: { ...request, id },
      },
      '*'
    );
  });
}

export function requestWalletConnectSign(request, { timeoutMs = 20000 } = {}) {
  return postAndWait('walletconnect-sign', request, timeoutMs);
}

export function requestWalletConnectSession(request, { timeoutMs = 20000 } = {}) {
  return postAndWait('walletconnect-session', request, timeoutMs);
}

window.addEventListener('message', event => {
  const data = event.data;
  if (!data || typeof data !== 'object') return;
  if (data.type === 'walletconnect-sign-result' || data.type === 'walletconnect-session-result') {
    listeners.forEach(cb => cb(data));
  }
});
