import { PawClient } from '../client';
import { SendParams, QueryBalance, TxResult, GasOptions } from '../types';
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
    const client = this.client.getClient();

    // Use query client if available
    const queryClient = client.forceGetQueryClient();
    if (queryClient) {
      const response = await queryClient.bank.supplyOf(denom);
      return response.amount || null;
    }

    return null;
  }

  /**
   * Get all denoms
   */
  async getAllDenoms(): Promise<string[]> {
    const client = this.client.getClient();
    const queryClient = client.forceGetQueryClient();

    if (queryClient) {
      const response = await queryClient.bank.totalSupply();
      return response.supply.map((coin: Coin) => coin.denom);
    }

    return [];
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
