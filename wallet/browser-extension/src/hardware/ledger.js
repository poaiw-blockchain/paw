import TransportWebUSB from '@ledgerhq/hw-transport-webusb';
import TransportWebHID from '@ledgerhq/hw-transport-webhid';
import CosmosApp from '@ledgerhq/hw-app-cosmos';
import { fromBech32 } from '@cosmjs/encoding';

const DEFAULT_PREFIX = 'paw';
const DEFAULT_TIMEOUT = 60000;
const DEFAULT_ALLOWED_FEE_DENOMS = ['upaw'];
const DEFAULT_MAX_ACCOUNT = 4;

function hexToBytes(hex) {
  const clean = hex.startsWith('0x') ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < clean.length; i += 2) {
    out[i / 2] = parseInt(clean.substr(i, 2), 16);
  }
  return out;
}

function base64ToBytes(b64) {
  if (typeof Buffer !== 'undefined') {
    return Uint8Array.from(Buffer.from(b64, 'base64'));
  }
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

export function normalizePath(path, maxAccount = DEFAULT_MAX_ACCOUNT) {
  const sanitized = path.startsWith('m/') ? path.slice(2) : path;
  const segments = sanitized.split('/');
  if (segments.length !== 5) {
    throw new Error(`Invalid derivation path: ${path}`);
  }
  const accountSeg = segments[2];
  const account = parseInt(accountSeg.replace("'", ''), 10);
  if (Number.isNaN(account) || account > maxAccount) {
    throw new Error(`Account index exceeds allowed maximum (${maxAccount})`);
  }
  return segments
    .map((seg, idx) => {
      const hardened = seg.endsWith("'");
      const val = parseInt(hardened ? seg.slice(0, -1) : seg, 10);
      if (Number.isNaN(val)) {
        throw new Error(`Invalid path segment: ${seg}`);
      }
      if (idx < 3 && !hardened) {
        throw new Error(`Path segment must be hardened: ${seg}`);
      }
      return hardened || idx < 3 ? `${val}'` : `${val}`;
    })
    .join('/');
}

export function assertBech32Prefix(address, expectedPrefix = DEFAULT_PREFIX) {
  const decoded = fromBech32(address);
  if (decoded.prefix !== expectedPrefix) {
    throw new Error(`Address prefix mismatch: expected ${expectedPrefix}, got ${decoded.prefix}`);
  }
}

export function validateSignDocBasics(signDoc, { enforceChainId, allowedFeeDenoms = DEFAULT_ALLOWED_FEE_DENOMS } = {}) {
  if (!signDoc?.chain_id) throw new Error('chain_id is required for hardware signing');
  if (enforceChainId && signDoc.chain_id !== enforceChainId) {
    throw new Error(`Refusing to sign: chain-id mismatch (${signDoc.chain_id} != ${enforceChainId})`);
  }
  const gas = signDoc?.fee?.gas;
  if (!gas || Number.isNaN(Number(gas)) || Number(gas) <= 0) {
    throw new Error('Invalid or missing gas value');
  }
  const amounts = signDoc?.fee?.amount || [];
  if (!Array.isArray(amounts) || amounts.length === 0) {
    throw new Error('Fee amount is required for hardware signing');
  }
  for (const coin of amounts) {
    if (!coin?.denom || !allowedFeeDenoms.includes(coin.denom)) {
      throw new Error(`Fee denom ${coin?.denom || 'unknown'} is not permitted for hardware signing`);
    }
    if (Number.isNaN(Number(coin.amount)) || Number(coin.amount) < 0) {
      throw new Error('Fee amount must be a non-negative number');
    }
  }
}

async function createTransport(preference = 'webhid', timeoutMs = DEFAULT_TIMEOUT) {
  const candidates = preference === 'webusb' ? ['webusb', 'webhid'] : ['webhid', 'webusb'];

  for (const candidate of candidates) {
    if (candidate === 'webhid' && (await TransportWebHID.isSupported())) {
      const t = await TransportWebHID.create();
      t.setExchangeTimeout(timeoutMs);
      return { transport: t, type: 'webhid' };
    }
    if (candidate === 'webusb' && (await TransportWebUSB.isSupported())) {
      const t = await TransportWebUSB.create();
      t.setExchangeTimeout(timeoutMs);
      return { transport: t, type: 'webusb' };
    }
  }

  throw new Error('No supported Ledger transport (WebHID/WebUSB) available in this environment');
}

function basicAttestationCheck(transport, allowedManufacturers = ['Ledger'], allowedProducts = []) {
  const manufacturer = transport?.device?.manufacturerName || '';
  const productName = transport?.device?.productName || '';

  if (allowedManufacturers.length && manufacturer) {
    const ok = allowedManufacturers.some((m) => manufacturer.toLowerCase().includes(m.toLowerCase()));
    if (!ok) throw new Error(`Unexpected manufacturer: ${manufacturer}`);
  }

  if (allowedProducts.length && productName) {
    const ok = allowedProducts.some((p) => productName.toLowerCase().includes(p.toLowerCase()));
    if (!ok) throw new Error(`Unexpected device model: ${productName}`);
  }
}

export async function getLedgerAddress({
  path = "m/44'/118'/0'/0/0",
  prefix = DEFAULT_PREFIX,
  transportPreference = 'webhid',
  timeoutMs = DEFAULT_TIMEOUT,
  allowedManufacturers,
  allowedProducts,
} = {}) {
  const normalizedPath = normalizePath(path);
  const { transport, type } = await createTransport(transportPreference, timeoutMs);
  try {
    basicAttestationCheck(transport, allowedManufacturers, allowedProducts);
    const app = new CosmosApp(transport);
    const res = await app.getAddress(normalizedPath, prefix, false);
    assertBech32Prefix(res.address, prefix);
    return { address: res.address, publicKey: hexToBytes(res.publicKey), transportType: type };
  } finally {
    await transport.close();
  }
}

export async function signAmino({
  signDoc,
  path = "m/44'/118'/0'/0/0",
  prefix = DEFAULT_PREFIX,
  transportPreference = 'webhid',
  timeoutMs = DEFAULT_TIMEOUT,
  enforceChainId,
  allowedFeeDenoms = DEFAULT_ALLOWED_FEE_DENOMS,
  allowedManufacturers,
  allowedProducts,
} = {}) {
  const normalizedPath = normalizePath(path);
  validateSignDocBasics(signDoc, { enforceChainId, allowedFeeDenoms });

  const { transport, type } = await createTransport(transportPreference, timeoutMs);
  try {
    basicAttestationCheck(transport, allowedManufacturers, allowedProducts);
    const app = new CosmosApp(transport);
    const response = await app.sign(normalizedPath, JSON.stringify(signDoc));
    const publicKey = await app.getAddress(normalizedPath, prefix, false);
    assertBech32Prefix(publicKey.address, prefix);

    return {
      signature: base64ToBytes(response.signature),
      publicKey: hexToBytes(publicKey.publicKey),
      transportType: type,
    };
  } finally {
    await transport.close();
  }
}
