const DEFAULT_API_HOST = 'http://localhost:8545';
const WC_TAB_MAP = new Map(); // id -> tabId

chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.set({ apiHost: DEFAULT_API_HOST }, () => {
    console.log('PAW Wallet Miner: default API host stored');
  });
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.__wcForwarded) {
    // Already forwarded to popup; avoid loops.
    return;
  }

  if (message.type === 'getApiHost') {
    chrome.storage.local.get(['apiHost'], result => {
      sendResponse({ apiHost: result.apiHost || DEFAULT_API_HOST });
    });
    return true;
  }
  if (message.type === 'setApiHost') {
    chrome.storage.local.set({ apiHost: message.apiHost }, () => {
      sendResponse({ apiHost: message.apiHost });
    });
    return true;
  }

  // WalletConnect signing request: forward to popup (hardware-first routing lives there)
  if (message.type === 'walletconnect-sign') {
    if (sender?.tab?.id) {
      WC_TAB_MAP.set(message.id, sender.tab.id);
    }
    chrome.runtime.sendMessage({ ...message, __wcForwarded: true }, response => {
      sendResponse(response);
    });
    return true;
  }

  // WalletConnect session proposal (stub): forward to popup for UI
  if (message.type === 'walletconnect-session') {
    if (sender?.tab?.id) {
      WC_TAB_MAP.set(message.id, sender.tab.id);
    }
    chrome.runtime.sendMessage({ ...message, __wcForwarded: true }, response => {
      sendResponse(response);
    });
    return true;
  }

  // Forward WC results back to potential content script or dApp bridge
  if (message.type === 'walletconnect-sign-result' || message.type === 'walletconnect-session-result') {
    const tabId = WC_TAB_MAP.get(message.id);
    if (tabId !== undefined) {
      chrome.tabs.sendMessage(tabId, message);
      if (message.type === 'walletconnect-sign-result' || message.type === 'walletconnect-session-result') {
        WC_TAB_MAP.delete(message.id);
      }
    } else {
      chrome.runtime.sendMessage(message);
    }
    sendResponse?.({ delivered: tabId !== undefined });
    return true;
  }
});
