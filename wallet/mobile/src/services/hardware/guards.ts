import { bech32 } from 'bech32';

export function assertBech32Prefix(address: string, expectedPrefix: string) {
  const decoded = bech32.decode(address);
  if (decoded.prefix !== expectedPrefix) {
    throw new Error(`Address prefix mismatch: expected ${expectedPrefix}, got ${decoded.prefix}`);
  }
}

export function validateFee(
  fee: { amount?: Array<{ denom?: string; amount?: string }>; gas?: string },
  allowedFeeDenoms: string[] = ['upaw']
) {
  if (!fee?.amount || !Array.isArray(fee.amount) || fee.amount.length === 0) {
    throw new Error('Fee amount required');
  }
  if (!fee.gas || Number.isNaN(Number(fee.gas)) || Number(fee.gas) <= 0) {
    throw new Error('Invalid gas');
  }
  for (const coin of fee.amount) {
    if (!coin?.denom || !allowedFeeDenoms.includes(coin.denom)) {
      throw new Error(`Fee denom ${coin?.denom || 'unknown'} not permitted`);
    }
    if (Number.isNaN(Number(coin.amount)) || Number(coin.amount) < 0) {
      throw new Error('Fee amount must be non-negative');
    }
  }
}

export function normalizePath(path: string, maxAccount = 4): string {
  const sanitized = path.startsWith('m/') ? path.slice(2) : path;
  const segments = sanitized.split('/');
  if (segments.length !== 5) {
    throw new Error(`Invalid derivation path: ${path}`);
  }
  const accountSeg = segments[2];
  const account = parseInt(accountSeg.replace("'", ''), 10);
  if (Number.isNaN(account) || account > maxAccount) {
    throw new Error(`Account index exceeds maximum (${maxAccount})`);
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
