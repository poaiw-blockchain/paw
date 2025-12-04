import * as bip39 from 'bip39';
import {ec as EC} from 'elliptic';
import CryptoJS from 'crypto-js';
import {sha256} from 'js-sha256';
import Ripemd160 from 'ripemd160';
import {bech32} from 'bech32';

const curve = new EC('secp256k1');
const DEFAULT_PATH = "m/44'/118'/0'/0/0";

export function generateMnemonic(strength = 256) {
  return bip39.generateMnemonic(strength);
}

export function validateMnemonic(mnemonic) {
  return bip39.validateMnemonic(mnemonic);
}

export function derivePrivateKeyFromMnemonic(mnemonic, path = DEFAULT_PATH) {
  if (!validateMnemonic(mnemonic)) {
    throw new Error('Invalid mnemonic phrase');
  }
  const seed = bip39.mnemonicToSeedSync(mnemonic);
  // Use the first 32 bytes of the seed as a deterministic private key
  return seed.slice(0, 32).toString('hex');
}

export function getPublicKey(privateKeyHex) {
  const keyPair = curve.keyFromPrivate(privateKeyHex, 'hex');
  const publicKey = keyPair.getPublic(true, 'hex');
  return publicKey;
}

export function deriveAddress(publicKeyHex, prefix = 'paw') {
  const publicKey = Buffer.from(publicKeyHex, 'hex');
  const shaHash = Buffer.from(sha256.arrayBuffer(publicKey));
  const ripe = new Ripemd160().update(shaHash).digest();
  const words = bech32.toWords(ripe);
  return bech32.encode(prefix, words);
}

export function generateWallet() {
  const mnemonic = generateMnemonic();
  const privateKey = derivePrivateKeyFromMnemonic(mnemonic);
  const publicKey = getPublicKey(privateKey);
  const address = deriveAddress(publicKey);
  return {
    mnemonic,
    privateKey,
    publicKey,
    address,
  };
}

export function importWalletFromMnemonic(mnemonic) {
  const privateKey = derivePrivateKeyFromMnemonic(mnemonic);
  const publicKey = getPublicKey(privateKey);
  const address = deriveAddress(publicKey);
  return {
    mnemonic,
    privateKey,
    publicKey,
    address,
  };
}

export function importWalletFromPrivateKey(privateKey) {
  if (!privateKey || privateKey.length !== 64) {
    throw new Error('Invalid private key');
  }
  const publicKey = getPublicKey(privateKey);
  const address = deriveAddress(publicKey);
  return {
    privateKey,
    publicKey,
    address,
  };
}

export function signMessage(message, privateKeyHex) {
  const keyPair = curve.keyFromPrivate(privateKeyHex, 'hex');
  const messageHash = Buffer.from(sha256.arrayBuffer(Buffer.from(message)));
  const signature = keyPair.sign(messageHash, {canonical: true});
  return {
    r: signature.r.toString('hex'),
    s: signature.s.toString('hex'),
    recoveryParam: signature.recoveryParam ?? 0,
  };
}

export function verifySignature(message, signature, publicKeyHex) {
  const keyPair = curve.keyFromPublic(publicKeyHex, 'hex');
  const messageHash = Buffer.from(sha256.arrayBuffer(Buffer.from(message)));
  return keyPair.verify(messageHash, {
    r: signature.r,
    s: signature.s,
  });
}

export function encrypt(data, password) {
  return CryptoJS.AES.encrypt(data, password).toString();
}

export function decrypt(ciphertext, password) {
  try {
    const bytes = CryptoJS.AES.decrypt(ciphertext, password);
    const decrypted = bytes.toString(CryptoJS.enc.Utf8);
    return decrypted || '';
  } catch (error) {
    return '';
  }
}
