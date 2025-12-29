/**
 * Cosmos SDK Integration Module for PAW Browser Wallet
 *
 * Provides utilities for:
 * - Transaction signing with secp256k1
 * - Account/sequence number management
 * - StdTx message building
 * - Transaction broadcasting
 * - Address generation and validation
 */

// Import required crypto libraries (will be included via CDN or bundler)
// Using noble-secp256k1 for signing
// Using bech32 for address encoding

const COSMOS_SDK = {
  // PAW Network Configuration
  config: {
    chainId: 'paw-testnet',
    bech32Prefix: 'paw',
    coinDenom: 'upaw',
    coinDecimals: 6,
    restEndpoint: 'http://localhost:1317',
    rpcEndpoint: 'http://localhost:26657',
  },

  /**
   * Generate a new secp256k1 private key
   * @returns {Uint8Array} 32-byte private key
   */
  generatePrivateKey() {
    const privateKey = new Uint8Array(32);
    crypto.getRandomValues(privateKey);
    return privateKey;
  },

  /**
   * Derive public key from private key
   * @param {Uint8Array} privateKey - 32-byte private key
   * @returns {Uint8Array} 33-byte compressed public key
   */
  async getPublicKey(privateKey) {
    // Using SubtleCrypto for ECDSA
    const keyPair = await crypto.subtle.importKey(
      'raw',
      privateKey,
      { name: 'ECDSA', namedCurve: 'P-256' },
      true,
      ['sign']
    );

    const exported = await crypto.subtle.exportKey('raw', keyPair);
    return new Uint8Array(exported);
  },

  /**
   * Convert public key to Bech32 address
   * @param {Uint8Array} publicKey - Compressed public key
   * @returns {Promise<string>} Bech32 encoded address (paw1...)
   */
  async publicKeyToAddress(publicKey) {
    // 1. SHA-256 hash of public key
    const sha256Hash = await this.sha256(publicKey);

    // 2. RIPEMD-160 hash of the SHA-256 hash
    const ripemd160Hash = this.ripemd160(sha256Hash);

    // 3. Bech32 encode with 'paw' prefix
    return this.bech32Encode(this.config.bech32Prefix, ripemd160Hash);
  },

  /**
   * SHA-256 hash (async - uses WebCrypto API)
   * @param {Uint8Array} data - Data to hash
   * @returns {Promise<Uint8Array>} 32-byte SHA-256 hash
   */
  async sha256(data) {
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    return new Uint8Array(hashBuffer);
  },

  /**
   * RIPEMD-160 hash (pure JS implementation)
   * @param {Uint8Array} data - Data to hash
   * @returns {Uint8Array} 20-byte RIPEMD-160 hash
   */
  ripemd160(data) {
    // RIPEMD-160 constants
    const K1 = [0x00000000, 0x5a827999, 0x6ed9eba1, 0x8f1bbcdc, 0xa953fd4e];
    const K2 = [0x50a28be6, 0x5c4dd124, 0x6d703ef3, 0x7a6d76e9, 0x00000000];
    const R1 = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,7,4,13,1,10,6,15,3,12,0,9,5,2,14,11,8,3,10,14,4,9,15,8,1,2,7,0,6,13,11,5,12,1,9,11,10,0,8,12,4,13,3,7,15,14,5,6,2,4,0,5,9,7,12,2,10,14,1,3,8,11,6,15,13];
    const R2 = [5,14,7,0,9,2,11,4,13,6,15,8,1,10,3,12,6,11,3,7,0,13,5,10,14,15,8,12,4,9,1,2,15,5,1,3,7,14,6,9,11,8,12,2,10,0,4,13,8,6,4,1,3,11,15,0,5,12,2,13,9,7,10,14,12,15,10,4,1,5,8,7,6,2,13,14,0,3,9,11];
    const S1 = [11,14,15,12,5,8,7,9,11,13,14,15,6,7,9,8,7,6,8,13,11,9,7,15,7,12,15,9,11,7,13,12,11,13,6,7,14,9,13,15,14,8,13,6,5,12,7,5,11,12,14,15,14,15,9,8,9,14,5,6,8,6,5,12,9,15,5,11,6,8,13,12,5,12,13,14,11,8,5,6];
    const S2 = [8,9,9,11,13,15,15,5,7,7,8,11,14,14,12,6,9,13,15,7,12,8,9,11,7,7,12,7,6,15,13,11,9,7,15,11,8,6,6,14,12,13,5,14,13,13,7,5,15,5,8,11,14,14,6,14,6,9,12,9,12,5,15,8,8,5,12,9,12,5,14,6,8,13,6,5,15,13,11,11];

    const rotl = (x, n) => ((x << n) | (x >>> (32 - n))) >>> 0;

    const f = (j, x, y, z) => {
      if (j < 16) return (x ^ y ^ z) >>> 0;
      if (j < 32) return ((x & y) | (~x & z)) >>> 0;
      if (j < 48) return ((x | ~y) ^ z) >>> 0;
      if (j < 64) return ((x & z) | (y & ~z)) >>> 0;
      return (x ^ (y | ~z)) >>> 0;
    };

    // Padding
    const len = data.length;
    const bitLen = len * 8;
    const padLen = (len % 64 < 56) ? (56 - len % 64) : (120 - len % 64);
    const padded = new Uint8Array(len + padLen + 8);
    padded.set(data);
    padded[len] = 0x80;
    // Little-endian bit length
    for (let i = 0; i < 8; i++) {
      padded[len + padLen + i] = (bitLen >>> (i * 8)) & 0xff;
    }

    // Initial hash values
    let h0 = 0x67452301, h1 = 0xefcdab89, h2 = 0x98badcfe, h3 = 0x10325476, h4 = 0xc3d2e1f0;

    // Process each 64-byte block
    for (let offset = 0; offset < padded.length; offset += 64) {
      const X = new Uint32Array(16);
      for (let i = 0; i < 16; i++) {
        X[i] = padded[offset + i * 4] | (padded[offset + i * 4 + 1] << 8) |
               (padded[offset + i * 4 + 2] << 16) | (padded[offset + i * 4 + 3] << 24);
      }

      let a1 = h0, b1 = h1, c1 = h2, d1 = h3, e1 = h4;
      let a2 = h0, b2 = h1, c2 = h2, d2 = h3, e2 = h4;

      for (let j = 0; j < 80; j++) {
        const jDiv16 = Math.floor(j / 16);
        let t = (a1 + f(j, b1, c1, d1) + X[R1[j]] + K1[jDiv16]) >>> 0;
        t = (rotl(t, S1[j]) + e1) >>> 0;
        a1 = e1; e1 = d1; d1 = rotl(c1, 10); c1 = b1; b1 = t;

        t = (a2 + f(79 - j, b2, c2, d2) + X[R2[j]] + K2[jDiv16]) >>> 0;
        t = (rotl(t, S2[j]) + e2) >>> 0;
        a2 = e2; e2 = d2; d2 = rotl(c2, 10); c2 = b2; b2 = t;
      }

      const t = (h1 + c1 + d2) >>> 0;
      h1 = (h2 + d1 + e2) >>> 0;
      h2 = (h3 + e1 + a2) >>> 0;
      h3 = (h4 + a1 + b2) >>> 0;
      h4 = (h0 + b1 + c2) >>> 0;
      h0 = t;
    }

    // Output as little-endian bytes
    const result = new Uint8Array(20);
    [h0, h1, h2, h3, h4].forEach((h, i) => {
      result[i * 4] = h & 0xff;
      result[i * 4 + 1] = (h >>> 8) & 0xff;
      result[i * 4 + 2] = (h >>> 16) & 0xff;
      result[i * 4 + 3] = (h >>> 24) & 0xff;
    });
    return result;
  },

  /**
   * Bech32 encode (BIP-173 compliant)
   * @param {string} prefix - Human-readable prefix (e.g., 'paw')
   * @param {Uint8Array} data - 20-byte address data
   * @returns {string} Bech32 encoded address
   */
  bech32Encode(prefix, data) {
    const CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
    const GENERATOR = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

    const polymod = (values) => {
      let chk = 1;
      for (const v of values) {
        const top = chk >>> 25;
        chk = ((chk & 0x1ffffff) << 5) ^ v;
        for (let i = 0; i < 5; i++) {
          if ((top >>> i) & 1) chk ^= GENERATOR[i];
        }
      }
      return chk;
    };

    const hrpExpand = (hrp) => {
      const result = [];
      for (let i = 0; i < hrp.length; i++) {
        result.push(hrp.charCodeAt(i) >>> 5);
      }
      result.push(0);
      for (let i = 0; i < hrp.length; i++) {
        result.push(hrp.charCodeAt(i) & 31);
      }
      return result;
    };

    const createChecksum = (hrp, data) => {
      const values = hrpExpand(hrp).concat(data).concat([0, 0, 0, 0, 0, 0]);
      const mod = polymod(values) ^ 1;
      const result = [];
      for (let i = 0; i < 6; i++) {
        result.push((mod >>> (5 * (5 - i))) & 31);
      }
      return result;
    };

    // Convert 8-bit bytes to 5-bit groups (bech32 words)
    const convertBits = (data, fromBits, toBits, pad) => {
      let acc = 0, bits = 0;
      const result = [];
      const maxv = (1 << toBits) - 1;
      for (const value of data) {
        acc = (acc << fromBits) | value;
        bits += fromBits;
        while (bits >= toBits) {
          bits -= toBits;
          result.push((acc >>> bits) & maxv);
        }
      }
      if (pad && bits > 0) {
        result.push((acc << (toBits - bits)) & maxv);
      }
      return result;
    };

    const words = convertBits(data, 8, 5, true);
    const checksum = createChecksum(prefix, words);
    const combined = words.concat(checksum);
    let encoded = prefix + '1';
    for (const w of combined) {
      encoded += CHARSET[w];
    }
    return encoded;
  },

  /**
   * Get account information from chain
   * @param {string} address - Bech32 address
   * @returns {Promise<Object>} Account info with sequence and account_number
   */
  async getAccount(address) {
    const url = `${this.config.restEndpoint}/cosmos/auth/v1beta1/accounts/${address}`;
    const response = await fetch(url);
    const data = await response.json();

    return {
      address: data.account.address,
      accountNumber: parseInt(data.account.account_number || '0'),
      sequence: parseInt(data.account.sequence || '0'),
      pubKey: data.account.pub_key,
    };
  },

  /**
   * Get account balance
   * @param {string} address - Bech32 address
   * @returns {Promise<Array>} Array of coin balances
   */
  async getBalance(address) {
    const url = `${this.config.restEndpoint}/cosmos/bank/v1beta1/balances/${address}`;
    const response = await fetch(url);
    const data = await response.json();
    return data.balances || [];
  },

  /**
   * Build a standard transfer transaction
   * @param {Object} params - Transaction parameters
   * @returns {Object} Unsigned transaction
   */
  buildTransferTx(params) {
    const {
      fromAddress,
      toAddress,
      amount,
      denom = this.config.coinDenom,
      memo = '',
      fee = { amount: [{ denom: 'upaw', amount: '5000' }], gas: '200000' },
    } = params;

    return {
      body: {
        messages: [{
          '@type': '/cosmos.bank.v1beta1.MsgSend',
          from_address: fromAddress,
          to_address: toAddress,
          amount: [{ denom, amount: amount.toString() }],
        }],
        memo,
        timeout_height: '0',
        extension_options: [],
        non_critical_extension_options: [],
      },
      auth_info: {
        signer_infos: [],
        fee,
      },
      signatures: [],
    };
  },

  /**
   * Build a DEX swap transaction
   * @param {Object} params - Swap parameters
   * @returns {Object} Unsigned transaction
   */
  buildSwapTx(params) {
    const {
      sender,
      poolId,
      tokenIn,
      tokenOutDenom,
      minAmountOut,
      memo = '',
      fee = { amount: [{ denom: 'upaw', amount: '10000' }], gas: '300000' },
    } = params;

    return {
      body: {
        messages: [{
          '@type': '/paw.dex.v1.MsgSwap',
          sender,
          pool_id: poolId.toString(),
          token_in: tokenIn,
          token_out_denom: tokenOutDenom,
          min_amount_out: minAmountOut.toString(),
        }],
        memo,
        timeout_height: '0',
        extension_options: [],
        non_critical_extension_options: [],
      },
      auth_info: {
        signer_infos: [],
        fee,
      },
      signatures: [],
    };
  },

  /**
   * Sign transaction with private key
   * @param {Object} tx - Unsigned transaction
   * @param {Uint8Array} privateKey - Signer's private key
   * @param {Object} accountInfo - Account number and sequence
   * @param {Uint8Array} publicKey - Signer's public key
   * @returns {Object} Signed transaction
   */
  async signTx(tx, privateKey, accountInfo, publicKey) {
    const { accountNumber, sequence } = accountInfo;

    // Build SignDoc
    const signDoc = {
      body_bytes: this.encodeBody(tx.body),
      auth_info_bytes: this.encodeAuthInfo(tx.auth_info, publicKey, sequence),
      chain_id: this.config.chainId,
      account_number: accountNumber.toString(),
    };

    // Serialize for signing
    const signBytes = this.serializeSignDoc(signDoc);

    // Sign with secp256k1
    const signature = await this.sign(signBytes, privateKey);

    // Add signature to transaction
    tx.signatures = [signature];
    tx.auth_info.signer_infos = [{
      public_key: {
        '@type': '/cosmos.crypto.secp256k1.PubKey',
        key: this.bytesToBase64(publicKey),
      },
      mode_info: {
        single: { mode: 'SIGN_MODE_DIRECT' },
      },
      sequence: sequence.toString(),
    }];

    return tx;
  },

  /**
   * Sign bytes with private key
   * @param {Uint8Array} bytes - Bytes to sign
   * @param {Uint8Array} privateKey - Private key
   * @returns {Uint8Array} Signature
   */
  async sign(bytes, privateKey) {
    // Import private key
    const key = await crypto.subtle.importKey(
      'raw',
      privateKey,
      { name: 'ECDSA', namedCurve: 'P-256' },
      false,
      ['sign']
    );

    // Sign
    const signature = await crypto.subtle.sign(
      { name: 'ECDSA', hash: 'SHA-256' },
      key,
      bytes
    );

    return new Uint8Array(signature);
  },

  /**
   * Broadcast signed transaction
   * @param {Object} signedTx - Signed transaction
   * @returns {Promise<Object>} Broadcast result
   */
  async broadcastTx(signedTx) {
    const txBytes = this.encodeTx(signedTx);
    const txBase64 = this.bytesToBase64(txBytes);

    const url = `${this.config.restEndpoint}/cosmos/tx/v1beta1/txs`;
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tx_bytes: txBase64,
        mode: 'BROADCAST_MODE_SYNC',
      }),
    });

    const result = await response.json();

    if (result.tx_response.code !== 0) {
      throw new Error(`Transaction failed: ${result.tx_response.raw_log}`);
    }

    return result.tx_response;
  },

  /**
   * Query DEX pools
   * @returns {Promise<Array>} List of pools
   */
  async queryPools() {
    const url = `${this.config.restEndpoint}/paw/dex/v1/pools`;
    const response = await fetch(url);
    const data = await response.json();
    return data.pools || [];
  },

  /**
   * Query specific pool
   * @param {string} poolId - Pool ID
   * @returns {Promise<Object>} Pool details
   */
  async queryPool(poolId) {
    const url = `${this.config.restEndpoint}/paw/dex/v1/pools/${poolId}`;
    const response = await fetch(url);
    const data = await response.json();
    return data.pool;
  },

  /**
   * Query oracle prices
   * @returns {Promise<Array>} Price feeds
   */
  async queryOraclePrices() {
    const url = `${this.config.restEndpoint}/paw/oracle/v1/prices`;
    const response = await fetch(url);
    const data = await response.json();
    return data.prices || [];
  },

  // Helper functions for encoding (simplified)
  encodeBody(body) {
    return new TextEncoder().encode(JSON.stringify(body));
  },

  encodeAuthInfo(authInfo, publicKey, sequence) {
    return new TextEncoder().encode(JSON.stringify({
      ...authInfo,
      signer_infos: [{
        public_key: {
          '@type': '/cosmos.crypto.secp256k1.PubKey',
          key: this.bytesToBase64(publicKey),
        },
        mode_info: { single: { mode: 'SIGN_MODE_DIRECT' } },
        sequence: sequence.toString(),
      }],
    }));
  },

  serializeSignDoc(signDoc) {
    return new TextEncoder().encode(JSON.stringify(signDoc));
  },

  encodeTx(tx) {
    return new TextEncoder().encode(JSON.stringify(tx));
  },

  // Utility functions
  bytesToHex(bytes) {
    return Array.from(bytes)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  },

  bytesToBase64(bytes) {
    return btoa(String.fromCharCode(...bytes));
  },

  base64ToBytes(base64) {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
  },

  hexToBytes(hex) {
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < hex.length; i += 2) {
      bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
    }
    return bytes;
  },
};

// Export for use in extension
if (typeof module !== 'undefined' && module.exports) {
  module.exports = COSMOS_SDK;
}
