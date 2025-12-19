// Import Cosmos SDK module
/* global COSMOS_SDK */
import { DirectSecp256k1Wallet } from '@cosmjs/proto-signing';
import { Secp256k1Wallet } from '@cosmjs/amino';
import { fromHex } from '@cosmjs/encoding';
import { getLedgerAddress, signAmino, validateSignDocBasics, assertBech32Prefix } from './hardware/ledger.js';
import { signAminoRequest } from './walletconnect-hw.js';
import { handleSessionProposal } from './walletconnect-session.js';

const API_KEY = 'apiHost';
const PRIVATE_KEY_STORAGE = 'walletPrivateKey';
const HARDWARE_WALLET_KEY = 'hardwareWallet';
const SESSION_TOKEN_KEY = 'walletSessionToken';
const SESSION_SECRET_KEY = 'walletSessionSecret';
const SESSION_ADDRESS_KEY = 'walletSessionAddress';
const HKDF_INFO = new TextEncoder().encode('walletconnect-trade');
const WC_PENDING_SIGN = {};
const WC_ALLOWLIST_KEY = 'wcAllowedOrigins';
const WC_AUDIT_LOG_KEY = 'wcAuditLog';
const WC_DEFAULT_ALLOWLIST = ['https://trusted-dapp.example'];
const WC_MODAL_ID = 'wcModal';
const WC_RATE_LIMIT = new Map();
const WC_RATE_WINDOW_MS = 60000;
const WC_RATE_LIMIT_MAX = 5;
const PASSKEY_STORAGE_KEY = 'wcPasskeyId';
const BLOCKED_ORIGIN_PATTERNS = [/\\.onion$/i, /phish/i, /malware/i];

