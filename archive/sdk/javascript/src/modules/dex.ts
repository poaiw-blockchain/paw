import { PawClient } from '../client';
import {
  Pool,
  PoolParams,
  SwapParams,
  AddLiquidityParams,
  RemoveLiquidityParams,
  TxResult,
  GasOptions
} from '../types';

export class DexModule {
  constructor(private client: PawClient) {}

  /**
   * Create a new liquidity pool
   */
  async createPool(
    creator: string,
    params: PoolParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/paw.dex.v1.MsgCreatePool',
      value: {
        creator,
        tokenA: params.tokenA,
        tokenB: params.tokenB,
        amountA: params.amountA,
        amountB: params.amountB
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(creator, [message], options);
  }

  /**
   * Add liquidity to an existing pool
   */
  async addLiquidity(
    sender: string,
    params: AddLiquidityParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/paw.dex.v1.MsgAddLiquidity',
      value: {
        sender,
        poolId: params.poolId,
        amountA: params.amountA,
        amountB: params.amountB,
        minShares: params.minShares
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(sender, [message], options);
  }

  /**
   * Remove liquidity from a pool
   */
  async removeLiquidity(
    sender: string,
    params: RemoveLiquidityParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/paw.dex.v1.MsgRemoveLiquidity',
      value: {
        sender,
        poolId: params.poolId,
        shares: params.shares,
        minAmountA: params.minAmountA,
        minAmountB: params.minAmountB
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(sender, [message], options);
  }

  /**
   * Swap tokens
   */
  async swap(
    sender: string,
    params: SwapParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/paw.dex.v1.MsgSwap',
      value: {
        sender,
        poolId: params.poolId,
        tokenIn: params.tokenIn,
        amountIn: params.amountIn,
        minAmountOut: params.minAmountOut,
        recipient: params.recipient || sender
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(sender, [message], options);
  }

  /**
   * Get pool by ID
   */
  async getPool(poolId: string): Promise<Pool | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/paw/dex/v1/pools/${poolId}`);
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.pool || null;
    } catch (error) {
      console.error('Error fetching pool:', error);
      return null;
    }
  }

  /**
   * Get all pools
   */
  async getAllPools(): Promise<Pool[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/paw/dex/v1/pools`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.pools || [];
    } catch (error) {
      console.error('Error fetching pools:', error);
      return [];
    }
  }

  /**
   * Get pool for token pair
   */
  async getPoolByTokens(tokenA: string, tokenB: string): Promise<Pool | null> {
    const pools = await this.getAllPools();
    return pools.find(pool =>
      (pool.tokenA === tokenA && pool.tokenB === tokenB) ||
      (pool.tokenA === tokenB && pool.tokenB === tokenA)
    ) || null;
  }

  /**
   * Calculate swap output amount
   */
  calculateSwapOutput(
    amountIn: string,
    reserveIn: string,
    reserveOut: string,
    swapFee: string = '0.003'
  ): string {
    const amountInBig = BigInt(amountIn);
    const reserveInBig = BigInt(reserveIn);
    const reserveOutBig = BigInt(reserveOut);
    const feeBig = BigInt(Math.floor(parseFloat(swapFee) * 10000));

    // Apply fee: amountInWithFee = amountIn * (10000 - fee) / 10000
    const amountInWithFee = (amountInBig * (10000n - feeBig)) / 10000n;

    // Constant product formula: amountOut = (amountInWithFee * reserveOut) / (reserveIn + amountInWithFee)
    const numerator = amountInWithFee * reserveOutBig;
    const denominator = reserveInBig + amountInWithFee;

    return (numerator / denominator).toString();
  }

  /**
   * Calculate price impact
   */
  calculatePriceImpact(
    amountIn: string,
    reserveIn: string,
    reserveOut: string
  ): number {
    const amountOut = this.calculateSwapOutput(amountIn, reserveIn, reserveOut, '0');
    const priceBeforeSwap = parseFloat(reserveOut) / parseFloat(reserveIn);
    const priceAfterSwap = parseFloat(amountOut) / parseFloat(amountIn);

    return Math.abs((priceAfterSwap - priceBeforeSwap) / priceBeforeSwap) * 100;
  }

  /**
   * Calculate shares for liquidity addition
   */
  calculateShares(
    amountA: string,
    amountB: string,
    reserveA: string,
    reserveB: string,
    totalShares: string
  ): string {
    if (totalShares === '0') {
      // First liquidity provider
      const amountABig = BigInt(amountA);
      const amountBBig = BigInt(amountB);
      return (amountABig * amountBBig).toString();
    }

    const amountABig = BigInt(amountA);
    const reserveABig = BigInt(reserveA);
    const totalSharesBig = BigInt(totalShares);

    return ((amountABig * totalSharesBig) / reserveABig).toString();
  }
}
