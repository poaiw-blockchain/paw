import { PawClient } from '../client';
import { SendParams, TxResult, GasOptions } from '../types';
import { Coin } from '@cosmjs/stargate';

export class BankModule {
  constructor(private client: PawClient) {}

  /**
   * Get account balance for a specific denom
   */
  async getBalance(address: string, denom: string): Promise<Coin | null> {
    const client = this.client.getClient();
    return await client.getBalance(address, denom);
  }

  /**
   * Get all account balances
   */
  async getAllBalances(address: string): Promise<readonly Coin[]> {
    const client = this.client.getClient();
    return await client.getAllBalances(address);
  }

  /**
   * Send tokens to another address
   */
  async send(
    senderAddress: string,
    params: SendParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const denom = params.denom || 'upaw';
    const message = {
      typeUrl: '/cosmos.bank.v1beta1.MsgSend',
      value: {
        fromAddress: senderAddress,
        toAddress: params.recipient,
        amount: [{ denom, amount: params.amount }]
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(senderAddress, [message], {
      ...options,
      memo: params.memo
    });
  }

  /**
   * Multi-send tokens to multiple recipients
   */
  async multiSend(
    senderAddress: string,
    recipients: Array<{ address: string; amount: string; denom?: string }>,
    options?: GasOptions
  ): Promise<TxResult> {
    const outputs = recipients.map(recipient => ({
      address: recipient.address,
      coins: [{ denom: recipient.denom || 'upaw', amount: recipient.amount }]
    }));

    const message = {
      typeUrl: '/cosmos.bank.v1beta1.MsgMultiSend',
      value: {
        inputs: [{
          address: senderAddress,
          coins: outputs.flatMap(o => o.coins)
        }],
        outputs
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(senderAddress, [message], options);
  }

  /**
   * Get total supply of a denom
   */
  async getTotalSupply(denom: string): Promise<Coin | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/bank/v1beta1/supply/by_denom?denom=${denom}`);
      if (!response.ok) {
        return null;
      }

      const data = await response.json() as { amount?: Coin };
      return data.amount || null;
    } catch (error) {
      console.error('Error fetching total supply:', error);
      return null;
    }
  }

  /**
   * Get all denoms
   */
  async getAllDenoms(): Promise<string[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/bank/v1beta1/supply`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json() as { supply?: Coin[] };
      return (data.supply || []).map((coin: Coin) => coin.denom);
    } catch (error) {
      console.error('Error fetching all denoms:', error);
      return [];
    }
  }

  /**
   * Get spendable balance (available for spending)
   */
  async getSpendableBalance(address: string, denom: string): Promise<Coin | null> {
    // For now, spendable balance is the same as regular balance
    // In the future, this might account for locked/vesting tokens
    return await this.getBalance(address, denom);
  }

  /**
   * Format balance for display
   */
  formatBalance(balance: Coin, decimals: number = 6): string {
    const amount = parseInt(balance.amount) / Math.pow(10, decimals);
    return `${amount.toFixed(decimals)} ${balance.denom.replace('u', '').toUpperCase()}`;
  }
}
