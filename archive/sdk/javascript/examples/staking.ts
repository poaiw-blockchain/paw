import { PawClient, PawWallet } from '@paw-chain/sdk';

async function main() {
  // Setup wallet and client
  const wallet = new PawWallet('paw');
  await wallet.fromMnemonic(process.env.MNEMONIC!);
  const address = await wallet.getAddress();

  const client = new PawClient({
    rpcEndpoint: process.env.RPC_ENDPOINT || 'http://localhost:26657',
    restEndpoint: process.env.REST_ENDPOINT || 'http://localhost:1317',
    chainId: process.env.CHAIN_ID || 'paw-testnet-1'
  });

  await client.connectWithWallet(wallet);

  // 1. Get all validators
  const validators = await client.staking.getValidators();
  console.log('Active validators:', validators.length);

  // Display top 5 validators by voting power
  const topValidators = validators
    .sort((a, b) => parseInt(b.tokens) - parseInt(a.tokens))
    .slice(0, 5);

  console.log('\nTop 5 Validators:');
  topValidators.forEach((v, i) => {
    console.log(`${i + 1}. ${v.description.moniker}`);
    console.log(`   Voting power: ${v.tokens}`);
    console.log(`   Commission: ${(parseFloat(v.commission.rate) * 100).toFixed(2)}%`);
  });

  // 2. Delegate to a validator
  const validatorAddress = topValidators[0].operatorAddress;
  const delegateResult = await client.staking.delegate(address, {
    validatorAddress,
    amount: '1000000' // 1 PAW
  });
  console.log('\nDelegation successful! TX:', delegateResult.transactionHash);

  // 3. Get your delegations
  const delegations = await client.staking.getDelegations(address);
  console.log('\nYour delegations:', delegations.length);
  delegations.forEach(d => {
    console.log(`- ${d.delegation.validatorAddress}: ${d.balance.amount}`);
  });

  // 4. Get your rewards
  const rewards = await client.staking.getRewards(address);
  console.log('\nPending rewards:', rewards);

  // 5. Withdraw rewards from a validator
  const withdrawResult = await client.staking.withdrawRewards(address, validatorAddress);
  console.log('Rewards withdrawn! TX:', withdrawResult.transactionHash);

  // 6. Redelegate to another validator
  if (topValidators.length > 1) {
    const newValidatorAddress = topValidators[1].operatorAddress;
    const redelegateResult = await client.staking.redelegate(address, {
      srcValidatorAddress: validatorAddress,
      dstValidatorAddress: newValidatorAddress,
      amount: '500000' // 0.5 PAW
    });
    console.log('Redelegation successful! TX:', redelegateResult.transactionHash);
  }

  // 7. Undelegate
  const undelegateResult = await client.staking.undelegate(address, {
    validatorAddress,
    amount: '500000' // 0.5 PAW
  });
  console.log('Undelegation successful! TX:', undelegateResult.transactionHash);

  // 8. Get unbonding delegations
  const unbonding = await client.staking.getUnbondingDelegations(address);
  console.log('\nUnbonding delegations:', unbonding.length);

  // 9. Get staking pool info
  const pool = await client.staking.getPool();
  console.log('\nStaking pool:');
  console.log('- Bonded tokens:', pool?.bondedTokens);
  console.log('- Not bonded tokens:', pool?.notBondedTokens);

  await client.disconnect();
}

main().catch(console.error);
