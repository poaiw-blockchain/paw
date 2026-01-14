#!/usr/bin/env node
/**
 * Minimal physical-device smoke test for Ledger hardware signing.
 * - Lists HID devices
 * - Connects to the first Ledger
 * - Retrieves address for m/44'/118'/0'/0/0
 * - Signs a tiny amino signDoc (chain-id + fee only)
 *
 * Requires a Ledger with the Cosmos app open. Safe to run locally; no secrets are stored.
 */

const { createRequire } = require('module');

const requireFrom = (pkg) =>
  require(
    require.resolve(pkg, {
      paths: [
        __dirname + '/../desktop/node_modules',
        __dirname + '/../browser-extension/node_modules',
        __dirname + '/../core/node_modules',
      ],
    })
  );

async function main() {
  const TransportNodeHid = requireFrom('@ledgerhq/hw-transport-node-hid').default || requireFrom('@ledgerhq/hw-transport-node-hid');
  const CosmosApp = requireFrom('@ledgerhq/hw-app-cosmos').default || requireFrom('@ledgerhq/hw-app-cosmos');

  const devices = await TransportNodeHid.list();
  if (!devices.length) {
    console.log('[SKIP] No Ledger device detected over HID; connect a Ledger and open the Cosmos app.');
    return;
  }

  console.log(`[INFO] Found ${devices.length} HID device(s); using the first one.`);
  const transport = await TransportNodeHid.create();
  transport.setExchangeTimeout(30000);
  const app = new CosmosApp(transport);

  const path = "44'/118'/0'/0/0";
  const addressRes = await app.getAddress(path, 'paw', false);
  console.log(`[INFO] Ledger address: ${addressRes.address}`);

  const signDoc = {
    chain_id: 'paw-mvp-1',
    account_number: '0',
    sequence: '0',
    fee: { amount: [{ denom: 'upaw', amount: '0' }], gas: '200000' },
    msgs: [],
    memo: 'physical-ledger-check',
  };

  const sig = await app.sign(path, JSON.stringify(signDoc));
  console.log(`[INFO] Signature (base64, first 32 chars): ${sig.signature.slice(0, 32)}...`);
  await transport.close();
  console.log('[OK] Ledger physical signing check completed');
}

main().catch((err) => {
  console.error('[FAIL] Ledger physical signing check failed:', err.message || err);
  process.exit(1);
});
