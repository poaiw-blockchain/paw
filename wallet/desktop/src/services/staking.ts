import Decimal from 'decimal.js';
import {
  Balance,
  BroadcastResult,
  PAWRPCClient,
  PAWWallet,
  RPCConfig,
} from '@paw-chain/wallet-core';
import { SigningStargateClient, GasPrice } from '@cosmjs/stargate';
import { coins } from '@cosmjs/amino';
import { ApiService } from './api';
import { KeystoreService } from './keystore';

const DEFAULT_STAKE_DENOM = 'upaw';
const DISPLAY_DECIMALS = 6;
const BASELINE_APR = 0.18;

interface DelegationResponse {
  delegation?: {
    delegator_address?: string;
    validator_address?: string;
    shares?: string;
  };
  balance?: Balance;
}

interface RewardResponse {
  validator_address?: string;
  reward?: Balance[];
}

export interface ValidatorMetrics {
  validatorAddress: string;
  moniker: string;
  website?: string;
  commissionRate: number;
  commissionFormatted: string;
  votingPowerPercent: number;
  status: string;
  jailed: boolean;
  aprEstimate: number;
  myDelegationBase: string;
  myDelegationDisplay: string;
}

export interface DelegationPosition {
  validatorAddress: string;
  validatorMoniker: string;
  amountBase: string;
  amountDisplay: string;
  rewardsBase: string;
  rewardsDisplay: string;
  status: string;
  denom: string;
}

export interface StakingPortfolio {
  summary: {
    totalDelegatedDisplay: string;
    totalRewardsDisplay: string;
    denom: string;
    symbol: string;
    activeValidators: number;
    averageApr: number;
  };
  validators: ValidatorMetrics[];
  delegations: DelegationPosition[];
  rewards: DelegationPosition[];
  updatedAt: number;
}

export interface DelegateParams {
  validatorAddress: string;
  amount: string;
  password?: string;
  offlineSigner?: any;
  fromAddress?: string;
  memo?: string;
}

export interface UndelegateParams {
  validatorAddress: string;
  amount: string;
  password?: string;
  offlineSigner?: any;
  fromAddress?: string;
  memo?: string;
}

export interface RedelegateParams {
  srcValidatorAddress: string;
  dstValidatorAddress: string;
  amount: string;
  password?: string;
  offlineSigner?: any;
  fromAddress?: string;
  memo?: string;
}

export interface WithdrawParams {
  validatorAddress: string;
  password?: string;
  offlineSigner?: any;
  fromAddress?: string;
  memo?: string;
}

interface UnlockedWallet {
  address: string;
  publicKey: string;
  privateKey: string;
}

export class StakingService {
  private apiService = new ApiService();
  private keystore = new KeystoreService();
  private rpcClientPromise?: Promise<PAWRPCClient>;
  private stakeDenom = DEFAULT_STAKE_DENOM;
  private stakeSymbol = 'PAW';

