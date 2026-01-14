import { SigningStargateClient, StdFee, DeliverTxResponse } from '@cosmjs/stargate';
import { EncodeObject } from '@cosmjs/proto-signing';
import { GasOptions, TxResult } from './types';

export class TxBuilder {
  private client: SigningStargateClient;
  private defaultGasPrice: string;
  private defaultGasAdjustment: number;

  constructor(
    client: SigningStargateClient,
    gasPrice: string = '0.025upaw',
    gasAdjustment: number = 1.5
  ) {
    this.client = client;
    this.defaultGasPrice = gasPrice;
    this.defaultGasAdjustment = gasAdjustment;
  }

  /**
   * Sign and broadcast a transaction
   */
  async signAndBroadcast(
    signerAddress: string,
    messages: readonly EncodeObject[],
    options?: GasOptions
  ): Promise<TxResult> {
    const fee = await this.calculateFee(signerAddress, messages, options);
    const memo = options?.memo || '';

    const result = await this.client.signAndBroadcast(
      signerAddress,
      messages,
      fee,
      memo
    );

    return this.formatResult(result);
  }

  /**
   * Calculate transaction fee
   */
  async calculateFee(
    signerAddress: string,
    messages: readonly EncodeObject[],
    options?: GasOptions
  ): Promise<StdFee> {
    const gasPrice = options?.gasPrice || this.defaultGasPrice;

    if (options?.gasLimit) {
      return {
        amount: [{ denom: this.extractDenom(gasPrice), amount: this.calculateAmount(options.gasLimit, gasPrice) }],
        gas: options.gasLimit.toString()
      };
    }

    // Simulate to estimate gas
    const gasEstimate = await this.client.simulate(signerAddress, messages, '');
    const gasLimit = Math.round(gasEstimate * this.defaultGasAdjustment);

    return {
      amount: [{ denom: this.extractDenom(gasPrice), amount: this.calculateAmount(gasLimit, gasPrice) }],
      gas: gasLimit.toString()
    };
  }

  /**
   * Simulate transaction
   */
  async simulate(
    signerAddress: string,
    messages: readonly EncodeObject[]
  ): Promise<number> {
    return await this.client.simulate(signerAddress, messages, '');
  }

  /**
   * Extract denom from gas price string
   */
  private extractDenom(gasPrice: string): string {
    const match = gasPrice.match(/[a-z]+$/);
    return match ? match[0] : 'upaw';
  }

  /**
   * Calculate fee amount
   */
  private calculateAmount(gasLimit: number, gasPrice: string): string {
    const priceValue = parseFloat(gasPrice.replace(/[a-z]+$/, ''));
    return Math.ceil(gasLimit * priceValue).toString();
  }

  /**
   * Format transaction result
   */
  private formatResult(result: DeliverTxResponse): TxResult {
    return {
      transactionHash: result.transactionHash,
      height: result.height,
      code: result.code,
      rawLog: result.rawLog,
      gasUsed: Number(result.gasUsed),
      gasWanted: Number(result.gasWanted)
    };
  }
}
