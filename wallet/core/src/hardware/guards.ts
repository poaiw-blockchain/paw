import { fromBech32 } from '@cosmjs/encoding';
import { HardwareWalletConfig } from './types';

export function assertBech32Prefix(address: string, expectedPrefix: string): void {
  const decoded = fromBech32(address);
  if (decoded.prefix !== expectedPrefix) {
    throw new Error(`Address prefix mismatch: expected ${expectedPrefix}, got ${decoded.prefix}`);
  }
}

export function validateSignDocBasics(
  doc: {
    chain_id?: string;
    fee?: { amount?: Array<{ denom?: string; amount?: string }>; gas?: string };
  },
  {
    enforceChainId,
    allowedFeeDenoms = ['upaw'],
  }: Pick<HardwareWalletConfig, 'enforceChainId' | 'allowedFeeDenoms'>
): void {
  if (!doc.chain_id) {
    throw new Error('chain_id is required for hardware signing');
  }

  if (enforceChainId && doc.chain_id !== enforceChainId) {
    throw new Error(`Refusing to sign: chain-id mismatch (${doc.chain_id} != ${enforceChainId})`);
  }

  const gas = doc.fee?.gas;
  if (!gas || Number.isNaN(Number(gas)) || Number(gas) <= 0) {
    throw new Error('Invalid or missing gas value');
  }

  const amounts = doc.fee?.amount || [];
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
