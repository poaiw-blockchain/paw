/**
 * PAW SDK Basic Usage Example
 *
 * This example demonstrates how to connect to the PAW testnet
 * and perform basic operations.
 */
import { PawClient, PawWallet, PAW_TESTNET_CONFIG } from '@paw-chain/sdk';

async function main() {
  // 1. Create a new wallet
  const mnemonic = PawWallet.generateMnemonic();
  console.log('Generated mnemonic:', mnemonic);
  console.log('WARNING: Store this mnemonic securely!');
  console.log('');

  const wallet = new PawWallet('paw');
  await wallet.fromMnemonic(mnemonic);

  const address = await wallet.getAddress();
  console.log('Wallet address:', address);
  console.log('');

  // 2. Connect to the PAW testnet (paw-mvp-1)
  // Use the built-in testnet configuration
  const client = new PawClient(PAW_TESTNET_CONFIG);

  // Or specify custom endpoints:
  // const client = new PawClient({
  //   rpcEndpoint: 'https://testnet-rpc.poaiw.org',
  //   restEndpoint: 'https://testnet-api.poaiw.org',
  //   chainId: 'paw-mvp-1'
  // });

  await client.connect();
  console.log('Connected to chain:', await client.getChainId());
  console.log('Current block height:', await client.getHeight());
  console.log('');

  // 3. Query validators
  console.log('--- Validators ---');
  const validators = await client.staking.getValidators();
  for (const validator of validators) {
    console.log(`  ${validator.description.moniker}: ${validator.operator_address}`);
    console.log(`    Status: ${validator.status}, Tokens: ${validator.tokens}`);
    console.log(`    Commission: ${(parseFloat(validator.commission.commission_rates.rate) * 100).toFixed(2)}%`);
  }
  console.log('');

  // 4. Check balance (will be empty for new wallets)
  console.log('--- Balance ---');
  const balance = await client.bank.getBalance(address, 'upaw');
  console.log('Balance:', balance ? `${balance.amount} ${balance.denom}` : '0 upaw');

  // To get tokens, use the faucet: https://testnet-faucet.poaiw.org
  console.log('');
  console.log('Get testnet tokens at: https://testnet-faucet.poaiw.org');
  console.log('');

  // 5. Query governance proposals
  console.log('--- Governance Proposals ---');
  const proposals = await client.governance.getProposals();
  if (proposals.length === 0) {
    console.log('No active proposals');
  } else {
    for (const proposal of proposals) {
      console.log(`  Proposal #${proposal.proposal_id}: ${proposal.content?.title || 'Untitled'}`);
    }
  }
  console.log('');

  // 6. DEX module (disabled in testnet)
  console.log('--- DEX Module Status ---');
  const dexEnabled = await client.dex.isEnabled();
  console.log(`DEX module enabled: ${dexEnabled}`);
  if (!dexEnabled) {
    console.log('DEX module is disabled in paw-mvp-1. Enable via governance proposal.');
  }
  console.log('');

  // 7. Disconnect
  await client.disconnect();
  console.log('Disconnected from testnet');
}

main().catch(console.error);
