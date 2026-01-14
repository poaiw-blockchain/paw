/**
 * PAW SDK DEX Trading Example
 *
 * ⚠️ IMPORTANT: The DEX module is DISABLED in paw-mvp-1 testnet.
 * This example demonstrates the API, but transactions will fail
 * until DEX is enabled via governance proposal.
 *
 * Prerequisites:
 * - Set MNEMONIC environment variable with your wallet mnemonic
 * - DEX module must be enabled on the chain (currently disabled)
 */
import { PawClient, PawWallet, PAW_TESTNET_CONFIG, ModuleDisabledError } from '@paw-chain/sdk';

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

  // Check if DEX module is enabled
  const dexEnabled = await client.dex.isEnabled();
  console.log('DEX module enabled:', dexEnabled);

  if (!dexEnabled) {
    console.log('\n⚠️  DEX module is DISABLED in paw-mvp-1 testnet.');
    console.log('To enable DEX, submit a governance proposal.');
    console.log('');
    console.log('The examples below show the API usage, but will fail with ModuleDisabledError.');
    console.log('');
  }

  // 1. Query pools (will return empty array when disabled)
  console.log('--- DEX Pools ---');
  const pools = await client.dex.getAllPools();
  console.log('Available pools:', pools.length);

  if (pools.length > 0) {
    pools.forEach(pool => {
      console.log(`  Pool ${pool.id}: ${pool.tokenA}/${pool.tokenB}`);
      console.log(`    Reserves: ${pool.reserveA} / ${pool.reserveB}`);
    });
  } else {
    console.log('No pools available (DEX disabled or no pools created)');
  }
  console.log('');

  // 2. Demonstrate offline calculations (these work even when DEX is disabled)
  console.log('--- Offline Calculations (Work Without DEX Enabled) ---');

  // Example pool data for calculations
  const mockPool = {
    reserveA: '1000000000', // 1000 tokens
    reserveB: '500000000',  // 500 tokens
    swapFee: '0.003'        // 0.3% fee
  };

  const amountIn = '1000000'; // 1 token
  const amountOut = client.dex.calculateSwapOutput(
    amountIn,
    mockPool.reserveA,
    mockPool.reserveB,
    mockPool.swapFee
  );
  console.log(`Swap ${amountIn} TokenA for ${amountOut} TokenB`);

  const priceImpact = client.dex.calculatePriceImpact(
    amountIn,
    mockPool.reserveA,
    mockPool.reserveB
  );
  console.log(`Price impact: ${priceImpact.toFixed(4)}%`);

  const shares = client.dex.calculateShares(
    '1000000',
    '500000',
    mockPool.reserveA,
    mockPool.reserveB,
    '100000000' // existing total shares
  );
  console.log(`Liquidity provision would mint ${shares} shares`);
  console.log('');

  // 3. Demonstrate DEX transaction (will fail when disabled)
  console.log('--- DEX Transactions (Require DEX Enabled) ---');

  try {
    // This will throw ModuleDisabledError when DEX is disabled
    const swapResult = await client.dex.swap(address, {
      poolId: '1',
      tokenIn: 'upaw',
      amountIn: '1000000',
      minAmountOut: '900000'
    });
    console.log('Swap successful!');
    console.log(`TX Hash: ${swapResult.transactionHash}`);
  } catch (error) {
    if (error instanceof ModuleDisabledError) {
      console.log('Expected error: DEX module is disabled');
      console.log(`Error message: ${error.message}`);
    } else {
      console.error('Unexpected error:', error);
    }
  }
  console.log('');

  try {
    // Create pool attempt
    const createResult = await client.dex.createPool(address, {
      tokenA: 'upaw',
      tokenB: 'uatom',
      amountA: '10000000',
      amountB: '5000000'
    });
    console.log('Pool created!');
    console.log(`TX Hash: ${createResult.transactionHash}`);
  } catch (error) {
    if (error instanceof ModuleDisabledError) {
      console.log('Expected error: Cannot create pool - DEX disabled');
    } else {
      console.error('Unexpected error:', error);
    }
  }
  console.log('');

  // Summary
  console.log('--- Summary ---');
  console.log('DEX module status: DISABLED');
  console.log('');
  console.log('To enable DEX on paw-mvp-1:');
  console.log('1. Submit a governance proposal to enable the DEX module');
  console.log('2. Wait for voting period to complete');
  console.log('3. If proposal passes, DEX will be enabled');
  console.log('');

  await client.disconnect();
  console.log('Disconnected from testnet');
}

main().catch(console.error);
