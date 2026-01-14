import { Coin } from '@cosmjs/stargate';

/**
 * PAW Testnet configuration
 * Chain ID: paw-mvp-1
 */
export const PAW_TESTNET_CONFIG = {
  chainId: 'paw-mvp-1',
  rpcEndpoint: 'https://testnet-rpc.poaiw.org',
  restEndpoint: 'https://testnet-api.poaiw.org',
  prefix: 'paw',
  gasPrice: '0.025upaw',
  gasAdjustment: 1.5,
  denom: 'upaw',
  displayDenom: 'PAW',
  decimals: 6
} as const;

export interface PawChainConfig {
  rpcEndpoint: string;
  restEndpoint?: string;
  chainId: string;
  prefix?: string;
  gasPrice?: string;
  gasAdjustment?: number;
  denom?: string;
}

export interface WalletAccount {
  address: string;
  pubkey: Uint8Array;
  algo: string;
}

/**
 * DEX Pool - NOTE: DEX module is disabled in current testnet (paw-mvp-1)
 * Enable via governance proposal when ready
 */
export interface Pool {
  id: string;
  tokenA: string;
  tokenB: string;
  reserveA: string;
  reserveB: string;
  totalShares: string;
  swapFee: string;
}

/**
 * @deprecated DEX module disabled in testnet - will return "Not Implemented"
 */
export interface PoolParams {
  tokenA: string;
  tokenB: string;
  amountA: string;
  amountB: string;
}

/**
 * @deprecated DEX module disabled in testnet - will return "Not Implemented"
 */
export interface SwapParams {
  poolId: string;
  tokenIn: string;
  amountIn: string;
  minAmountOut: string;
  recipient?: string;
}

/**
 * @deprecated DEX module disabled in testnet - will return "Not Implemented"
 */
export interface AddLiquidityParams {
  poolId: string;
  amountA: string;
  amountB: string;
  minShares: string;
}

/**
 * @deprecated DEX module disabled in testnet - will return "Not Implemented"
 */
export interface RemoveLiquidityParams {
  poolId: string;
  shares: string;
  minAmountA: string;
  minAmountB: string;
}

/**
 * Custom error for disabled modules
 */
export class ModuleDisabledError extends Error {
  constructor(moduleName: string) {
    super(`${moduleName} module is disabled in paw-mvp-1 testnet. Enable via governance proposal.`);
    this.name = 'ModuleDisabledError';
  }
}

/**
 * Validator as returned by the PAW testnet API
 * Uses snake_case to match actual API responses
 */
export interface Validator {
  operator_address: string;
  consensus_pubkey: {
    '@type': string;
    key: string;
  };
  jailed: boolean;
  status: 'BOND_STATUS_UNSPECIFIED' | 'BOND_STATUS_UNBONDED' | 'BOND_STATUS_UNBONDING' | 'BOND_STATUS_BONDED';
  tokens: string;
  delegator_shares: string;
  description: {
    moniker: string;
    identity: string;
    website: string;
    security_contact: string;
    details: string;
  };
  unbonding_height: string;
  unbonding_time: string;
  commission: {
    commission_rates: {
      rate: string;
      max_rate: string;
      max_change_rate: string;
    };
    update_time: string;
  };
  min_self_delegation: string;
  unbonding_on_hold_ref_count: string;
  unbonding_ids: string[];
}

/**
 * Simplified validator for display purposes (camelCase)
 */
export interface ValidatorDisplay {
  operatorAddress: string;
  moniker: string;
  jailed: boolean;
  status: string;
  tokens: string;
  commissionRate: string;
}

export interface DelegateParams {
  validatorAddress: string;
  amount: string;
  denom?: string;
}

export interface UndelegateParams {
  validatorAddress: string;
  amount: string;
  denom?: string;
}

export interface RedelegateParams {
  srcValidatorAddress: string;
  dstValidatorAddress: string;
  amount: string;
  denom?: string;
}

export interface Proposal {
  proposalId: string;
  content: {
    typeUrl: string;
    value: Uint8Array;
  };
  status: number;
  finalTallyResult: {
    yes: string;
    abstain: string;
    no: string;
    noWithVeto: string;
  };
  submitTime: Date;
  depositEndTime: Date;
  totalDeposit: Coin[];
  votingStartTime: Date;
  votingEndTime: Date;
}

export interface VoteParams {
  proposalId: string;
  option: VoteOption;
  metadata?: string;
}

export interface DepositParams {
  proposalId: string;
  amount: string;
  denom?: string;
}

export enum VoteOption {
  UNSPECIFIED = 0,
  YES = 1,
  ABSTAIN = 2,
  NO = 3,
  NO_WITH_VETO = 4,
}

export interface TxResult {
  transactionHash: string;
  height: number;
  code: number;
  rawLog?: string;
  gasUsed: number;
  gasWanted: number;
}

export interface QueryBalance {
  denom: string;
  amount: string;
}

export interface SendParams {
  recipient: string;
  amount: string;
  denom?: string;
  memo?: string;
}

export interface GasOptions {
  gasLimit?: number;
  gasPrice?: string;
  memo?: string;
}
