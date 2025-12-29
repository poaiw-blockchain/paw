/**
 * Transaction signing utilities for PAW wallet
 * Implements Cosmos SDK Amino signing format
 */

import {sha256} from 'js-sha256';
import {ec as EC} from 'elliptic';
import {Buffer} from 'buffer';

const curve = new EC('secp256k1');

/**
 * Sort object keys recursively for canonical JSON (Amino signing)
 * @param {any} obj - Object to sort
 * @returns {any} Sorted object
 */
function sortObject(obj) {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    return obj.map(sortObject);
  }
  const sorted = {};
  Object.keys(obj)
    .sort()
    .forEach(key => {
      sorted[key] = sortObject(obj[key]);
    });
  return sorted;
}

/**
 * Create canonical JSON bytes for Amino signing
 * @param {Object} signDoc - The sign document
 * @returns {Uint8Array} Canonical JSON bytes
 */
function makeSignBytes(signDoc) {
  const sorted = sortObject(signDoc);
  const jsonString = JSON.stringify(sorted);
  return new TextEncoder().encode(jsonString);
}

/**
 * Sign a Cosmos SDK Amino sign document
 * @param {Object} signDoc - Amino sign document with chain_id, account_number, sequence, fee, msgs, memo
 * @param {string} privateKeyHex - Hex-encoded private key
 * @returns {Object} Signature object with signature and pub_key
 */
export function signAmino(signDoc, privateKeyHex) {
  // Create sign bytes (canonical JSON)
  const signBytes = makeSignBytes(signDoc);

  // SHA256 hash
  const hash = Buffer.from(sha256.arrayBuffer(signBytes));

  // Sign with secp256k1
  const keyPair = curve.keyFromPrivate(privateKeyHex, 'hex');
  const signature = keyPair.sign(hash, {canonical: true});

  // Get compressed public key (33 bytes)
  const publicKey = keyPair.getPublic(true, 'hex');

  // Convert signature to fixed 64-byte format (r: 32 bytes, s: 32 bytes)
  const rBytes = signature.r.toArray('be', 32);
  const sBytes = signature.s.toArray('be', 32);
  const signatureBytes = new Uint8Array([...rBytes, ...sBytes]);

  return {
    signature: Buffer.from(signatureBytes).toString('base64'),
    pub_key: {
      type: 'tendermint/PubKeySecp256k1',
      value: Buffer.from(publicKey, 'hex').toString('base64'),
    },
  };
}

/**
 * Build a broadcast-ready signed transaction
 * @param {Object} signDoc - Amino sign document
 * @param {Object} signResult - Result from signAmino
 * @returns {Object} StdTx ready for broadcast
 */
export function buildSignedTx(signDoc, signResult) {
  return {
    tx: {
      msg: signDoc.msgs,
      fee: signDoc.fee,
      signatures: [
        {
          signature: signResult.signature,
          pub_key: signResult.pub_key,
        },
      ],
      memo: signDoc.memo || '',
    },
    mode: 'sync',
  };
}

/**
 * Encode transaction to bytes for REST broadcast endpoint
 * @param {Object} stdTx - The signed StdTx object
 * @returns {string} Base64 encoded transaction bytes
 */
export function encodeTxForBroadcast(stdTx) {
  // For the REST /cosmos/tx/v1beta1/txs endpoint, we need to use the newer format
  // This is a simplified version that works with the legacy Amino endpoint
  const jsonString = JSON.stringify(stdTx.tx);
  return Buffer.from(jsonString).toString('base64');
}

/**
 * Build and sign a complete send transaction
 * @param {Object} params - Transaction parameters
 * @param {string} params.fromAddress - Sender address
 * @param {string} params.toAddress - Recipient address
 * @param {string} params.amount - Amount in base units (upaw)
 * @param {string} params.denom - Token denomination
 * @param {string} params.chainId - Chain ID
 * @param {string} params.accountNumber - Account number from chain
 * @param {string} params.sequence - Account sequence from chain
 * @param {string} params.privateKey - Hex-encoded private key
 * @param {string} params.memo - Optional memo
 * @param {Object} params.fee - Optional fee override
 * @returns {Object} Signed transaction ready for broadcast
 */
export function buildAndSignSendTx(params) {
  const {
    fromAddress,
    toAddress,
    amount,
    denom = 'upaw',
    chainId,
    accountNumber,
    sequence,
    privateKey,
    memo = '',
    fee = {amount: [{denom: 'upaw', amount: '2500'}], gas: '200000'},
  } = params;

  // Build Amino sign document
  const signDoc = {
    chain_id: chainId,
    account_number: accountNumber,
    sequence: sequence,
    fee: fee,
    msgs: [
      {
        type: 'cosmos-sdk/MsgSend',
        value: {
          from_address: fromAddress,
          to_address: toAddress,
          amount: [{denom, amount: amount.toString()}],
        },
      },
    ],
    memo: memo,
  };

  // Sign the transaction
  const signResult = signAmino(signDoc, privateKey);

  // Build the signed transaction
  return buildSignedTx(signDoc, signResult);
}
