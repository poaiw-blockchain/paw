import Decimal from 'decimal.js';
import {
  PAWRPCClient,
  PAWWallet,
  Pool,
  PriceData,
  RPCConfig,
  BroadcastResult,
} from '@paw-chain/wallet-core';
import { ApiService } from './api';
import { KeystoreService } from './keystore';

export interface DenomMetadata {
  denom: string;
  symbol: string;
  display: string;
  decimals: number;
}

export interface SwapQuote {
  poolId: number;
  tokenIn: DenomMetadata;
  tokenOut: DenomMetadata;
  amountIn: string;
  normalizedAmountIn: string;
  expectedAmountOut: string;
  minAmountOut: string;
  minAmountOutBase: string;
  executionPrice: string;
  inverseExecutionPrice: string;
  priceImpactPercent: number;
  updatedAt: number;
}

export interface ExecuteSwapParams {
  tokenIn: string;
  tokenOut: string;
  amountIn: string;
  slippagePercent: number;
  password: string;
  memo?: string;
}

const DEFAULT_DECIMALS = 6;

const SLIPPAGE_MIN = 0.1;
const SLIPPAGE_MAX = 5;

interface UnlockedWallet {
  address: string;
  publicKey: string;
  privateKey: string;
}

export class DexService {
  private apiService: ApiService;
  private keystore: KeystoreService;
  private rpcClientPromise?: Promise<PAWRPCClient>;
  private denomCache: Map<string, DenomMetadata>;
  private priceCache?: Map<string, PriceData>;
  private priceCacheTimestamp = 0;
  private poolCache?: Pool[];
  private lastPoolSync = 0;

  constructor() {
    this.apiService = new ApiService();
    this.keystore = new KeystoreService();
    this.denomCache = new Map();
  }

  async getPools(force = false): Promise<Pool[]> {
    const now = Date.now();
    if (!force && this.poolCache && now - this.lastPoolSync < 30_000) {
      return this.poolCache;
    }

    const client = await this.getRpcClient();
    this.poolCache = await client.getPools();
    this.lastPoolSync = now;
    return this.poolCache;
  }

  async getTradableTokens(): Promise<DenomMetadata[]> {
    const pools = await this.getPools();
    const denoms = new Set<string>();
    pools.forEach((pool) => {
      denoms.add(pool.tokenA);
      denoms.add(pool.tokenB);
    });

    const metadata: DenomMetadata[] = [];
    for (const denom of denoms) {
      metadata.push(await this.getDenomMetadata(denom));
    }

    return metadata.sort((a, b) => a.symbol.localeCompare(b.symbol));
  }

  async quoteSwap(params: {
    tokenIn: string;
    tokenOut: string;
    amountIn: string;
    slippagePercent: number;
  }): Promise<SwapQuote> {
    const sanitizedAmount = params.amountIn.trim();
    if (!sanitizedAmount || Number(sanitizedAmount) <= 0) {
      throw new Error('Enter a valid amount to quote');
    }
    const slippage = this.normalizeSlippage(params.slippagePercent);

    const pool = await this.findPool(params.tokenIn, params.tokenOut);
    const tokenInInfo = await this.getDenomMetadata(params.tokenIn);
    const tokenOutInfo = await this.getDenomMetadata(params.tokenOut);
    const amountInBase = this.toBaseAmount(sanitizedAmount, tokenInInfo.decimals);

    const client = await this.getRpcClient();
    const { amountOut } = await client.simulateSwap(
      pool.id,
      params.tokenIn,
      params.tokenOut,
      amountInBase
    );

    const amountOutDisplay = this.fromBaseAmount(amountOut, tokenOutInfo.decimals);

    const minAmountOutBase = this.applySlippage(amountOut, slippage);
    const executionPrice = this.calculateExecutionPrice(
      sanitizedAmount,
      amountOutDisplay
    );
    const inversePrice = this.calculateExecutionPrice(
      amountOutDisplay,
      sanitizedAmount
    );
    const priceImpact = this.calculatePriceImpact(pool, params.tokenIn, sanitizedAmount, amountOutDisplay);

    return {
      poolId: Number(pool.id),
      tokenIn: tokenInInfo,
      tokenOut: tokenOutInfo,
      amountIn: sanitizedAmount,
      normalizedAmountIn: amountInBase,
      expectedAmountOut: this.formatTokenAmount(amountOutDisplay),
      minAmountOut: this.fromBaseAmount(minAmountOutBase, tokenOutInfo.decimals),
      minAmountOutBase,
      executionPrice,
      inverseExecutionPrice: inversePrice,
      priceImpactPercent: priceImpact,
      updatedAt: Date.now(),
    };
  }

