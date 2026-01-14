/**
 * PAW SDK Staking Example
 *
 * This example demonstrates how to interact with the staking module
 * on the PAW testnet (paw-mvp-1).
 *
 * Prerequisites:
 * - Set MNEMONIC environment variable with your wallet mnemonic
 * - Have testnet tokens (get from https://testnet-faucet.poaiw.org)
 */
import { PawClient, PawWallet, PAW_TESTNET_CONFIG } from '@paw-chain/sdk';

async function main() {
  // Setup wallet and client
  if (!process.env.MNEMONIC) {
    console.error('Please set MNEMONIC environment variable');
    process.exit(1);
  }

  const wallet = new PawWallet('paw');
  await wallet.fromMnemonic(process.env.MNEMONIC);
  const address = await wallet.getAddress();

  console.log('Wallet address:', address);
  console.log('');

  // Connect to PAW testnet
  const client = new PawClient(PAW_TESTNET_CONFIG);
  await client.connectWithWallet(wallet);

  // 1. Get all validators
  const validators = await client.staking.getValidators();
  console.log('Active validators:', validators.length);

  // Display validators sorted by voting power
  const sortedValidators = validators
    .sort((a, b) => parseInt(b.tokens) - parseInt(a.tokens));

  console.log('\nValidators by voting power:');
  sortedValidators.forEach((v, i) => {
    console.log(`${i + 1}. ${v.description.moniker}`);
    console.log(`   Address: ${v.operator_address}`);
    console.log(`   Voting power: ${v.tokens}`);
    console.log(`   Commission: ${(parseFloat(v.commission.commission_rates.rate) * 100).toFixed(2)}%`);
    console.log(`   Status: ${v.status}`);
  });

  // Check balance before delegating
  const balance = await client.bank.getBalance(address, 'upaw');
  console.log(`\nYour balance: ${balance?.amount || '0'} upaw`);

  if (!balance || parseInt(balance.amount) < 1000000) {
    console.log('\nInsufficient balance for staking operations.');
    console.log('Get testnet tokens at: https://testnet-faucet.poaiw.org');
    await client.disconnect();
    return;
  }

  // 2. Delegate to a validator
  const validatorAddress = sortedValidators[0].operator_address;
  console.log(`\nDelegating 1 PAW to ${sortedValidators[0].description.moniker}...`);

  const delegateResult = await client.staking.delegate(address, {
    validatorAddress,
    amount: '1000000' // 1 PAW = 1,000,000 upaw
  });
  console.log('Delegation successful!');
  console.log(`TX Hash: ${delegateResult.transactionHash}`);
  console.log(`Gas used: ${delegateResult.gasUsed}`);

  // 3. Get your delegations
  const delegations = await client.staking.getDelegations(address);
  console.log('\nYour delegations:', delegations.length);
  delegations.forEach(d => {
    console.log(`- ${d.delegation.validator_address}: ${d.balance.amount} ${d.balance.denom}`);
  });

  // 4. Get your rewards
  const rewards = await client.staking.getRewards(address);
  console.log('\nPending rewards:', rewards);

  // 5. Withdraw rewards from a validator
  if (rewards.length > 0) {
    console.log('\nWithdrawing rewards...');
    const withdrawResult = await client.staking.withdrawRewards(address, validatorAddress);
    console.log('Rewards withdrawn!');
    console.log(`TX Hash: ${withdrawResult.transactionHash}`);
  }

  // 6. Redelegate to another validator (if multiple validators exist)
  if (sortedValidators.length > 1) {
    const newValidatorAddress = sortedValidators[1].operator_address;
    console.log(`\nRedelegating 0.5 PAW to ${sortedValidators[1].description.moniker}...`);

    const redelegateResult = await client.staking.redelegate(address, {
      srcValidatorAddress: validatorAddress,
      dstValidatorAddress: newValidatorAddress,
      amount: '500000' // 0.5 PAW
    });
    console.log('Redelegation successful!');
    console.log(`TX Hash: ${redelegateResult.transactionHash}`);
  }

  // 7. Undelegate
  console.log('\nUndelegating 0.5 PAW...');
  const undelegateResult = await client.staking.undelegate(address, {
    validatorAddress,
    amount: '500000' // 0.5 PAW
  });
  console.log('Undelegation successful!');
  console.log(`TX Hash: ${undelegateResult.transactionHash}`);
  console.log('Note: Tokens will be available after unbonding period.');

  // 8. Get unbonding delegations
  const unbonding = await client.staking.getUnbondingDelegations(address);
  console.log('\nUnbonding delegations:', unbonding.length);

  // 9. Get staking pool info
  const pool = await client.staking.getPool();
  console.log('\nStaking pool:');
  console.log('- Bonded tokens:', pool?.bonded_tokens);
  console.log('- Not bonded tokens:', pool?.not_bonded_tokens);

  await client.disconnect();
  console.log('\nDisconnected from testnet');
}

main().catch(console.error);
