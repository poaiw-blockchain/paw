// Content script bridge: forwards WalletConnect requests to the extension and relays results back to the DApp.

const REQUEST_TYPES = new Set(['walletconnect-sign', 'walletconnect-session']);
const RESULT_TYPES = new Set(['walletconnect-sign-result', 'walletconnect-session-result']);

function normalizeRequest(event) {
  const raw = event.data?.request || event.data;
  const id = event.data?.id || raw?.id || `wc-${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const origin = event.origin || window.location.origin;

  if (event.data?.type === 'walletconnect-sign') {
    const params = raw?.params || raw?.parameters || [];
    const first = { ...(params[0] || {}) };
    if (!first.origin) {
      first.origin = origin;
    }
    return { id, request: { ...raw, id, params: [first] } };
  }

  if (event.data?.type === 'walletconnect-session') {
    return { id, request: { ...raw, id, origin: raw?.origin || origin } };
  }

  return { id, request: raw };
}

function forwardRequestToExtension(type, event) {
  const { id, request } = normalizeRequest(event);
  chrome.runtime.sendMessage({ type, id, request }, response => {
    const payload = response || {};
    window.postMessage(
      {
        type: `${type}-result`,
        id,
        result: { ...payload, status: payload?.error ? 'error' : payload?.status || 'ok' },
      },
      '*'
    );
  });
}

window.addEventListener('message', event => {
  if (!event.data || typeof event.data !== 'object') return;
  if (REQUEST_TYPES.has(event.data.type)) {
    forwardRequestToExtension(event.data.type, event);
  }
});

chrome.runtime.onMessage.addListener((message, _sender, _sendResponse) => {
  if (RESULT_TYPES.has(message.type)) {
    window.postMessage(message, '*');
  }
});