  async executeSwap(params: ExecuteSwapParams): Promise<BroadcastResult> {
    const quote = await this.quoteSwap({
      tokenIn: params.tokenIn,
      tokenOut: params.tokenOut,
      amountIn: params.amountIn,
      slippagePercent: params.slippagePercent,
    });

    const rpcConfig = await this.buildRpcConfig();
    const wallet = new PAWWallet({ rpcConfig });

    const unlocked = (await this.keystore.unlockWallet(params.password)) as UnlockedWallet;
    await wallet.createFromMnemonic(unlocked.privateKey);

    return wallet.swap(
      quote.poolId,
      params.tokenIn,
      params.tokenOut,
      quote.normalizedAmountIn,
      quote.minAmountOutBase,
      params.memo ? { memo: params.memo } : undefined
    );
  }

  async getOraclePrice(denom: string): Promise<PriceData | undefined> {
    const cache = await this.loadPriceCache();
    const assetKey = this.deriveAssetSymbol(denom);
    return cache.get(assetKey);
  }

  private async findPool(tokenIn: string, tokenOut: string): Promise<Pool> {
    const pools = await this.getPools();
    const pool = pools.find(
      (p) =>
        (p.tokenA === tokenIn && p.tokenB === tokenOut) ||
        (p.tokenA === tokenOut && p.tokenB === tokenIn)
    );

    if (!pool) {
      throw new Error(`No pool available for ${tokenIn} -> ${tokenOut}`);
    }

    return pool;
  }

  private calculateExecutionPrice(amountIn: string, amountOut: string): string {
    try {
      const inDec = new Decimal(amountIn);
      const outDec = new Decimal(amountOut);
      if (inDec.isZero()) {
        return '0';
      }
      return outDec.div(inDec).toSignificantDigits(6).toString();
    } catch {
      return '0';
    }
  }

  private calculatePriceImpact(pool: Pool, tokenIn: string, amountInDisplay: string, amountOutDisplay: string): number {
    try {
      const tokenOut = pool.tokenA === tokenIn ? pool.tokenB : pool.tokenA;
      const inInfo = this.denomCache.get(tokenIn);
      const outInfo = this.denomCache.get(tokenOut);
      if (!inInfo || !outInfo) {
        return 0;
      }

      const reserveIn = new Decimal(this.fromBaseAmount(pool.tokenA === tokenIn ? pool.reserveA : pool.reserveB, inInfo.decimals));
      const reserveOut = new Decimal(this.fromBaseAmount(pool.tokenA === tokenIn ? pool.reserveB : pool.reserveA, outInfo.decimals));
      if (reserveIn.isZero() || reserveOut.isZero()) {
        return 0;
      }

      const spotPrice = reserveOut.div(reserveIn);
      const executionPrice = new Decimal(amountOutDisplay).div(new Decimal(amountInDisplay));
      const impact = spotPrice.minus(executionPrice).div(spotPrice).times(100);

      if (!impact.isFinite() || impact.isNegative()) {
        return 0;
      }

      return impact.toSignificantDigits(4).toNumber();
    } catch {
      return 0;
    }
  }

