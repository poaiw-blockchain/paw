import { LedgerSigner } from '@cosmjs/ledger-amino';
import { makeCosmoshubPath } from '@cosmjs/amino';
import TransportNodeHid from '@ledgerhq/hw-transport-node-hid';
import { SigningStargateClient, GasPrice } from '@cosmjs/stargate';
import { ApiService } from './api';
import { bech32 } from 'bech32';

const DEFAULT_PREFIX = 'paw';
const DEFAULT_GAS_PRICE = '0.025upaw';
const DEFAULT_ACCOUNT_INDICES = [0, 1, 2, 3, 4];
const DEFAULT_ALLOWED_FEE_DENOMS = ['upaw'];
const SIGN_LIMIT_PER_MINUTE = 5;

export class LedgerService {
  constructor() {
    this.storageKey = 'paw-ledger-wallet';
    this.api = new ApiService();
    this.allowedManufacturers = ['Ledger'];
    this.allowedProducts = ['Nano S', 'Nano X', 'Nano S Plus', 'Ledger Device'];
    this.signTimestamps = [];
  }

  enforceRateLimit() {
    const now = Date.now();
    this.signTimestamps = (this.signTimestamps || []).filter(ts => now - ts < 60000);
    if (this.signTimestamps.length >= SIGN_LIMIT_PER_MINUTE) {
      throw new Error('Signing rate limited: too many requests in a short window');
    }
    this.signTimestamps.push(now);
  }

  async getTransportPreference() {
    if (window.electron?.store) {
      const pref = await window.electron.store.get('ledgerTransport');
      return pref || 'hid';
    }
    return localStorage.getItem('ledgerTransport') || 'hid';
  }

  async setTransportPreference(preference) {
    if (window.electron?.store) {
      await window.electron.store.set('ledgerTransport', preference);
    } else {
      localStorage.setItem('ledgerTransport', preference);
    }
  }

  attestTransport(transport) {
    const manufacturer = transport?.device?.manufacturerName || '';
    const product = transport?.device?.productName || '';
    if (manufacturer && !this.allowedManufacturers.some(m => manufacturer.toLowerCase().includes(m.toLowerCase()))) {
      throw new Error(`Unexpected Ledger manufacturer: ${manufacturer}`);
    }
    if (product && !this.allowedProducts.some(p => product.toLowerCase().includes(p.toLowerCase()))) {
      throw new Error(`Unexpected Ledger model: ${product}`);
    }
  }

  async getTransport(transportType = null) {
    const preference = transportType || (await this.getTransportPreference());
    if (preference === 'webhid') {
      try {
        const WebHID = (await import('@ledgerhq/hw-transport-webhid')).default;
        const t = await WebHID.create();
        this.attestTransport(t);
        return t;
      } catch (err) {
        throw new Error('WebHID transport not available in this build; install @ledgerhq/hw-transport-webhid or use HID.');
      }
    }

    const t = await TransportNodeHid.create();
    this.attestTransport(t);
    return t;
  }

  validateSignDoc(signDoc, expectedChainId, prefix = DEFAULT_PREFIX) {
    if (!signDoc?.chain_id) {
      throw new Error('chain_id is required for Ledger signing');
    }
    if (expectedChainId && signDoc.chain_id !== expectedChainId) {
      throw new Error(`Chain-id mismatch (expected ${expectedChainId}, got ${signDoc.chain_id})`);
    }
    const fee = signDoc?.fee || {};
    if (!fee.gas || Number.isNaN(Number(fee.gas)) || Number(fee.gas) <= 0) {
      throw new Error('Invalid gas value in fee');
    }
    if (!Array.isArray(fee.amount) || !fee.amount.length) {
      throw new Error('Fee amount required');
    }
    fee.amount.forEach((coin) => {
      if (!DEFAULT_ALLOWED_FEE_DENOMS.includes(coin.denom)) {
        throw new Error(`Fee denom ${coin.denom} not permitted`);
      }
      if (Number.isNaN(Number(coin.amount)) || Number(coin.amount) < 0) {
        throw new Error('Fee amount must be non-negative');
      }
    });

    const msgs = signDoc.msgs || signDoc.messages || [];
    msgs.forEach(msg => {
      const value = msg.value || msg;
      if (value && typeof value === 'object') {
        Object.entries(value).forEach(([key, val]) => {
          if (typeof val === 'string' && key.toLowerCase().includes('address')) {
            this.validateBech32Prefix(val, prefix);
          }
        });
      }
    });
  }

