import CosmosApp from '@ledgerhq/hw-app-cosmos';
import TransportBLE from './bleTransport';
import { assertBech32Prefix, normalizePath, validateFee } from './guards';

const DEFAULT_PREFIX = 'paw';
const DEFAULT_ALLOWED_FEE_DENOMS = ['upaw'];
const DEFAULT_MAX_ACCOUNT_INDEX = 4;

export class LedgerServiceMobile {
  transport: TransportBLE | null = null;
  app: CosmosApp | null = null;

  async connect(accountIndex = 0): Promise<{ address: string; publicKey: Uint8Array; path: string }> {
    if (accountIndex < 0 || accountIndex > DEFAULT_MAX_ACCOUNT_INDEX) {
      throw new Error('Account index must be between 0 and 4');
    }
    const path = normalizePath(`m/44'/118'/${accountIndex}'/0/0`, DEFAULT_MAX_ACCOUNT_INDEX);
    this.transport = await TransportBLE.create();
    this.app = new CosmosApp(this.transport as any);
    const res = await this.app.getAddress(path, DEFAULT_PREFIX, false);
    assertBech32Prefix(res.address, DEFAULT_PREFIX);
    return { address: res.address, publicKey: hexToBytes(res.publicKey), path };
  }

  async disconnect(): Promise<void> {
    if (this.transport) {
      await this.transport.close();
      this.transport = null;
      this.app = null;
    }
  }

  async signAmino(path: string, signDoc: any) {
    if (!this.app) {
      throw new Error('Ledger not connected');
    }
    if (this.transport?.requireBiometric) {
      await this.transport.requireBiometric();
    }
    const normPath = normalizePath(path, DEFAULT_MAX_ACCOUNT_INDEX);
    validateFee(signDoc.fee, DEFAULT_ALLOWED_FEE_DENOMS);
    if (!signDoc.chain_id) {
      throw new Error('chain_id is required for Ledger signing');
    }
    if (signDoc.chain_id !== signDoc.chainId && signDoc.chainId) {
      throw new Error('chain_id mismatch in sign doc');
    }
    this.validateMsgAddresses(signDoc.msgs);
    const res = await this.app.sign(normPath, JSON.stringify(signDoc));
    return {
      signature: Buffer.from(res.signature, 'base64'),
    };
  }

  private validateMsgAddresses(msgs: any[] = []) {
    msgs.forEach((msg) => {
      const value = msg?.value || msg;
      if (value && typeof value === 'object') {
        Object.entries(value).forEach(([key, val]) => {
          if (typeof val === 'string' && key.toLowerCase().includes('address')) {
            assertBech32Prefix(val, DEFAULT_PREFIX);
          }
        });
      }
    });
  }
}

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.startsWith('0x') ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < clean.length; i += 2) {
    out[i / 2] = parseInt(clean.substr(i, 2), 16);
  }
  return out;
}
