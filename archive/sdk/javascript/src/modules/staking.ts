import { PawClient } from '../client';
import {
  Validator,
  DelegateParams,
  UndelegateParams,
  RedelegateParams,
  TxResult,
  GasOptions
} from '../types';
import { Coin } from '@cosmjs/stargate';

export class StakingModule {
  constructor(private client: PawClient) {}

  /**
   * Delegate tokens to a validator
   */
  async delegate(
    delegator: string,
    params: DelegateParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const denom = params.denom || 'upaw';
    const message = {
      typeUrl: '/cosmos.staking.v1beta1.MsgDelegate',
      value: {
        delegatorAddress: delegator,
        validatorAddress: params.validatorAddress,
        amount: { denom, amount: params.amount }
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(delegator, [message], options);
  }

  /**
   * Undelegate tokens from a validator
   */
  async undelegate(
    delegator: string,
    params: UndelegateParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const denom = params.denom || 'upaw';
    const message = {
      typeUrl: '/cosmos.staking.v1beta1.MsgUndelegate',
      value: {
        delegatorAddress: delegator,
        validatorAddress: params.validatorAddress,
        amount: { denom, amount: params.amount }
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(delegator, [message], options);
  }

  /**
   * Redelegate tokens from one validator to another
   */
  async redelegate(
    delegator: string,
    params: RedelegateParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const denom = params.denom || 'upaw';
    const message = {
      typeUrl: '/cosmos.staking.v1beta1.MsgBeginRedelegate',
      value: {
        delegatorAddress: delegator,
        validatorSrcAddress: params.srcValidatorAddress,
        validatorDstAddress: params.dstValidatorAddress,
        amount: { denom, amount: params.amount }
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(delegator, [message], options);
  }

  /**
   * Withdraw delegation rewards from a validator
   */
  async withdrawRewards(
    delegator: string,
    validatorAddress: string,
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward',
      value: {
        delegatorAddress: delegator,
        validatorAddress
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(delegator, [message], options);
  }

  /**
   * Withdraw all delegation rewards
   */
  async withdrawAllRewards(
    delegator: string,
    options?: GasOptions
  ): Promise<TxResult> {
    const delegations = await this.getDelegations(delegator);
    const messages = delegations.map(delegation => ({
      typeUrl: '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward',
      value: {
        delegatorAddress: delegator,
        validatorAddress: delegation.delegation.validatorAddress
      }
    }));

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(delegator, messages, options);
  }

  /**
   * Get all validators
   */
  async getValidators(): Promise<Validator[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.validators || [];
    } catch (error) {
      console.error('Error fetching validators:', error);
      return [];
    }
  }

  /**
   * Get validator by address
   */
  async getValidator(validatorAddress: string): Promise<Validator | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/staking/v1beta1/validators/${validatorAddress}`);
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.validator || null;
    } catch (error) {
      console.error('Error fetching validator:', error);
      return null;
    }
  }

  /**
   * Get delegations for a delegator
   */
  async getDelegations(delegator: string): Promise<any[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/staking/v1beta1/delegations/${delegator}`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.delegation_responses || [];
    } catch (error) {
      console.error('Error fetching delegations:', error);
      return [];
    }
  }

  /**
   * Get delegation to a specific validator
   */
  async getDelegation(delegator: string, validatorAddress: string): Promise<any | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(
        `${restEndpoint}/cosmos/staking/v1beta1/validators/${validatorAddress}/delegations/${delegator}`
      );
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.delegation_response || null;
    } catch (error) {
      console.error('Error fetching delegation:', error);
      return null;
    }
  }

  /**
   * Get unbonding delegations
   */
  async getUnbondingDelegations(delegator: string): Promise<any[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(
        `${restEndpoint}/cosmos/staking/v1beta1/delegators/${delegator}/unbonding_delegations`
      );
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.unbonding_responses || [];
    } catch (error) {
      console.error('Error fetching unbonding delegations:', error);
      return [];
    }
  }

  /**
   * Get rewards for a delegator
   */
  async getRewards(delegator: string): Promise<Coin[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(
        `${restEndpoint}/cosmos/distribution/v1beta1/delegators/${delegator}/rewards`
      );
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.total || [];
    } catch (error) {
      console.error('Error fetching rewards:', error);
      return [];
    }
  }

  /**
   * Get staking pool
   */
  async getPool(): Promise<any | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/staking/v1beta1/pool`);
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
   * Calculate APY for a validator
   */
  calculateAPY(validator: Validator, annualProvisions: string, totalBondedTokens: string): number {
    const commission = parseFloat(validator.commission.rate);
    const inflation = parseFloat(annualProvisions) / parseFloat(totalBondedTokens);
    return (inflation * (1 - commission)) * 100;
  }
}