function escapeHtml(value) {
  if (value === null || value === undefined) {
    return '';
  }
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function html(strings, ...values) {
  return strings.reduce((acc, string, index) => {
    const rawValue = index < values.length ? values[index] : '';
    return acc + string + escapeHtml(rawValue);
  }, '');
}

function bufferToHex(buffer) {
  return Array.from(new Uint8Array(buffer))
    .map(byte => byte.toString(16).padStart(2, '0'))
    .join('');
}

function hexToBytes(hex) {
  const bytes = [];
  for (let c = 0; c < hex.length; c += 2) {
    bytes.push(parseInt(hex.substr(c, 2), 16));
  }
  return new Uint8Array(bytes);
}

function bufferToBase64(buffer) {
  return btoa(String.fromCharCode(...new Uint8Array(buffer)));
}

function base64ToBytes(base64) {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

function stableStringify(value) {
  if (value === null) {return 'null';}
  if (Array.isArray(value)) {
    return `[${value.map(item => stableStringify(item)).join(',')}]`;
  }
  if (typeof value === 'object') {
    const keys = Object.keys(value).sort();
    return `{${keys.map(key => `"${key}":${stableStringify(value[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

function enforceOriginHealth(origin) {
  if (!origin) throw new Error('WalletConnect request missing origin');
  try {
    const url = new URL(origin);
    const isLocal = ['localhost', '127.0.0.1'].includes(url.hostname);
    if (!isLocal && url.protocol !== 'https:') {
      throw new Error('WalletConnect origin must use https');
    }
    if (BLOCKED_ORIGIN_PATTERNS.some(pattern => pattern.test(origin))) {
      throw new Error('WalletConnect origin failed safety heuristics');
    }
  } catch (err) {
    if (err instanceof TypeError) {
      throw new Error('WalletConnect origin is malformed');
    }
    throw err;
  }
}

function enforceRateLimit(origin) {
  const now = Date.now();
  const entries = WC_RATE_LIMIT.get(origin) || [];
  const recent = entries.filter(ts => now - ts < WC_RATE_WINDOW_MS);
  if (recent.length >= WC_RATE_LIMIT_MAX) {
    throw new Error('Signing limited: too many requests from this origin');
  }
  recent.push(now);
  WC_RATE_LIMIT.set(origin, recent);
}

function detectSignMode(signDoc) {
  if (!signDoc) return 'amino';
  if (signDoc.bodyBytes || signDoc.body_bytes || signDoc.authInfoBytes || signDoc.auth_info_bytes) {
    return 'direct';
  }
  return 'amino';
}

function normalizeBytes(value) {
  if (value instanceof Uint8Array) return value;
  if (value instanceof ArrayBuffer) return new Uint8Array(value);
  if (typeof value === 'string') return base64ToBytes(value);
  throw new Error('Unable to normalize byte data for signing');
}

function validateMsgAddresses(signDoc, prefix) {
  const msgs = signDoc?.msgs || signDoc?.messages || [];
  msgs.forEach(msg => {
    const value = msg.value || msg;
    if (value && typeof value === 'object') {
      Object.entries(value).forEach(([key, val]) => {
        if (typeof val === 'string' && key.toLowerCase().includes('address')) {
          assertBech32Prefix(val, prefix);
        }
      });
    }
  });
}

async function getStoredPasskeyId() {
  return new Promise(resolve => {
    chrome.storage.local.get([PASSKEY_STORAGE_KEY], result => {
      resolve(result[PASSKEY_STORAGE_KEY] || null);
    });
  });
}

async function setStoredPasskeyId(id) {
  return new Promise(resolve => {
    chrome.storage.local.set({ [PASSKEY_STORAGE_KEY]: id }, () => resolve());
  });
}

async function ensurePasskeyEnrollment() {
  const existing = await getStoredPasskeyId();
  if (existing) return existing;
  if (!navigator.credentials || !window.PublicKeyCredential) {
    throw new Error('Passkeys are required for software signing');
  }

  const challenge = crypto.getRandomValues(new Uint8Array(32));
  const userId = crypto.getRandomValues(new Uint8Array(16));
  const credential = await navigator.credentials.create({
    publicKey: {
      challenge,
      rp: { name: 'PAW Wallet' },
      user: {
        id: userId,
        name: 'paw-user',
        displayName: 'PAW Wallet',
      },
      pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
      authenticatorSelection: { userVerification: 'preferred' },
      timeout: 30000,
    },
  });

  if (!credential?.rawId) {
    throw new Error('Passkey enrollment failed');
  }

  const id = bufferToBase64(credential.rawId);
  await setStoredPasskeyId(id);
  return id;
}

async function requirePasskey() {
  const id = await ensurePasskeyEnrollment();
  const challenge = crypto.getRandomValues(new Uint8Array(32));
  const assertion = await navigator.credentials.get({
    publicKey: {
      timeout: 30000,
      challenge,
      allowCredentials: [
        {
          type: 'public-key',
          id: Uint8Array.from(atob(id), c => c.charCodeAt(0)),
        },
      ],
      userVerification: 'preferred',
    },
  });
  if (!assertion?.response) {
    throw new Error('Passkey verification failed');
  }
}

async function getActiveWalletAddress(preferredAddress) {
  if (preferredAddress) {
    return preferredAddress.trim();
  }
  const inputVal = $('#walletAddress')?.value?.trim();
  if (inputVal) {
    return inputVal;
  }
  return new Promise(resolve => {
    chrome.storage.local.get(['walletAddress', HARDWARE_WALLET_KEY], result => {
      if (result.walletAddress) {
        resolve(result.walletAddress);
        return;
      }
      const hw = result[HARDWARE_WALLET_KEY];
      resolve(hw?.address || null);
    });
  });
}

async function signPayload(payloadStr, secretHex) {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    'raw',
    hexToBytes(secretHex),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const sig = await crypto.subtle.sign('HMAC', key, encoder.encode(payloadStr));
  return bufferToHex(sig);
}

async function saveSession(token, secret, address) {
  return new Promise(resolve => {
    chrome.storage.local.set(
      {
        [SESSION_TOKEN_KEY]: token,
        [SESSION_SECRET_KEY]: secret,
        [SESSION_ADDRESS_KEY]: address,
      },
      () => resolve()
    );
  });
}

async function getSession() {
  return new Promise(resolve => {
    chrome.storage.local.get(
      [SESSION_TOKEN_KEY, SESSION_SECRET_KEY, SESSION_ADDRESS_KEY],
      result => {
        resolve({
          sessionToken: result[SESSION_TOKEN_KEY],
          sessionSecret: result[SESSION_SECRET_KEY],
          walletAddress: result[SESSION_ADDRESS_KEY],
        });
      }
    );
  });
}

async function registerSession(address) {
  const host = await getApiHost();
  const response = await fetch(`${host}/wallet-trades/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ wallet_address: address }),
  });
  const payload = await response.json();
  if (payload.success) {
    await saveSession(payload.session_token, payload.session_secret, address);
    return {
      sessionToken: payload.session_token,
      sessionSecret: payload.session_secret,
      walletAddress: address,
    };
  }
  return null;
}

// eslint-disable-next-line no-unused-vars
async function beginWalletConnectHandshake(address) {
  const host = await getApiHost();
  const response = await fetch(`${host}/wallet-trades/wc/handshake`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ wallet_address: address }),
  });
  const payload = await response.json();
  return payload.success ? payload : null;
}

// eslint-disable-next-line no-unused-vars
async function confirmWalletConnectHandshake(address, handshake) {
  const allowlist = await loadWalletConnectAllowlist();
  if (!handshake?.origin || !allowlist.includes(handshake.origin)) {
    throw new Error('WalletConnect origin not allowed');
  }

  const host = await getApiHost();
  const clientKeyPair = await crypto.subtle.generateKey(
    { name: 'ECDH', namedCurve: 'P-256' },
    true,
    ['deriveBits']
  );
  const clientPublicRaw = await crypto.subtle.exportKey('raw', clientKeyPair.publicKey);
  const serverPublicBytes = base64ToBytes(handshake.server_public);
  const serverKey = await crypto.subtle.importKey(
    'raw',
    serverPublicBytes,
    { name: 'ECDH', namedCurve: 'P-256' },
    false,
    []
  );
  const sharedBits = await crypto.subtle.deriveBits(
    { name: 'ECDH', public: serverKey },
    clientKeyPair.privateKey,
    256
  );
  const hkdfKey = await crypto.subtle.importKey(
    'raw',
    sharedBits,
    { name: 'HKDF', hash: 'SHA-256' },
    false,
    ['deriveBits']
  );
  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: 'SHA-256',
      salt: new TextEncoder().encode(handshake.handshake_id),
      info: HKDF_INFO,
    },
    hkdfKey,
    256
  );
  const derivedHex = bufferToHex(derivedBits);
  const clientPublicBase64 = bufferToBase64(clientPublicRaw);
  const response = await fetch(`${host}/wallet-trades/wc/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      handshake_id: handshake.handshake_id,
      wallet_address: address,
      client_public: clientPublicBase64,
    }),
  });
  const payload = await response.json();
  if (!payload.success) {return null;}
  await saveSession(payload.session_token, derivedHex, address);
  return { sessionToken: payload.session_token, sessionSecret: derivedHex, walletAddress: address };
}

// eslint-disable-next-line no-unused-vars
async function handleWalletConnectSignRequest(request) {
  const { params } = request;
  const [signParams] = params || [];
  const signDoc = signParams?.signDoc || signParams?.sign_doc || signParams;
  const origin = signParams?.origin || request.origin || 'unknown';
  enforceOriginHealth(origin);
  enforceRateLimit(origin);
  const allowlist = await loadWalletConnectAllowlist();
  const hardwareState = await getHardwareWallet();
  const chainId = signDoc?.chain_id || COSMOS_SDK.config.chainId || COSMOS_SDK.config?.chain_id || 'paw-testnet-1';
  const address = await getActiveWalletAddress(
    signParams?.signerAddress || signParams?.signer_address || hardwareState?.address
  );
  const signMode = detectSignMode(signDoc);
  const bech32Prefix = address?.split('1')[0] || COSMOS_SDK.config.bech32Prefix || 'paw';

  if (origin && !allowlist.includes(origin)) {
    await recordWcAudit({
      id: request.id,
      type: 'sign',
      origin,
      mode: 'blocked',
      chainId,
      address: address || 'n/a',
      timestamp: Date.now(),
      result: 'blocked',
    });
    throw new Error('Origin not allowlisted for WalletConnect signing');
  }

  if (!address) {
    throw new Error('Address required for signing');
  }
  assertBech32Prefix(address, bech32Prefix);

  const mode = hardwareState?.address ? 'hardware' : 'software';
  if (hardwareState) {
    window.__hwState = hardwareState;
  }

  try {
    if (mode === 'hardware') {
      if (signMode === 'direct') {
        throw new Error('Direct sign requests must use the software signer; request amino for hardware wallets.');
      }
      const res = await signAminoRequest({
        signDoc: { ...signDoc, chain_id: chainId },
        address,
        chainId,
        hardwareState,
      });
      WC_PENDING_SIGN[request.id] = { mode: 'hardware', transport: res.transport || hardwareState.transport };
      await recordWcAudit({
        id: request.id,
        type: 'sign',
        origin: origin || 'unknown',
        mode: 'hardware-amino',
        chainId,
        address,
        timestamp: Date.now(),
        result: 'ok',
      });
      chrome.runtime.sendMessage({
        type: 'walletconnect-sign-result',
        id: request.id,
        result: {
          signature: COSMOS_SDK.bytesToBase64(res.signature),
          publicKey: COSMOS_SDK.bytesToBase64(res.publicKey),
          mode: 'amino',
          status: 'ok',
        },
      });
      return {
        signature: COSMOS_SDK.bytesToBase64(res.signature),
        publicKey: COSMOS_SDK.bytesToBase64(res.publicKey),
        mode: 'amino',
        status: 'ok',
      };
    }

    await requirePasskey();
    validateSignDocBasics(
      { chain_id: chainId, fee: signDoc?.fee },
      { enforceChainId: chainId, allowedFeeDenoms: ['upaw'] }
    );
    validateMsgAddresses(signDoc, bech32Prefix);

    const privateKeyHex = await getPrivateKey();
    if (!privateKeyHex) {
      throw new Error('No signer available (connect Ledger or import key)');
    }

    const signed =
      signMode === 'direct'
        ? await signWithSoftwareDirect(signDoc, address, chainId, privateKeyHex, bech32Prefix)
        : await signWithSoftwareAmino(signDoc, address, chainId, privateKeyHex, bech32Prefix);

    WC_PENDING_SIGN[request.id] = { mode: signMode === 'direct' ? 'software-direct' : 'software' };
    await recordWcAudit({
      id: request.id,
      type: 'sign',
      origin: origin || 'unknown',
      mode: signMode === 'direct' ? 'software-direct' : 'software',
      chainId,
      address,
      timestamp: Date.now(),
      result: 'ok',
    });
    chrome.runtime.sendMessage({
      type: 'walletconnect-sign-result',
      id: request.id,
      result: {
        signature: COSMOS_SDK.bytesToBase64(signed.signature),
        publicKey: COSMOS_SDK.bytesToBase64(signed.publicKey),
        mode: signMode,
        status: 'ok',
      },
    });
    return {
      signature: COSMOS_SDK.bytesToBase64(signed.signature),
      publicKey: COSMOS_SDK.bytesToBase64(signed.publicKey),
      mode: signMode,
      status: 'ok',
    };
  } catch (err) {
    await recordWcAudit({
      id: request.id,
      type: 'sign',
      origin: origin || 'unknown',
      mode,
      chainId,
      address,
      timestamp: Date.now(),
      result: 'error',
      error: err.message || 'signing failed',
    });
    chrome.runtime.sendMessage({
      type: 'walletconnect-sign-result',
      id: request.id,
      result: { error: err.message || 'WalletConnect signing failed', status: 'error' },
    });
    throw err;
  }
}

async function signWithSoftwareAmino(signDoc, address, chainId, privateKeyHex, prefix) {
  const signer = await Secp256k1Wallet.fromKey(fromHex(privateKeyHex), prefix);
  const res = await signer.signAmino(address, { ...signDoc, chain_id: chainId });
  const accounts = await signer.getAccounts();
  const pubkeyFromSignature = res.signature?.pub_key?.value
    ? base64ToBytes(res.signature.pub_key.value)
    : null;
  const signatureBytes = res.signature?.signature
    ? base64ToBytes(res.signature.signature)
    : base64ToBytes(res.signature);
  return {
    signature: signatureBytes,
    publicKey: pubkeyFromSignature || accounts[0]?.pubkey || new Uint8Array(),
  };
}

async function signWithSoftwareDirect(signDoc, address, chainId, privateKeyHex, prefix) {
  if (!signDoc?.account_number && !signDoc?.accountNumber) {
    throw new Error('account_number is required for direct signing');
  }
  const signer = await DirectSecp256k1Wallet.fromKey(fromHex(privateKeyHex), prefix);
  const bodyBytes = normalizeBytes(signDoc.bodyBytes || signDoc.body_bytes);
  const authBytes = normalizeBytes(signDoc.authInfoBytes || signDoc.auth_info_bytes);
  const res = await signer.signDirect(address, {
    bodyBytes,
    authInfoBytes: authBytes,
    chainId,
    accountNumber: BigInt(signDoc.accountNumber ?? signDoc.account_number ?? 0),
  });
  const accounts = await signer.getAccounts();
  return {
    signature: res.signature.signature,
    publicKey: accounts[0]?.pubkey || new Uint8Array(),
  };
}

// Expose WC handler for future UI wiring / message passing
if (typeof window !== 'undefined') {
  window.handleWalletConnectSignRequest = async (req) => {
    try {
      return await handleWalletConnectSignRequest(req);
    } catch (err) {
      return { error: err.message || 'WalletConnect signing failed' };
    }
  };
}

export {
  detectSignMode,
  enforceOriginHealth,
  enforceRateLimit,
  normalizeBytes,
};

function summarizeSignRequest(request) {
  const params = request?.params?.[0] || {};
  const addr = params.signerAddress || params.signer_address || 'unknown';
  const fee = params.signDoc?.fee || params.sign_doc?.fee || params.fee || {};
  const chain = params.signDoc?.chain_id || params.sign_doc?.chain_id || 'unknown';
  const msgs = params.signDoc?.msgs || params.sign_doc?.msgs || params.msgs || [];
  const feeCoin = (fee.amount && fee.amount[0]) || {};
  return [
    `Origin: ${params.origin || 'unknown'}`,
    `Signer: ${addr}`,
    `Chain-ID: ${chain}`,
    `Msgs: ${msgs.length}`,
    `Fee: ${feeCoin.amount || '0'} ${feeCoin.denom || ''}`,
    `Gas: ${fee.gas || 'n/a'}`,
  ].join('\n');
}

export function ensureWalletConnectModal(summary) {
  let modal = document.getElementById(WC_MODAL_ID);
  if (!modal) {
    const tpl = document.getElementById('wcModalTemplate');
    if (tpl && tpl.innerHTML) {
      document.body.insertAdjacentHTML('beforeend', tpl.innerHTML);
      modal = document.getElementById(WC_MODAL_ID);
    }
  }
  if (!modal) {
    // Fallback inline modal
    modal = document.createElement('div');
    modal.id = WC_MODAL_ID;
    modal.style.position = 'fixed';
    modal.style.top = '0';
    modal.style.left = '0';
    modal.style.right = '0';
    modal.style.bottom = '0';
    modal.style.background = 'rgba(0,0,0,0.45)';
    modal.style.display = 'flex';
    modal.style.alignItems = 'center';
    modal.style.justifyContent = 'center';
    modal.style.zIndex = '9999';
    modal.innerHTML = `
      <div style="background:#fff;padding:16px;border-radius:8px;width:320px;box-shadow:0 2px 10px rgba(0,0,0,0.2);">
        <h3 style="margin-top:0;">WalletConnect Request</h3>
        <pre id="wcSummary" style="white-space:pre-wrap;font-size:12px;background:#f7f7f7;padding:8px;border-radius:6px;max-height:200px;overflow:auto;"></pre>
        <div style="display:flex;flex-direction:column;gap:4px;font-size:12px;margin-top:8px;">
          <div id="wcHardwareStatus"></div>
          <div id="wcAllowlistInfo"></div>
          <div id="wcStatus" style="color:#555;"></div>
        </div>
        <div style="display:flex;justify-content:flex-end;gap:8px;margin-top:12px;">
          <button id="wcReject">Reject</button>
          <button id="wcApprove" style="background:#0070f3;color:#fff;border:none;padding:6px 10px;border-radius:4px;">Approve</button>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
  }

  const approve = modal.querySelector('#wcApprove');
  const reject = modal.querySelector('#wcReject');
  if (approve && !approve.dataset.bound) {
    approve.addEventListener('click', () => {
      modal.dataset.action = 'approve';
      modal.style.display = 'none';
    });
    approve.dataset.bound = '1';
  }
  if (reject && !reject.dataset.bound) {
    reject.addEventListener('click', () => {
      modal.dataset.action = 'reject';
      modal.style.display = 'none';
    });
    reject.dataset.bound = '1';
  }

  const summaryEl = modal.querySelector('#wcSummary');
  if (summaryEl) summaryEl.textContent = summary;
  const hwEl = modal.querySelector('#wcHardwareStatus');
  const allowEl = modal.querySelector('#wcAllowlistInfo');
  const statusEl = modal.querySelector('#wcStatus');
  if (hwEl) {
    const hw = window.__hwState || {};
    hwEl.textContent = hw.address ? `Hardware: ${hw.address.slice(0, 8)}... (${hw.transport || 'unknown'})` : 'Hardware: none';
  }
  if (allowEl) {
    allowEl.textContent = 'Origin must be allowlisted';
  }
  if (statusEl) {
    statusEl.textContent = 'Awaiting approval';
    statusEl.style.color = '#555';
  }

  modal.dataset.action = '';
  modal.style.display = 'flex';
  return modal;
}

function setWcModalStatus(modal, text, isError = false) {
  if (!modal) return;
  const statusEl = modal.querySelector('#wcStatus');
  if (statusEl) {
    statusEl.textContent = text;
    statusEl.style.color = isError ? '#c00' : '#2d7a2d';
  }
}

async function ensureSession() {
  const address = $('#walletAddress').value.trim();
  if (!address) {return null;}
  const session = await getSession();
  if (
    session &&
    session.walletAddress === address &&
    session.sessionToken &&
    session.sessionSecret
  ) {
    return session;
  }
  return registerSession(address);
}

async function getApiHost() {
  return new Promise(resolve => {
    chrome.storage.local.get([API_KEY], result => {
      resolve(result[API_KEY] || 'http://localhost:1317');
    });
  });
}

/**
 * Cosmos SDK Integration Functions
 */

async function getPrivateKey() {
  return new Promise(resolve => {
    chrome.storage.local.get([PRIVATE_KEY_STORAGE], result => {
      resolve(result[PRIVATE_KEY_STORAGE] || null);
    });
  });
}

async function savePrivateKey(privateKeyHex) {
  return new Promise(resolve => {
    chrome.storage.local.set({ [PRIVATE_KEY_STORAGE]: privateKeyHex }, () => resolve());
  });
}

async function generateNewWallet() {
  try {
    // Confirm wallet creation
    const existingKey = await getPrivateKey();
    if (existingKey) {
      const confirmed = confirm(
        'You already have a wallet. Creating a new one will replace it. ' +
        'Make sure you have backed up your current private key! Continue?'
      );
      if (!confirmed) {
        return;
      }
    }

    showMessage('walletMessage', 'Generating new wallet...');

    const privateKey = COSMOS_SDK.generatePrivateKey();
    if (privateKey.length !== 32) {
      throw new Error('Invalid private key length');
    }

    const privateKeyHex = COSMOS_SDK.bytesToHex(privateKey);
    const publicKey = await COSMOS_SDK.getPublicKey(privateKey);
    const address = COSMOS_SDK.publicKeyToAddress(publicKey);

    if (!validateCosmosAddress(address)) {
      throw new Error('Generated address validation failed');
    }

    await savePrivateKey(privateKeyHex);
    $('#walletAddress').value = address;
    chrome.storage.local.set({ walletAddress: address });

    showMessage('walletMessage', `New wallet created: ${address}`);

    // Show backup warning
    setTimeout(() => {
      alert(
        'IMPORTANT: Back up your private key!\n\n' +
        'Use the "Export Private Key" button to view and save your private key. ' +
        'Without it, you cannot recover your wallet if you lose access to this browser.'
      );
    }, 500);

    await updateBalance();
  } catch (error) {
    showMessage('walletMessage', `Error creating wallet: ${error.message}`, true);
    console.error('Wallet generation error:', error);
  }
}

async function importWallet(privateKeyHex) {
  try {
    // Validate input
    if (!privateKeyHex || typeof privateKeyHex !== 'string') {
      throw new Error('Private key is required');
    }

    // Remove whitespace and validate hex format
    privateKeyHex = privateKeyHex.trim().toLowerCase();
    if (!/^[0-9a-f]{64}$/i.test(privateKeyHex)) {
      throw new Error('Invalid private key format. Must be 64 hex characters (32 bytes)');
    }

    showMessage('walletMessage', 'Importing wallet...');

    const privateKey = COSMOS_SDK.hexToBytes(privateKeyHex);
    if (privateKey.length !== 32) {
      throw new Error('Invalid private key length. Must be 32 bytes');
    }

    const publicKey = await COSMOS_SDK.getPublicKey(privateKey);
    const address = COSMOS_SDK.publicKeyToAddress(publicKey);

    if (!validateCosmosAddress(address)) {
      throw new Error('Generated address validation failed');
    }

    // Confirm if overwriting existing wallet
    const existingKey = await getPrivateKey();
    if (existingKey && existingKey !== privateKeyHex) {
      const confirmed = confirm(
        'You already have a wallet. Importing will replace it. ' +
        'Make sure you have backed up your current private key! Continue?'
      );
      if (!confirmed) {
        return;
      }
    }

    await savePrivateKey(privateKeyHex);
    $('#walletAddress').value = address;
    chrome.storage.local.set({ walletAddress: address });

    showMessage('walletMessage', `Wallet imported successfully: ${address}`);
    await updateBalance();
  } catch (error) {
    showMessage('walletMessage', `Error importing wallet: ${error.message}`, true);
    console.error('Wallet import error:', error);
  }
}

async function updateBalance() {
  const address = $('#walletAddress').value.trim();
  if (!address) {
    $('#balanceDisplay').textContent = 'Balance: Enter address';
    return;
  }

  try {
    const balances = await COSMOS_SDK.getBalance(address);
    if (balances.length === 0) {
      $('#balanceDisplay').textContent = 'Balance: 0 PAW';
      return;
    }

    const balanceText = balances
      .map(b => {
        const amount = parseInt(b.amount) / Math.pow(10, COSMOS_SDK.config.coinDecimals);
        const denom = b.denom === 'upaw' ? 'PAW' : b.denom;
        return `${amount} ${denom}`;
      })
      .join(', ');

    $('#balanceDisplay').textContent = `Balance: ${balanceText}`;
  } catch (error) {
    $('#balanceDisplay').textContent = `Balance: Error - ${error.message}`;
  }
}

async function sendTokens(toAddress, amount, denom = 'upaw') {
  const fromAddress = $('#walletAddress').value.trim();

  // Validation
  if (!fromAddress) {
    showMessage('transactionMessage', 'Please enter your wallet address', true);
    return null;
  }

  if (!validateCosmosAddress(fromAddress)) {
    showMessage('transactionMessage', 'Invalid sender address format', true);
    return null;
  }

  if (!validateCosmosAddress(toAddress)) {
    showMessage('transactionMessage', 'Invalid recipient address format', true);
    return null;
  }

  if (!amount || amount <= 0) {
    showMessage('transactionMessage', 'Invalid amount. Must be greater than 0', true);
    return null;
  }

  try {
    showMessage('transactionMessage', 'Preparing transaction...');

    const amountInMicroDenom = Math.floor(amount * Math.pow(10, COSMOS_SDK.config.coinDecimals));

    if (amountInMicroDenom <= 0) {
      throw new Error('Amount too small to send');
    }

    const tx = COSMOS_SDK.buildTransferTx({
      fromAddress,
      toAddress,
      amount: amountInMicroDenom,
      denom,
      memo: 'Sent from PAW Browser Wallet',
    });

    const result = await signAndBroadcastTx(tx, fromAddress, 'transactionMessage');
    showMessage('transactionMessage', `Transaction successful! Hash: ${result.txhash}`);
    await updateBalance();
    await refreshTradeHistory();
    return result;
  } catch (error) {
    const errorMsg = error.message || 'Unknown error occurred';
    showMessage('transactionMessage', `Transaction failed: ${errorMsg}`, true);
    console.error('Send tokens error:', error);
    return null;
  }
}

async function executeSwap(poolId, tokenInDenom, tokenInAmount, tokenOutDenom, minAmountOut) {
  const sender = $('#walletAddress').value.trim();

  // Validation
  if (!sender) {
    showMessage('tradeMessage', 'Please enter your wallet address', true);
    return null;
  }

  if (!validateCosmosAddress(sender)) {
    showMessage('tradeMessage', 'Invalid wallet address format', true);
    return null;
  }

  if (!poolId || poolId <= 0) {
    showMessage('tradeMessage', 'Invalid pool ID', true);
    return null;
  }

  if (!tokenInAmount || tokenInAmount <= 0) {
    showMessage('tradeMessage', 'Invalid swap amount. Must be greater than 0', true);
    return null;
  }

  if (!tokenInDenom || !tokenOutDenom) {
    showMessage('tradeMessage', 'Token denominations required', true);
    return null;
  }

  try {
    showMessage('tradeMessage', 'Preparing swap transaction...');

    const tx = COSMOS_SDK.buildSwapTx({
      sender,
      poolId,
      tokenIn: {
        denom: tokenInDenom,
        amount: tokenInAmount.toString(),
      },
      tokenOutDenom,
      minAmountOut: minAmountOut.toString(),
      memo: 'DEX Swap from PAW Browser Wallet',
    });

    const result = await signAndBroadcastTx(tx, sender, 'tradeMessage');

    showMessage('tradeMessage', `Swap successful! Hash: ${result.txhash}`);
    await updateBalance();
    await refreshPools();
    await refreshTradeHistory();
    return result;
  } catch (error) {
    const errorMsg = error.message || 'Unknown swap error occurred';
    showMessage('tradeMessage', `Swap failed: ${errorMsg}`, true);
    console.error('Swap execution error:', error);
    return null;
  }
}

function showMessage(elementId, message, isError = false) {
  const element = $(`#${elementId}`);
  if (element) {
    element.textContent = message;
    element.classList.toggle('error', isError);
    setTimeout(() => {
      element.textContent = '';
      element.classList.remove('error');
    }, 10000);
  }
}

function updateHardwareStatus(message, isError = false) {
  const el = document.getElementById('hardwareStatus');
  if (el) {
    el.textContent = message;
    el.className = isError ? 'error' : 'success';
  }
}

async function saveHardwareWallet(info) {
  return new Promise(resolve => {
    chrome.storage.local.set({ [HARDWARE_WALLET_KEY]: info }, () => resolve());
  });
}

async function getHardwareWallet() {
  return new Promise(resolve => {
    chrome.storage.local.get([HARDWARE_WALLET_KEY], result => {
      resolve(result[HARDWARE_WALLET_KEY] || null);
    });
  });
}

async function clearHardwareWallet() {
  return new Promise(resolve => {
    chrome.storage.local.remove([HARDWARE_WALLET_KEY], () => resolve());
  });
}

function injectHardwareControls() {
  if (document.getElementById('hardwareControls')) return;

  const target =
    document.getElementById('walletMessage')?.parentElement ||
    document.getElementById('walletSection') ||
    document.body;

  const container = document.createElement('div');
  container.id = 'hardwareControls';
  container.style.marginTop = '12px';
  container.innerHTML = `
    <div style="display:flex; flex-direction:column; gap:6px; border:1px solid #eee; padding:8px; border-radius:6px;">
      <strong>Hardware Wallet (Ledger)</strong>
      <label style="font-size:12px;">Account index (0-4): <input id="ledgerAccountIndex" type="number" min="0" max="4" value="0" style="width:60px;margin-left:4px;"></label>
      <button id="connectLedgerHw" style="padding:6px 8px;">Connect Ledger (WebHID/WebUSB)</button>
      <span id="hardwareStatus" class="muted">Not connected</span>
    </div>
  `;

  target.appendChild(container);

  const connectBtn = document.getElementById('connectLedgerHw');
  if (connectBtn) {
    connectBtn.addEventListener('click', connectLedgerHardware);
  }

  // WalletConnect prompt placeholders (if present in DOM)
  const wcPrompt = document.getElementById('wcPrompt');
  const wcApproveBtn = document.getElementById('wcApprove');
  const wcRejectBtn = document.getElementById('wcReject');

  if (wcApproveBtn && wcRejectBtn && wcPrompt) {
    wcApproveBtn.addEventListener('click', async () => {
      wcPrompt.dataset.action = 'approve';
      wcPrompt.style.display = 'none';
    });
    wcRejectBtn.addEventListener('click', () => {
      wcPrompt.dataset.action = 'reject';
      wcPrompt.style.display = 'none';
    });
  }
}

async function setApiHost(host) {
  chrome.storage.local.set({ [API_KEY]: host });
  // Update Cosmos SDK config
  COSMOS_SDK.config.restEndpoint = host;
  COSMOS_SDK.config.rpcEndpoint = host.replace('1317', '26657');
}

/**
 * Additional Helper Functions
 */

async function exportPrivateKey() {
  const privateKeyHex = await getPrivateKey();
  if (!privateKeyHex) {
    showMessage('walletMessage', 'No private key found', true);
    return;
  }

  const confirmed = confirm(
    'WARNING: Never share your private key with anyone! ' +
    'Anyone with access to your private key can steal your funds. ' +
    'Are you sure you want to view it?'
  );

  if (confirmed) {
    alert(`Your private key:\n\n${privateKeyHex}\n\nStore this securely and never share it!`);
  }
}

function buildAminoMsgs(messages) {
  return messages.map(msg => {
    const { ['@type']: type, ...rest } = msg;
    return {
      type: type || 'cosmos-sdk/MsgSend',
      value: rest,
    };
  });
}

async function signWithHardware(tx, accountInfo, fromAddress) {
  const hw = await getHardwareWallet();
  if (!hw?.address || hw.address !== fromAddress) {
    throw new Error('Hardware wallet not connected for this address');
  }

  const signDoc = {
    chain_id: COSMOS_SDK.config.chainId,
    account_number: accountInfo.accountNumber.toString(),
    sequence: accountInfo.sequence.toString(),
    fee: tx.auth_info.fee,
    msgs: buildAminoMsgs(tx.body.messages),
    memo: tx.body.memo || '',
  };

  const res = await signAmino({
    signDoc,
    prefix: COSMOS_SDK.config.bech32Prefix || 'paw',
    enforceChainId: COSMOS_SDK.config.chainId,
  });

  const pubKeyBase64 = COSMOS_SDK.bytesToBase64(res.publicKey);
  const signatureBase64 = COSMOS_SDK.bytesToBase64(res.signature);

  tx.auth_info.signer_infos = [{
    public_key: {
      '@type': '/cosmos.crypto.secp256k1.PubKey',
      key: pubKeyBase64,
    },
    mode_info: { single: { mode: 'SIGN_MODE_LEGACY_AMINO_JSON' } },
    sequence: accountInfo.sequence.toString(),
  }];
  tx.signatures = [signatureBase64];

  return tx;
}

async function deleteWallet() {
  const confirmed = confirm(
    'WARNING: This will delete your private key from this browser. ' +
    'Make sure you have backed up your private key first! ' +
    'This action cannot be undone. Continue?'
  );

  if (confirmed) {
    await chrome.storage.local.remove([PRIVATE_KEY_STORAGE, 'walletAddress']);
    await clearHardwareWallet();
    $('#walletAddress').value = '';
    $('#balanceDisplay').textContent = 'Balance: Wallet deleted';
    showMessage('walletMessage', 'Wallet deleted successfully');
  }
}

async function resolveSigner(fromAddress) {
  const hw = await getHardwareWallet();
  if (hw?.address && hw.address === fromAddress) {
    return { mode: 'hardware', hw };
  }

  const privateKeyHex = await getPrivateKey();
  if (privateKeyHex) {
    return { mode: 'software', privateKeyHex };
  }

  return { mode: 'none' };
}

async function connectLedgerHardware() {
  try {
    const accountIndexInput = document.getElementById('ledgerAccountIndex');
    const accountIndex = parseInt(accountIndexInput?.value || '0', 10);
    if (Number.isNaN(accountIndex) || accountIndex < 0 || accountIndex > 4) {
      updateHardwareStatus('Account index must be between 0 and 4', true);
      return;
    }
    updateHardwareStatus('Requesting device...', false);

    const path = `m/44'/118'/${accountIndex}'/0/0`;
    const res = await getLedgerAddress({
      path,
      prefix: 'paw',
      transportPreference: 'webhid',
    });

    const info = {
      type: 'ledger',
      address: res.address,
      path,
      transport: res.transportType,
      connectedAt: Date.now(),
    };
    await saveHardwareWallet(info);

    const walletAddressInput = $('#walletAddress');
    if (walletAddressInput) {
      walletAddressInput.value = res.address;
      chrome.storage.local.set({ walletAddress: res.address });
    }

    updateHardwareStatus(`Connected Ledger (${res.transportType}) @ ${res.address.slice(0, 10)}...`, false);
    showMessage('walletMessage', 'Ledger connected. Hardware signing will be used when available.', false);
  } catch (error) {
    updateHardwareStatus(`Ledger connect failed: ${error.message}`, true);
  }
}

async function queryAccountInfo() {
  const address = $('#walletAddress').value.trim();
  if (!address) {
    showMessage('walletMessage', 'Enter a wallet address first', true);
    return;
  }

  try {
    const accountInfo = await COSMOS_SDK.getAccount(address);
    const message = `Account Number: ${accountInfo.accountNumber}\n` +
                   `Sequence: ${accountInfo.sequence}\n` +
                   `Address: ${accountInfo.address}`;
    alert(message);
  } catch (error) {
    showMessage('walletMessage', `Error fetching account: ${error.message}`, true);
  }
}

function validateCosmosAddress(address) {
  // Basic validation for Cosmos Bech32 addresses
  if (!address || typeof address !== 'string') {
    return false;
  }
  return address.startsWith(COSMOS_SDK.config.bech32Prefix) && address.length >= 39;
}

async function checkNetworkConnection() {
  try {
    const response = await fetch(`${COSMOS_SDK.config.rpcEndpoint}/status`);
    if (response.ok) {
      const data = await response.json();
      return {
        connected: true,
        chainId: data.result?.node_info?.network,
        latestHeight: data.result?.sync_info?.latest_block_height,
      };
    }
    return { connected: false };
  } catch (error) {
    return { connected: false, error: error.message };
  }
}

function $(selector) {
  return document.querySelector(selector);
}

async function updateMiningStatus() {
  const address = $('#walletAddress').value.trim();
  if (!address) {
    $('#miningStatus').textContent = 'Status: wallet address required';
    return;
  }

  try {
    // Query validator status from Cosmos SDK
    const validatorUrl = `${COSMOS_SDK.config.rpcEndpoint}/validators`;
    const statusRes = await fetch(validatorUrl);
    if (!statusRes.ok) {
      $('#miningStatus').textContent = 'Status: unable to reach validator API';
      return;
    }

    const data = await statusRes.json();
    $('#miningStatus').textContent = 'Status: Network connected';
    $('#miningMeta').textContent = `Validators: ${data.result?.validators?.length || 0}`;
  } catch (error) {
    $('#miningStatus').textContent = `Status: ${error.message}`;
    $('#miningMeta').textContent = 'Network unavailable';
  }
}

async function startMining() {
  const address = $('#walletAddress').value.trim();
  if (!address) {
    showMessage('miningMessage', 'Enter a wallet address first', true);
    return;
  }

  showMessage('miningMessage', 'Note: PAW uses Proof-of-Stake. Use staking instead of mining.');
  await updateMiningStatus();
}

async function stopMining() {
  showMessage('miningMessage', 'Note: PAW uses Proof-of-Stake. Check staking section.');
  await updateMiningStatus();
}

async function refreshPools() {
  try {
    const pools = await COSMOS_SDK.queryPools();
    const list = $('#ordersList');

    if (pools && pools.length > 0) {
      list.innerHTML = pools
        .slice(0, 10)
        .map(
          pool => html`
        <div class="entry">
          <strong>Pool ${pool.id ?? 'N/A'}</strong>
          <br />
          ${pool.token0 || 'N/A'} / ${pool.token1 || 'N/A'} | Liquidity: ${pool.liquidity || 'N/A'}
        </div>`
        )
        .join('');
    } else {
      list.innerHTML = '<div class="list-placeholder">No pools available</div>';
    }
  } catch (error) {
    $('#ordersList').innerHTML = html`<div class="list-placeholder">Error loading pools: ${error?.message || 'unknown error'}</div>`;
  }
}

async function refreshOrders() {
  await refreshPools();
}

async function refreshMatches() {
  try {
    const prices = await COSMOS_SDK.queryOraclePrices();
    const list = $('#matchesList');

    if (prices && prices.length > 0) {
      list.innerHTML = prices
        .slice(0, 10)
        .map(
          price => html`
        <div class="entry">
          ${price.symbol}: $${price.price}
          <br />
          Updated: ${new Date(price.timestamp * 1000).toLocaleTimeString()}
        </div>`
        )
        .join('');
    } else {
      list.innerHTML = '<div class="list-placeholder">No price feeds available</div>';
    }
  } catch (error) {
    $('#matchesList').innerHTML = html`<div class="list-placeholder">Error loading prices: ${error?.message || 'unknown error'}</div>`;
  }
}

async function refreshTradeHistory() {
  const address = $('#walletAddress').value.trim();
  if (!address) {
    $('#tradeHistory').innerHTML = '<div class="list-placeholder">Enter wallet address</div>';
    return;
  }

  try {
    const encodedEvent = encodeURIComponent(`message.sender='${address}'`);
    const url = `${COSMOS_SDK.config.restEndpoint}/cosmos/tx/v1beta1/txs?events=${encodedEvent}&order_by=ORDER_BY_DESC&limit=5`;
    const res = await fetch(url);

    if (!res.ok) {
      $('#tradeHistory').innerHTML = '<div class="list-placeholder">Unable to fetch history</div>';
      return;
    }

    const data = await res.json();
    const container = $('#tradeHistory');

    if (data.txs && data.txs.length > 0) {
      container.innerHTML = data.txs
        .map(
          tx => html`
        <div class="entry">
          TX @ height ${tx.height || 'pending'}
          <br />
          Hash: ${(tx.txhash && `${tx.txhash.substring(0, 12)}...`) || 'n/a'} | Fee: ${tx.auth_info?.fee?.amount?.[0]?.amount || 0}
        </div>`
        )
        .join('');
    } else {
      container.innerHTML = '<div class="list-placeholder">No transactions found</div>';
    }
  } catch (error) {
    $('#tradeHistory').innerHTML = html`<div class="list-placeholder">Error: ${error?.message || 'unknown error'}</div>`;
  }
}

async function refreshMinerStats() {
  const address = $('#walletAddress').value.trim();
  if (!address) {
    $('#minerStats').textContent = 'Enter address for staking stats';
    return;
  }

  try {
    const url = `${COSMOS_SDK.config.restEndpoint}/cosmos/staking/v1beta1/delegations/${address}`;
    const res = await fetch(url);

    if (!res.ok) {
      $('#minerStats').textContent = 'No staking data';
      $('#minerHistory').textContent = 'Not staking';
      return;
    }

    const data = await res.json();
    const delegations = data.delegation_responses || [];

    if (delegations.length > 0) {
      const totalStaked = delegations.reduce((sum, del) => {
        return sum + parseInt(del.balance?.amount || 0);
      }, 0) / Math.pow(10, COSMOS_SDK.config.coinDecimals);

      $('#minerStats').textContent = `Staked: ${totalStaked} PAW | Delegations: ${delegations.length}`;
      $('#minerHistory').textContent = 'Active validator delegations';
    } else {
      $('#minerStats').textContent = 'No active delegations';
      $('#minerHistory').textContent = 'Not staking';
    }
  } catch (error) {
    $('#minerStats').textContent = `Error: ${error.message}`;
    $('#minerHistory').textContent = 'Unable to fetch staking data';
  }
}

async function delegateTokens(validatorAddress, amount) {
  const delegatorAddress = $('#walletAddress').value.trim();
  if (!delegatorAddress || !validateCosmosAddress(delegatorAddress)) {
    showMessage('transactionMessage', 'Set a valid delegator address first', true);
    return null;
  }
  if (!validateCosmosAddress(validatorAddress)) {
    showMessage('transactionMessage', 'Invalid validator address', true);
    return null;
  }
  if (!amount || amount <= 0) {
    showMessage('transactionMessage', 'Invalid delegation amount', true);
    return null;
  }

  const amountInMicroDenom = Math.floor(amount * Math.pow(10, COSMOS_SDK.config.coinDecimals));
  if (amountInMicroDenom <= 0) {
    showMessage('transactionMessage', 'Amount too small to delegate', true);
    return null;
  }

  const tx = {
    body: {
      messages: [{
        '@type': '/cosmos.staking.v1beta1.MsgDelegate',
        delegator_address: delegatorAddress,
        validator_address: validatorAddress,
        amount: { denom: 'upaw', amount: amountInMicroDenom.toString() },
      }],
      memo: 'Staking from PAW Browser Wallet',
      timeout_height: '0',
      extension_options: [],
      non_critical_extension_options: [],
    },
    auth_info: { signer_infos: [], fee: { amount: [{ denom: 'upaw', amount: '5000' }], gas: '200000' } },
    signatures: [],
  };

  try {
    const result = await signAndBroadcastTx(tx, delegatorAddress, 'transactionMessage');
    showMessage('transactionMessage', `Delegation broadcasted: ${result.txhash}`);
    await refreshMinerStats();
    return result;
  } catch (error) {
    showMessage('transactionMessage', `Delegation failed: ${error.message}`, true);
    return null;
  }
}

async function ibcTransfer(channel, port, receiver, amount, denom = 'upaw', timeoutSeconds = 600) {
  const sender = $('#walletAddress').value.trim();
  if (!sender || !validateCosmosAddress(sender)) {
    showMessage('transactionMessage', 'Set a valid sender address first', true);
    return null;
  }
  if (!receiver || !validateCosmosAddress(receiver)) {
    showMessage('transactionMessage', 'Invalid receiver address', true);
    return null;
  }
  if (!channel || !port) {
    showMessage('transactionMessage', 'Channel and port are required', true);
    return null;
  }
  if (denom !== 'upaw') {
    showMessage('transactionMessage', 'IBC transfers must use upaw', true);
    return null;
  }
  const amountInMicroDenom = Math.floor(amount * Math.pow(10, COSMOS_SDK.config.coinDecimals));
  if (amountInMicroDenom <= 0) {
    showMessage('transactionMessage', 'Amount too small to transfer', true);
    return null;
  }

  const currentHeight = await fetch(`${COSMOS_SDK.config.restEndpoint}/cosmos/base/tendermint/v1beta1/blocks/latest`)
    .then(res => res.json())
    .then(data => parseInt(data.block?.header?.height || '0', 10))
    .catch(() => 0);

  const tx = {
    body: {
      messages: [{
        '@type': '/ibc.applications.transfer.v1.MsgTransfer',
        source_port: port,
        source_channel: channel,
        token: { denom: 'upaw', amount: amountInMicroDenom.toString() },
        sender,
        receiver,
        timeout_height: {
          revision_number: '0',
          revision_height: (currentHeight + 1000).toString(),
        },
        timeout_timestamp: ((Date.now() + timeoutSeconds * 1000) * 1_000_000).toString(),
      }],
      memo: '',
      timeout_height: '0',
      extension_options: [],
      non_critical_extension_options: [],
    },
    auth_info: { signer_infos: [], fee: { amount: [{ denom: 'upaw', amount: '7500' }], gas: '250000' } },
    signatures: [],
  };

  try {
    const result = await signAndBroadcastTx(tx, sender, 'transactionMessage');
    showMessage('transactionMessage', `IBC transfer broadcasted: ${result.txhash}`);
    await refreshTradeHistory();
    return result;
  } catch (error) {
    showMessage('transactionMessage', `IBC transfer failed: ${error.message}`, true);
    return null;
  }
}

async function govVote(proposalId, option = 'VOTE_OPTION_YES') {
  const voter = $('#walletAddress').value.trim();
  if (!voter || !validateCosmosAddress(voter)) {
    showMessage('transactionMessage', 'Set a valid voter address first', true);
    return null;
  }
  if (!proposalId || Number.isNaN(Number(proposalId))) {
    showMessage('transactionMessage', 'Invalid proposal id', true);
    return null;
  }
  const voteOption = option || 'VOTE_OPTION_YES';
  const tx = {
    body: {
      messages: [{
        '@type': '/cosmos.gov.v1.MsgVote',
        proposal_id: proposalId.toString(),
        voter,
        option: voteOption,
      }],
      memo: '',
      timeout_height: '0',
      extension_options: [],
      non_critical_extension_options: [],
    },
    auth_info: { signer_infos: [], fee: { amount: [{ denom: 'upaw', amount: '4000' }], gas: '180000' } },
    signatures: [],
  };

  try {
    const result = await signAndBroadcastTx(tx, voter, 'transactionMessage');
    showMessage('transactionMessage', `Vote broadcasted: ${result.txhash}`);
    return result;
  } catch (error) {
    showMessage('transactionMessage', `Vote failed: ${error.message}`, true);
    return null;
  }
}

async function submitOrder(event) {
  event.preventDefault();
  const form = event.target;
  const formData = new FormData(form);

  const tokenOffered = formData.get('tokenOffered');
  const amountOffered = parseFloat(formData.get('amountOffered'));
  const tokenRequested = formData.get('tokenRequested');
  const amountRequested = parseFloat(formData.get('amountRequested'));

  if (!tokenOffered || !tokenRequested || !amountOffered || !amountRequested) {
    showMessage('tradeMessage', 'Please fill in all fields', true);
    return;
  }

  try {
    // For now, use the swap function
    // In a real implementation, this would map to DEX order creation
    const poolId = 1; // Default pool - should be dynamically selected
    const minAmountOut = Math.floor(amountRequested * 0.95); // 5% slippage tolerance

    const result = await executeSwap(
      poolId,
      tokenOffered,
      Math.floor(amountOffered * Math.pow(10, COSMOS_SDK.config.coinDecimals)),
      tokenRequested,
      minAmountOut
    );

    if (result) {
      showMessage('tradeMessage', `Swap successful! Hash: ${result.txhash}`);
      await refreshPools();
      await refreshTradeHistory();
    }
  } catch (error) {
    showMessage('tradeMessage', `Swap failed: ${error.message}`, true);
  }
}

function setAiStatus(message, isError = false) {
  const aiStatus = $('#aiStatus');
  aiStatus.textContent = message;
  aiStatus.classList.toggle('error', isError);
}

function setKeyDeletionNotice(message) {
  $('#aiKeyDeleted').textContent = message;
}

function clearAiKeyField() {
  const keyInput = $('#aiApiKey');
  keyInput.value = '';
  setKeyDeletionNotice('Your AI API key has been deleted from this wallet.');
}

async function runPersonalAiSwap() {
  const userAddress = $('#walletAddress').value.trim();
  const mode = $('#aiKeyMode').value;
  let apiKey = $('#aiApiKey').value.trim();
  const provider = $('#aiProvider').value.trim() || 'anthropic';
  const model = $('#aiModel').value.trim() || 'claude-sonnet-4-5';
  const swapDetails = {
    from_coin: $('#aiFromCoin').value.trim() || 'PAW',
    to_coin: $('#aiToCoin').value.trim() || 'USDC',
    amount: parseFloat($('#aiAmount').value) || 0,
    recipient_address: $('#aiRecipient').value.trim() || userAddress,
  };

  if (!userAddress) {
    setAiStatus('Provide your wallet address before using the assistant', true);
    return;
  }
  if (mode === 'session') {
    apiKey = apiKey || (await getStoredAiKey());
  }
  if (!apiKey) {
    setAiStatus('Enter your AI API key for this session', true);
    return;
  }
  if (mode === 'session') {
    storeAiKey(apiKey);
  }
  if (!swapDetails.amount) {
    setAiStatus('Enter a swap amount before running the assistant', true);
    return;
  }

  setAiStatus('Preparing AI-assisted swap...');

  try {
    // Execute the swap using Cosmos SDK
    const poolId = 1; // Default pool
    const amountInMicroDenom = Math.floor(
      swapDetails.amount * Math.pow(10, COSMOS_SDK.config.coinDecimals)
    );
    const minAmountOut = Math.floor(amountInMicroDenom * 0.95); // 5% slippage

    const result = await executeSwap(
      poolId,
      swapDetails.from_coin.toLowerCase(),
      amountInMicroDenom,
      swapDetails.to_coin.toLowerCase(),
      minAmountOut
    );

    if (result) {
      setAiStatus(`Swap successful! Hash: ${result.txhash}`);
      setKeyDeletionNotice('Transaction complete. API key removed from this extension.');
      await updateBalance();
      await refreshTradeHistory();
    } else {
      throw new Error('Swap transaction failed');
    }
  } catch (error) {
    setAiStatus(`Swap error: ${error.message}`, true);
  } finally {
    if (mode === 'temporary' || mode === 'external') {
      clearAiKeyField();
    } else if (mode === 'session') {
      setKeyDeletionNotice('Transaction complete. Stored key remains until you click Clear Key.');
    }
  }
}

async function getStoredAiKey() {
  return new Promise(resolve => {
    chrome.storage.local.get(['personalAiApiKey'], result => {
      resolve(result.personalAiApiKey || '');
    });
  });
}

function storeAiKey(value) {
  chrome.storage.local.set({ personalAiApiKey: value });
}

function clearStoredAiKey() {
  chrome.storage.local.remove('personalAiApiKey');
  clearAiKeyField();
  setKeyDeletionNotice('Stored Personal AI key removed.');
}

function bindActions() {
  injectHardwareControls();

  // Wallet management
  const generateWalletBtn = $('#generateWallet');
  const importWalletBtn = $('#importWallet');
  const refreshBalanceBtn = $('#refreshBalance');
  const exportKeyBtn = $('#exportPrivateKey');
  const deleteWalletBtn = $('#deleteWallet');
  const accountInfoBtn = $('#accountInfo');

  if (generateWalletBtn) {
    generateWalletBtn.addEventListener('click', generateNewWallet);
  }
  if (importWalletBtn) {
    importWalletBtn.addEventListener('click', () => {
      const privateKey = prompt('Enter your private key (hex):');
      if (privateKey) {
        importWallet(privateKey);
      }
    });
  }
  if (refreshBalanceBtn) {
    refreshBalanceBtn.addEventListener('click', updateBalance);
  }
  if (exportKeyBtn) {
    exportKeyBtn.addEventListener('click', exportPrivateKey);
  }
  if (deleteWalletBtn) {
    deleteWalletBtn.addEventListener('click', deleteWallet);
  }
  if (accountInfoBtn) {
    accountInfoBtn.addEventListener('click', queryAccountInfo);
  }

  // Mining/Staking
  const startMiningBtn = $('#startMining');
  const stopMiningBtn = $('#stopMining');
  if (startMiningBtn) startMiningBtn.addEventListener('click', startMining);
  if (stopMiningBtn) stopMiningBtn.addEventListener('click', stopMining);

  // Trading
  const refreshOrdersBtn = $('#refreshOrders');
  const refreshMatchesBtn = $('#refreshMatches');
  const orderForm = $('#orderForm');

  if (refreshOrdersBtn) refreshOrdersBtn.addEventListener('click', refreshOrders);
  if (refreshMatchesBtn) refreshMatchesBtn.addEventListener('click', refreshMatches);
  if (orderForm) orderForm.addEventListener('submit', submitOrder);

  // AI Assistant
  const runAiSwapBtn = $('#runAiSwap');
  const clearAiKeyBtn = $('#clearAiKey');

  if (runAiSwapBtn) runAiSwapBtn.addEventListener('click', runPersonalAiSwap);
  if (clearAiKeyBtn) clearAiKeyBtn.addEventListener('click', clearStoredAiKey);

  // API Host
  const apiHostInput = $('#apiHost');
  if (apiHostInput) {
    apiHostInput.addEventListener('change', event => {
      const newHost = event.target.value.trim();
      setApiHost(newHost);
      // Update Cosmos SDK config
      COSMOS_SDK.config.restEndpoint = newHost;
      COSMOS_SDK.config.rpcEndpoint = newHost.replace('1317', '26657');
    });
  }

  const allowlistInput = document.getElementById('wcAllowlist');
  if (allowlistInput) {
    allowlistInput.addEventListener('change', async event => {
      const raw = event.target.value || '';
      const list = raw.split(',').map(item => item.trim()).filter(Boolean);
      const finalList = list.length ? list : WC_DEFAULT_ALLOWLIST;
      await saveWalletConnectAllowlist(finalList);
      renderWcAuditLog();
    });
  }

  const wcShowAuditBtn = document.getElementById('wcShowAudit');
  const wcClearAuditBtn = document.getElementById('wcClearAudit');
  if (wcShowAuditBtn) {
    wcShowAuditBtn.addEventListener('click', async () => {
      const log = await getWcAuditLog();
      alert(`WalletConnect audit log (last ${log.length}):\n${JSON.stringify(log, null, 2)}`);
      renderWcAuditLog();
    });
  }
  if (wcClearAuditBtn) {
    wcClearAuditBtn.addEventListener('click', async () => {
      await clearWcAuditLog();
      alert('WalletConnect audit log cleared');
      renderWcAuditLog();
    });
  }
}

function restoreSettings() {
  chrome.storage.local.get(['walletAddress', API_KEY, HARDWARE_WALLET_KEY, WC_ALLOWLIST_KEY], result => {
    const walletAddressInput = $('#walletAddress');
    const apiHostInput = $('#apiHost');

    if (result.walletAddress && walletAddressInput) {
      walletAddressInput.value = result.walletAddress;
    }
    if (result[API_KEY] && apiHostInput) {
      apiHostInput.value = result[API_KEY];
      // Update Cosmos SDK config
      COSMOS_SDK.config.restEndpoint = result[API_KEY];
      COSMOS_SDK.config.rpcEndpoint = result[API_KEY].replace('1317', '26657');
    }

    if (result[HARDWARE_WALLET_KEY]) {
      const hw = result[HARDWARE_WALLET_KEY];
      updateHardwareStatus(
        `Last connected Ledger (${hw.transport || 'unknown'}) @ ${hw.address?.slice(0, 10) || 'n/a'}...`,
        false
      );
      if (walletAddressInput && !walletAddressInput.value && hw.address) {
        walletAddressInput.value = hw.address;
      }
    }

    const allowlistInput = document.getElementById('wcAllowlist');
    if (allowlistInput) {
      const list = result[WC_ALLOWLIST_KEY] || WC_DEFAULT_ALLOWLIST;
      allowlistInput.value = list.join(',');
    }

    renderWcAuditLog();
  });

  const walletAddressInput = $('#walletAddress');
  if (walletAddressInput) {
    walletAddressInput.addEventListener('change', async event => {
      chrome.storage.local.set({ walletAddress: event.target.value.trim() });
      await updateBalance();
    });
  }
}

async function initializeWallet() {
  try {
    // Check network connection first
    const networkStatus = await checkNetworkConnection();
    const statusElement = $('#networkStatus');

    if (networkStatus.connected) {
      if (statusElement) {
        statusElement.textContent = `Connected to ${networkStatus.chainId || 'PAW'} | Block: ${networkStatus.latestHeight || 'N/A'}`;
        statusElement.classList.remove('error');
      }
    } else {
      if (statusElement) {
        statusElement.textContent = `Disconnected: ${networkStatus.error || 'Network unavailable'}`;
        statusElement.classList.add('error');
      }
    }

    // Check if we have a stored private key
    const privateKeyHex = await getPrivateKey();
    if (privateKeyHex) {
      const privateKey = COSMOS_SDK.hexToBytes(privateKeyHex);
      const publicKey = await COSMOS_SDK.getPublicKey(privateKey);
      const address = COSMOS_SDK.publicKeyToAddress(publicKey);

      const walletAddressInput = $('#walletAddress');
      if (walletAddressInput && !walletAddressInput.value) {
        walletAddressInput.value = address;
        chrome.storage.local.set({ walletAddress: address });
      }

      // Validate the address format
      if (!validateCosmosAddress(address)) {
        console.warn('Generated address may be invalid:', address);
      }
    } else {
      const hw = await getHardwareWallet();
      if (hw?.address) {
        const walletAddressInput = $('#walletAddress');
        if (walletAddressInput && !walletAddressInput.value) {
          walletAddressInput.value = hw.address;
          chrome.storage.local.set({ walletAddress: hw.address });
        }
        updateHardwareStatus(
          `Ledger ready (${hw.transport || 'last used'}) @ ${hw.address.slice(0, 10)}...`,
          false
        );
      }
    }
  } catch (error) {
    console.error('Error initializing wallet:', error);
    showMessage('walletMessage', `Initialization error: ${error.message}`, true);
  }
}

async function updateNetworkStatus() {
  const networkStatus = await checkNetworkConnection();
  const statusElement = $('#networkStatus');

  if (statusElement) {
    if (networkStatus.connected) {
      statusElement.textContent = `Connected to ${networkStatus.chainId || 'PAW'} | Block: ${networkStatus.latestHeight || 'N/A'}`;
      statusElement.classList.remove('error');
    } else {
      statusElement.textContent = `Disconnected: ${networkStatus.error || 'Network unavailable'}`;
      statusElement.classList.add('error');
    }
  }
}

async function loadWalletConnectAllowlist() {
  return new Promise(resolve => {
    chrome.storage.local.get([WC_ALLOWLIST_KEY], result => {
      resolve(result[WC_ALLOWLIST_KEY] || WC_DEFAULT_ALLOWLIST);
    });
  });
}

async function saveWalletConnectAllowlist(list) {
  return new Promise(resolve => {
    chrome.storage.local.set({ [WC_ALLOWLIST_KEY]: list }, () => resolve());
  });
}

async function recordWcAudit(entry) {
  return new Promise(resolve => {
    chrome.storage.local.get([WC_AUDIT_LOG_KEY], result => {
      const current = Array.isArray(result[WC_AUDIT_LOG_KEY]) ? result[WC_AUDIT_LOG_KEY] : [];
      const next = [...current, entry].slice(-10);
      chrome.storage.local.set({ [WC_AUDIT_LOG_KEY]: next }, () => resolve());
    });
  });
}

async function getWcAuditLog() {
  return new Promise(resolve => {
    chrome.storage.local.get([WC_AUDIT_LOG_KEY], result => {
      resolve(Array.isArray(result[WC_AUDIT_LOG_KEY]) ? result[WC_AUDIT_LOG_KEY] : []);
    });
  });
}

async function clearWcAuditLog() {
  return new Promise(resolve => {
    chrome.storage.local.remove([WC_AUDIT_LOG_KEY], () => resolve());
  });
}

async function renderWcAuditLog() {
  const container = document.getElementById('wcAuditLog');
  if (!container) return;
  const log = await getWcAuditLog();
  if (!log.length) {
    container.textContent = 'No WalletConnect activity yet.';
    return;
  }
  container.textContent = log
    .map(item => `${new Date(item.timestamp).toISOString()} | ${item.type || 'sign'} | ${item.origin} | ${item.mode} | ${item.chainId || 'n/a'} | ${item.address || 'n/a'}`)
    .join('\n');
}

if (typeof document !== 'undefined') {
  document.addEventListener('DOMContentLoaded', async () => {
    try {
      bindActions();
      restoreSettings();

      const apiHostInput = $('#apiHost');
      if (apiHostInput) {
        const host = await getApiHost();
        apiHostInput.value = host;
        COSMOS_SDK.config.restEndpoint = host;
        COSMOS_SDK.config.rpcEndpoint = host.replace('1317', '26657');
      }

      await initializeWallet();

      // Safe async calls with error handling
      await safeAsyncCall(updateBalance, 'balance update');
      await safeAsyncCall(refreshOrders, 'pools refresh');
      await safeAsyncCall(refreshMatches, 'prices refresh');
      await safeAsyncCall(updateMiningStatus, 'network status');
      await safeAsyncCall(refreshMinerStats, 'staking stats');
      await safeAsyncCall(refreshTradeHistory, 'transaction history');

      // Listen for WalletConnect sign requests forwarded from background
      chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
        if (message.type === 'walletconnect-session') {
          handleSessionProposal(message.request)
            .then(async res => {
              await recordWcAudit({
                id: message.request?.id || 'session',
                type: 'session',
                origin: message.request?.origin || 'unknown',
                mode: 'n/a',
                chainId: message.request?.chains?.join(',') || 'n/a',
                address: message.request?.address || 'n/a',
                timestamp: Date.now(),
                result: res.approved ? 'approved' : 'rejected',
              });
              chrome.runtime.sendMessage({
                type: 'walletconnect-session-result',
                id: message.request?.id,
                result: res,
              });
              sendResponse(res);
              renderWcAuditLog();
            })
            .catch(err => {
              recordWcAudit({
                id: message.request?.id || 'session',
                type: 'session',
                origin: message.request?.origin || 'unknown',
                mode: 'n/a',
                chainId: message.request?.chains?.join(',') || 'n/a',
                address: message.request?.address || 'n/a',
                timestamp: Date.now(),
                result: 'error',
                error: err.message || 'Session failed',
              });
              chrome.runtime.sendMessage({
                type: 'walletconnect-session-result',
                id: message.request?.id,
                result: { approved: false, error: err.message || 'Session failed' },
              });
              sendResponse({ approved: false, error: err.message || 'Session failed' });
            });
          return true;
        }
        if (message.type === 'walletconnect-sign') {
          const summary = summarizeSignRequest(message.request);
          const modal = ensureWalletConnectModal(summary);
          setWcModalStatus(modal, 'Awaiting approval', false);

          const watcher = setInterval(() => {
            if (modal.dataset.action === 'approve') {
              clearInterval(watcher);
              setWcModalStatus(modal, 'Signing...', false);
              handleWalletConnectSignRequest(message.request)
                .then(res => {
                  setWcModalStatus(modal, 'Signed successfully', false);
                  sendResponse(res);
                })
                .catch(err => {
                  setWcModalStatus(modal, err.message || 'WalletConnect signing failed', true);
                  sendResponse({ error: err.message || 'WalletConnect signing failed' });
                });
            } else if (modal.dataset.action === 'reject') {
              clearInterval(watcher);
              recordWcAudit({
                id: message.request?.id,
                type: 'sign',
                origin: message.request?.params?.[0]?.origin || 'unknown',
                mode: 'user-reject',
                chainId: message.request?.params?.[0]?.signDoc?.chain_id || 'n/a',
                address: message.request?.params?.[0]?.signerAddress || message.request?.params?.[0]?.signer_address || 'n/a',
                timestamp: Date.now(),
                result: 'rejected',
              });
              chrome.runtime.sendMessage({
                type: 'walletconnect-sign-result',
                id: message.request?.id,
                result: { error: 'User rejected WalletConnect request', status: 'rejected' },
              });
              sendResponse({ error: 'User rejected WalletConnect request', status: 'rejected' });
            }
          }, 100);
          return true;
        }
        return undefined;
      });

      // Set up auto-refresh every 30 seconds
      setInterval(async () => {
        await safeAsyncCall(updateNetworkStatus, 'network status');
        await safeAsyncCall(updateBalance, 'balance update');
        await safeAsyncCall(refreshPools, 'pools refresh');
        await safeAsyncCall(refreshMatches, 'prices refresh');
        await safeAsyncCall(updateMiningStatus, 'network status');
      }, 30000);

      console.log('PAW Browser Wallet initialized successfully');
    } catch (error) {
      console.error('Fatal initialization error:', error);
      showMessage('walletMessage', `Failed to initialize wallet: ${error.message}`, true);
    }
  });
}

/**
 * Safe async call wrapper with error handling
 */
async function safeAsyncCall(fn, operationName) {
  try {
    await fn();
  } catch (error) {
    console.error(`Error during ${operationName}:`, error);
    // Don't show UI errors for background operations to avoid spam
  }
}