  private normalizeSlippage(slippagePercent: number): number {
    if (Number.isNaN(slippagePercent)) {
      return 1;
    }
    return Math.min(Math.max(slippagePercent, SLIPPAGE_MIN), SLIPPAGE_MAX);
  }

  private applySlippage(amountOutBase: string, slippagePercent: number): string {
    const amount = new Decimal(amountOutBase);
    const factor = new Decimal(1).minus(new Decimal(slippagePercent).div(100));
    if (factor.lte(0)) {
      return '0';
    }
    return amount.times(factor).floor().toString();
  }

  private formatTokenAmount(amount: string): string {
    try {
      const value = new Decimal(amount);
      if (value.gte(1)) {
        return value.toFixed(6).replace(/\.?0+$/, '');
      }
      return value.toSignificantDigits(6).toString();
    } catch {
      return amount;
    }
  }

  private async getDenomMetadata(denom: string): Promise<DenomMetadata> {
    const cached = this.denomCache.get(denom);
    if (cached) {
      return cached;
    }

    let decimals = DEFAULT_DECIMALS;
    let symbol = denom.toUpperCase();
    let display = denom;

    try {
      const metadata = await this.apiService.getDenomMetadata(denom);
      if (metadata) {
        const displayUnit = metadata.denom_units?.find((unit: any) => unit.denom === metadata.display);
        if (displayUnit?.exponent) {
          decimals = displayUnit.exponent;
        }
        symbol = metadata.symbol || metadata.display || symbol;
        display = metadata.display || denom;
      } else if (denom.startsWith('u')) {
        decimals = 6;
        symbol = denom.slice(1).toUpperCase();
        display = symbol;
      }
    } catch {
      if (denom.startsWith('u')) {
        decimals = 6;
        symbol = denom.slice(1).toUpperCase();
        display = symbol;
      }
    }

    const normalized: DenomMetadata = {
      denom,
      symbol,
      display,
      decimals,
    };
    this.denomCache.set(denom, normalized);
    return normalized;
  }

  private toBaseAmount(amount: string, decimals: number): string {
    const value = new Decimal(amount || '0');
    const multiplier = new Decimal(10).pow(decimals);
    return value.times(multiplier).floor().toString();
  }

  private fromBaseAmount(amount: string, decimals: number): string {
    if (!amount) {
      return '0';
    }
    const value = new Decimal(amount);
    const divisor = new Decimal(10).pow(decimals);
    return value.div(divisor).toString();
  }

  private async buildRpcConfig(): Promise<RPCConfig> {
    const restUrl = await this.apiService.getEndpoint();
    const rpcUrl = this.deriveRpcUrl(restUrl);
    return {
      restUrl,
      rpcUrl,
    };
  }

  private deriveRpcUrl(restUrl: string): string {
    if (!restUrl) {
      return 'http://localhost:26657';
    }
    if (restUrl.includes('1317')) {
      return restUrl.replace('1317', '26657').replace(/\/cosmos.*$/, '');
    }
    return restUrl;
  }

  private async getRpcClient(): Promise<PAWRPCClient> {
    if (!this.rpcClientPromise) {
      this.rpcClientPromise = this.buildRpcConfig().then((config) => new PAWRPCClient(config));
    }
    return this.rpcClientPromise;
  }

  private async loadPriceCache(): Promise<Map<string, PriceData>> {
    const now = Date.now();
    if (this.priceCache && now - this.priceCacheTimestamp < 15_000) {
      return this.priceCache;
    }

    const client = await this.getRpcClient();
    const prices = await client.getAllPrices();
    const map = new Map<string, PriceData>();
    prices.forEach((price) => map.set(price.asset, price));
    this.priceCache = map;
    this.priceCacheTimestamp = now;
    return map;
  }

  private deriveAssetSymbol(denom: string): string {
    if (!denom) {
      return 'UNKNOWN/USD';
    }
    if (denom.startsWith('u')) {
      return `${denom.slice(1).toUpperCase()}/USD`;
    }
    return `${denom.toUpperCase()}/USD`;
  }
}
