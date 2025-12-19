import { LedgerServiceMobile } from './hardware';
import { assertBech32Prefix } from './hardware/guards';

const DEFAULT_PREFIX = 'paw';

/**
 * Sign an amino-encoded signDoc via Ledger over BLE.
 * Guards: address prefix check, chain-id match, fee validation handled in LedgerServiceMobile.
 */
export async function signWithLedger(signDoc, accountIndex = 0, expectedPrefix = DEFAULT_PREFIX) {
  if (!signDoc?.msgs || !signDoc?.chain_id) {
    throw new Error('signDoc must include msgs and chain_id');
  }

  const ledger = new LedgerServiceMobile();
  try {
    const { address, path, publicKey } = await ledger.connect(accountIndex);
    assertBech32Prefix(address, expectedPrefix);

    // Optional: ensure signer address matches first msg value if present
    const firstMsg = signDoc.msgs[0]?.value || {};
    const signer = firstMsg.delegator_address || firstMsg.sender || firstMsg.from_address || firstMsg.voter;
    if (signer && signer !== address) {
      throw new Error('Ledger address does not match signer in signDoc');
    }

    const res = await ledger.signAmino(path, signDoc);
    return {
      signature: res.signature,
      address,
      path,
      publicKey,
    };
  } finally {
    await ledger.disconnect();
  }
}
