import { signAmino } from './hardware/ledger.js';
import { bech32 } from 'bech32';

const DEFAULT_ALLOWED_FEE_DENOMS = ['upaw'];

export function normalizeAddress(addr) {
  try {
    const decoded = bech32.decode(addr);
    return bech32.encode(decoded.prefix, decoded.words);
  } catch (err) {
    throw new Error(`Invalid bech32 address: ${err.message}`);
  }
}

export function shouldUseHardware(requestedAddress, hardwareState) {
  if (!hardwareState?.address) return false;
  try {
    return normalizeAddress(requestedAddress) === normalizeAddress(hardwareState.address);
  } catch {
    return false;
  }
}

function validateFee(fee, allowedFeeDenoms = DEFAULT_ALLOWED_FEE_DENOMS) {
  if (!fee?.amount || !Array.isArray(fee.amount) || fee.amount.length === 0) {
    throw new Error('Fee amount required');
  }
  if (!fee.gas || Number.isNaN(Number(fee.gas)) || Number(fee.gas) <= 0) {
    throw new Error('Invalid gas');
  }
  for (const coin of fee.amount) {
    if (!allowedFeeDenoms.includes(coin.denom)) {
      throw new Error(`Fee denom ${coin.denom} not permitted`);
    }
    if (Number.isNaN(Number(coin.amount)) || Number(coin.amount) < 0) {
      throw new Error('Fee amount must be non-negative');
    }
  }
}

function toLegacyMsgs(msgs = []) {
  return msgs.map(msg => {
    const { ['@type']: type, ...rest } = msg;
    return {
      type: type || 'cosmos-sdk/MsgSend',
      value: rest,
    };
  });
}

export async function signAminoRequest({
  signDoc,
  address,
  chainId,
  hardwareState,
  allowedFeeDenoms = DEFAULT_ALLOWED_FEE_DENOMS,
}) {
  if (!hardwareState?.address) {
    throw new Error('No hardware wallet connected');
  }
  if (!shouldUseHardware(address, hardwareState)) {
    throw new Error('Hardware address does not match request');
  }
  if (!signDoc?.chain_id || signDoc.chain_id !== chainId) {
    throw new Error(`Chain-id mismatch (${signDoc?.chain_id || 'unknown'} != ${chainId})`);
  }

  validateFee(signDoc.fee, allowedFeeDenoms);

  const res = await signAmino({
    signDoc: {
      ...signDoc,
      msgs: toLegacyMsgs(signDoc.msgs),
    },
    enforceChainId: chainId,
    prefix: bech32.decode(address).prefix,
  });

  return {
    signature: res.signature,
    publicKey: res.publicKey,
    mode: 'hardware',
    transport: hardwareState.transport,
  };
}