  async getPortfolio(address: string): Promise<StakingPortfolio> {
    if (!address) {
      throw new Error('Wallet address required to load staking data');
    }

    const client = await this.getRpcClient();
    const [validatorsRaw, delegationsRaw, rewardsRaw] = await Promise.all([
      client.getValidators(),
      client.getDelegations(address),
      client.getRewards(address),
    ]);

    const totalNetworkTokens = validatorsRaw.reduce(
      (acc: Decimal, validator: any) => acc.plus(new Decimal(this.extractString(validator.tokens))),
      new Decimal(0)
    );

    const rewardMap = this.normalizeRewards(rewardsRaw.rewards || []);
    const delegations = this.normalizeDelegations(delegationsRaw as DelegationResponse[], rewardMap);

    const totalDelegatedBase = delegations.reduce(
      (acc, delegation) => acc.plus(new Decimal(delegation.amountBase || '0')),
      new Decimal(0)
    );
    const totalRewardsBase = delegations.reduce(
      (acc, delegation) => acc.plus(new Decimal(delegation.rewardsBase || '0')),
      new Decimal(0)
    );

    const detectedDenom = delegations[0]?.denom || rewardsRaw.total?.[0]?.denom || this.stakeDenom;
    this.updateStakeDenom(detectedDenom);

    const validatorLookup = new Map<string, any>();
    validatorsRaw.forEach((validator: any) => {
      validatorLookup.set(this.extractString(validator.operator_address || validator.operatorAddress), validator);
    });

    const validators = validatorsRaw.map((validator: any) => {
      const operatorAddress = this.extractString(validator.operator_address || validator.operatorAddress);
      const commissionRate = this.extractNumber(
        validator.commission?.commissionRates?.rate ||
          validator.commission?.commission_rates?.rate ||
          '0'
      );
      const status = this.extractString(validator.status) || 'unknown';
      const votingPowerPercent = this.calculateVotingPowerPercent(
        this.extractString(validator.tokens),
        totalNetworkTokens
      );
      const userDelegation = delegations.find((d) => d.validatorAddress === operatorAddress);
      const myDelegationBase = userDelegation?.amountBase || '0';

      return {
        validatorAddress: operatorAddress,
        moniker: validator.description?.moniker || 'Unknown',
        website: validator.description?.website,
        commissionRate,
        commissionFormatted: `${(commissionRate * 100).toFixed(2)}%`,
        votingPowerPercent,
        status,
        jailed: Boolean(validator.jailed),
        aprEstimate: this.estimateApr(commissionRate, status),
        myDelegationBase,
        myDelegationDisplay: this.formatAmount(myDelegationBase),
      };
    });

    const summary = {
      totalDelegatedDisplay: this.formatAmount(totalDelegatedBase.toString()),
      totalRewardsDisplay: this.formatAmount(totalRewardsBase.toString()),
      denom: this.stakeDenom,
      symbol: this.stakeSymbol,
      activeValidators: validators.filter((v) => !v.jailed && this.isBonded(v.status)).length,
      averageApr: validators.length
        ? validators.reduce((acc, v) => acc + v.aprEstimate, 0) / validators.length
        : 0,
    };

    const mappedDelegations = delegations.map((delegation) => {
      const validator = validatorLookup.get(delegation.validatorAddress);
      return {
        ...delegation,
        validatorMoniker: validator?.description?.moniker || delegation.validatorMoniker,
        status: validator?.status || delegation.status,
      };
    });

    return {
      summary,
      validators,
      delegations: mappedDelegations,
      rewards: mappedDelegations,
      updatedAt: Date.now(),
    };
  }

  async delegate(params: DelegateParams): Promise<BroadcastResult> {
    const amountBase = this.toBaseAmount(params.amount);
    if (params.offlineSigner) {
      const { client, fromAddress } = await this.getSigningClient(params.offlineSigner, params.fromAddress);
      const res = await client.delegateTokens(
        fromAddress,
        params.validatorAddress,
        coins(amountBase, this.stakeDenom),
        undefined,
        params.memo
      );
      await client.disconnect?.();
      return res as unknown as BroadcastResult;
    }

    const wallet = await this.buildWallet(params.password || '');
    return wallet.delegate(
      params.validatorAddress,
      amountBase,
      this.stakeDenom,
      params.memo ? { memo: params.memo } : undefined
    );
  }

  async undelegate(params: UndelegateParams): Promise<BroadcastResult> {
    const amountBase = this.toBaseAmount(params.amount);
    if (params.offlineSigner) {
      const { client, fromAddress } = await this.getSigningClient(params.offlineSigner, params.fromAddress);
      const res = await client.undelegateTokens(
        fromAddress,
        params.validatorAddress,
        coins(amountBase, this.stakeDenom),
        undefined,
        params.memo
      );
      await client.disconnect?.();
      return res as unknown as BroadcastResult;
    }

    const wallet = await this.buildWallet(params.password || '');
    return wallet.undelegate(
      params.validatorAddress,
      amountBase,
      this.stakeDenom,
      params.memo ? { memo: params.memo } : undefined
    );
  }

  async redelegate(params: RedelegateParams): Promise<BroadcastResult> {
    const amountBase = this.toBaseAmount(params.amount);
    if (params.offlineSigner) {
      const { client, fromAddress } = await this.getSigningClient(params.offlineSigner, params.fromAddress);
      const res = await client.redelegateTokens(
        fromAddress,
        params.srcValidatorAddress,
        params.dstValidatorAddress,
        coins(amountBase, this.stakeDenom),
        undefined,
        params.memo
      );
      await client.disconnect?.();
      return res as unknown as BroadcastResult;
    }

    const wallet = await this.buildWallet(params.password || '');
    return wallet.redelegate(
      params.srcValidatorAddress,
      params.dstValidatorAddress,
      amountBase,
      this.stakeDenom,
      params.memo ? { memo: params.memo } : undefined
    );
  }

  async withdraw(params: WithdrawParams): Promise<BroadcastResult> {
    if (params.offlineSigner) {
      const { client, fromAddress } = await this.getSigningClient(params.offlineSigner, params.fromAddress);
      const res = await client.withdrawRewards(
        fromAddress,
        params.validatorAddress,
        undefined,
        params.memo
      );
      await client.disconnect?.();
      return res as unknown as BroadcastResult;
    }

    const wallet = await this.buildWallet(params.password || '');
    return wallet.withdrawRewards(
      params.validatorAddress,
      params.memo ? { memo: params.memo } : undefined
    );
  }

