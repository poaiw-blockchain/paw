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

  // 1. Get all DEX pools
  const pools = await client.dex.getAllPools();
  console.log('Available pools:', pools.length);

  // 2. Get specific pool
  const pool = await client.dex.getPoolByTokens('upaw', 'uatom');
  if (pool) {
    console.log('PAW/ATOM Pool:', {
      id: pool.id,
      reserveA: pool.reserveA,
      reserveB: pool.reserveB,
      swapFee: pool.swapFee
    });

    // 3. Calculate swap output
    const amountIn = '1000000'; // 1 PAW
    const amountOut = client.dex.calculateSwapOutput(
      amountIn,
      pool.reserveA,
      pool.reserveB,
      pool.swapFee
    );
    console.log('Swap 1 PAW for', amountOut, 'uatom');

    // 4. Calculate price impact
    const priceImpact = client.dex.calculatePriceImpact(
      amountIn,
      pool.reserveA,
      pool.reserveB
    );
    console.log('Price impact:', priceImpact.toFixed(2), '%');

    // 5. Execute swap
    const swapResult = await client.dex.swap(address, {
      poolId: pool.id,
      tokenIn: 'upaw',
      amountIn: amountIn,
      minAmountOut: (BigInt(amountOut) * 95n / 100n).toString() // 5% slippage
    });
    console.log('Swap successful! TX:', swapResult.transactionHash);
  }

  // 6. Create new pool
  const createPoolResult = await client.dex.createPool(address, {
    tokenA: 'upaw',
    tokenB: 'uosmo',
    amountA: '10000000', // 10 PAW
    amountB: '5000000'   // 5 OSMO
  });
  console.log('Pool created! TX:', createPoolResult.transactionHash);

  // 7. Add liquidity to existing pool
  if (pool) {
    const addLiquidityResult = await client.dex.addLiquidity(address, {
      poolId: pool.id,
      amountA: '1000000',
      amountB: '1000000',
      minShares: '0'
    });
    console.log('Liquidity added! TX:', addLiquidityResult.transactionHash);
  }

  await client.disconnect();
}

main().catch(console.error);