  async getSigner(prefix = DEFAULT_PREFIX, accountIndices = DEFAULT_ACCOUNT_INDICES, transportType = null) {
    const indices = Array.isArray(accountIndices) ? accountIndices : [accountIndices];
    indices.forEach((idx) => {
      if (!DEFAULT_ACCOUNT_INDICES.includes(idx)) {
        throw new Error('Account index must be between 0 and 4');
      }
    });
    const transport = await this.getTransport(transportType);
    const paths = indices.map(idx => makeCosmoshubPath(idx));
    return new LedgerSigner(transport, {
      hdPaths: paths,
      prefix,
    });
  }

  async getAccounts(accountIndices = DEFAULT_ACCOUNT_INDICES, prefix = DEFAULT_PREFIX, transportType = 'hid') {
    const signer = await this.getSigner(prefix, accountIndices, transportType);
    const accounts = await signer.getAccounts();
    if (!accounts?.length) {
      throw new Error('Ledger did not return any accounts');
    }
    accounts.forEach(({ address }) => this.validateBech32Prefix(address, prefix));
    return accounts;
  }

  async getSigningClient({ accountIndex = 0, transportType = null, expectedChainId, signDoc } = {}) {
    this.enforceRateLimit();
    if (signDoc) {
      this.validateSignDoc(signDoc, expectedChainId, DEFAULT_PREFIX);
    }
    const signer = await this.getSigner(DEFAULT_PREFIX, [accountIndex], transportType);
    const restEndpoint = await this.api.getEndpoint();
    const rpcEndpoint = this.normalizeRpcEndpoint(restEndpoint);
    const client = await SigningStargateClient.connectWithSigner(rpcEndpoint, signer, {
      gasPrice: GasPrice.fromString(DEFAULT_GAS_PRICE),
    });

    if (expectedChainId) {
      const chainId = await client.getChainId();
      if (chainId !== expectedChainId) {
        throw new Error(`Chain-id mismatch (expected ${expectedChainId}, got ${chainId})`);
      }
    }

    return { client, signer };
  }

  async saveWalletMetadata(wallet) {
    if (!wallet?.address) return;
    if (window.electron?.store) {
      await window.electron.store.set(this.storageKey, wallet);
    } else {
      localStorage.setItem(this.storageKey, JSON.stringify(wallet));
    }
  }

  async getSavedWallet() {
    let data = null;
    if (window.electron?.store) {
      data = await window.electron.store.get(this.storageKey);
    } else {
      const stored = localStorage.getItem(this.storageKey);
      data = stored ? JSON.parse(stored) : null;
    }
    return data;
  }

  async clearSavedWallet() {
    if (window.electron?.store) {
      await window.electron.store.delete(this.storageKey);
    } else {
      localStorage.removeItem(this.storageKey);
    }
  }

  normalizeRpcEndpoint(restEndpoint) {
    if (!restEndpoint) {
      throw new Error('REST endpoint is required to derive RPC endpoint');
    }
    return restEndpoint.replace('1317', '26657').replace(/\/cosmos.*/, '');
  }

  validateBech32Prefix(address, prefix = DEFAULT_PREFIX) {
    const { prefix: decodedPrefix } = bech32.decode(address);
    if (decodedPrefix !== prefix) {
      throw new Error(`Address prefix mismatch: expected ${prefix}, got ${decodedPrefix}`);
    }
  }
}

export default LedgerService;