  private async buildWallet(password: string): Promise<PAWWallet> {
    if (!password || password.length < 8) {
      throw new Error('Wallet password is required to sign staking transactions');
    }

    const rpcConfig = await this.buildRpcConfig();
    const wallet = new PAWWallet({ rpcConfig });
    const unlocked = (await this.keystore.unlockWallet(password)) as UnlockedWallet;
    await wallet.createFromMnemonic(unlocked.privateKey);
    return wallet;
  }

  private normalizeDelegations(
    delegations: DelegationResponse[],
    rewardMap: Map<string, Decimal>
  ): DelegationPosition[] {
    return delegations.map((entry) => {
      const validatorAddress = this.extractString(
        entry.delegation?.validator_address || entry.delegation?.validatorAddress
      );
      const amountBase = entry.balance?.amount || '0';
      const rewardsBase = rewardMap.get(validatorAddress) || new Decimal(0);
      const status = 'active';
      const denom = entry.balance?.denom || this.stakeDenom;

      return {
        validatorAddress,
        validatorMoniker: validatorAddress,
        amountBase,
        amountDisplay: this.formatAmount(amountBase),
        rewardsBase: rewardsBase.toString(),
        rewardsDisplay: this.formatAmount(rewardsBase.toString()),
        status,
        denom,
      };
    });
  }

  private normalizeRewards(rewards: RewardResponse[]): Map<string, Decimal> {
    const map = new Map<string, Decimal>();

    rewards.forEach((reward) => {
      const validatorAddress = this.extractString(reward.validator_address);
      const baseAmount = reward.reward?.[0]?.amount || '0';
      map.set(validatorAddress, new Decimal(baseAmount));
    });

    return map;
  }

  private estimateApr(commissionRate: number, status: string): number {
    const availabilityFactor = this.isBonded(status) ? 1 : 0.4;
    return Number((BASELINE_APR * (1 - commissionRate) * availabilityFactor * 100).toFixed(2));
  }

  private calculateVotingPowerPercent(tokens: string, total: Decimal): number {
    if (total.isZero()) {
      return 0;
    }
    try {
      return new Decimal(tokens || '0').div(total).times(100).toNumber();
    } catch {
      return 0;
    }
  }

  private extractString(value: any): string {
    if (value === null || value === undefined) {
      return '';
    }
    return String(value);
  }

  private extractNumber(value: any): number {
    const parsed = parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  private formatAmount(baseAmount: string): string {
    try {
      const value = new Decimal(baseAmount || '0').div(new Decimal(10).pow(DISPLAY_DECIMALS));
      if (value.eq(0)) {
        return `0 ${this.stakeSymbol}`;
      }
      if (value.gte(1)) {
        return `${value.toFixed(4).replace(/\.?0+$/, '')} ${this.stakeSymbol}`;
      }
      return `${value.toSignificantDigits(6).toString()} ${this.stakeSymbol}`;
    } catch {
      return `0 ${this.stakeSymbol}`;
    }
  }

  private toBaseAmount(displayAmount: string): string {
    const value = new Decimal(displayAmount || '0');
    const multiplier = new Decimal(10).pow(DISPLAY_DECIMALS);
    return value.times(multiplier).floor().toString();
  }

  private updateStakeDenom(denom: string) {
    if (!denom) {
      return;
    }
    this.stakeDenom = denom;
    this.stakeSymbol = denom.startsWith('u') ? denom.slice(1).toUpperCase() : denom.toUpperCase();
  }

  private isBonded(status: string): boolean {
    return status === 'BOND_STATUS_BONDED' || status === 'bonded';
  }

  private async buildRpcConfig(): Promise<RPCConfig> {
    const restUrl = await this.apiService.getEndpoint();
    const rpcUrl = this.deriveRpcUrl(restUrl);
    return { restUrl, rpcUrl };
  }

  private async getSigningClient(offlineSigner: any, fromAddress?: string) {
    if (!offlineSigner) {
      throw new Error('Offline signer is required for hardware wallet operations');
    }
    const restUrl = await this.apiService.getEndpoint();
    const rpcUrl = this.deriveRpcUrl(restUrl);
    const client = await SigningStargateClient.connectWithSigner(rpcUrl, offlineSigner, {
      gasPrice: GasPrice.fromString('0.025upaw'),
    });
    let address = fromAddress;
    if (!address) {
      const accounts = await offlineSigner.getAccounts();
      address = accounts?.[0]?.address;
    }
    if (!address) {
      throw new Error('Failed to determine sender address for staking transaction');
    }
    return { client, fromAddress: address };
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
}
