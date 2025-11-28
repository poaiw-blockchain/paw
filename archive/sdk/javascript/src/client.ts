import { SigningStargateClient, StargateClient } from '@cosmjs/stargate';
import { Tendermint34Client } from '@cosmjs/tendermint-rpc';
import { PawChainConfig } from './types';
import { PawWallet } from './wallet';
import { TxBuilder } from './tx';
import { BankModule } from './modules/bank';
import { DexModule } from './modules/dex';
import { StakingModule } from './modules/staking';
import { GovernanceModule } from './modules/governance';

export class PawClient {
  private config: PawChainConfig;
  private client: StargateClient | null = null;
  private signingClient: SigningStargateClient | null = null;
  private txBuilder: TxBuilder | null = null;

  public readonly bank: BankModule;
  public readonly dex: DexModule;
  public readonly staking: StakingModule;
  public readonly governance: GovernanceModule;

  constructor(config: PawChainConfig) {
    this.config = {
      prefix: 'paw',
      gasPrice: '0.025upaw',
      gasAdjustment: 1.5,
      ...config
    };

    this.bank = new BankModule(this);
    this.dex = new DexModule(this);
    this.staking = new StakingModule(this);
    this.governance = new GovernanceModule(this);
  }

  /**
   * Connect to the blockchain without signing capabilities
   */
  async connect(): Promise<void> {
    const tmClient = await Tendermint34Client.connect(this.config.rpcEndpoint);
    this.client = await StargateClient.create(tmClient);
  }

  /**
   * Connect with a wallet for signing transactions
   */
  async connectWithWallet(wallet: PawWallet): Promise<void> {
    const signer = wallet.getSigner();
    this.signingClient = await SigningStargateClient.connectWithSigner(
      this.config.rpcEndpoint,
      signer,
      {
        prefix: this.config.prefix,
        gasPrice: this.config.gasPrice
      }
    );

    this.txBuilder = new TxBuilder(
      this.signingClient,
      this.config.gasPrice,
      this.config.gasAdjustment
    );
  }

  /**
   * Get the read-only client
   */
  getClient(): StargateClient {
    if (!this.client) {
      throw new Error('Client not connected. Call connect() first');
    }
    return this.client;
  }

  /**
   * Get the signing client
   */
  getSigningClient(): SigningStargateClient {
    if (!this.signingClient) {
      throw new Error('Signing client not connected. Call connectWithWallet() first');
    }
    return this.signingClient;
  }

  /**
   * Get the transaction builder
   */
  getTxBuilder(): TxBuilder {
    if (!this.txBuilder) {
      throw new Error('Transaction builder not available. Call connectWithWallet() first');
    }
    return this.txBuilder;
  }

  /**
   * Get chain configuration
   */
  getConfig(): PawChainConfig {
    return this.config;
  }

  /**
   * Get current block height
   */
  async getHeight(): Promise<number> {
    return await this.getClient().getHeight();
  }

  /**
   * Get chain ID
   */
  async getChainId(): Promise<string> {
    return await this.getClient().getChainId();
  }

  /**
   * Disconnect from the blockchain
   */
  async disconnect(): Promise<void> {
    if (this.client) {
      this.client.disconnect();
      this.client = null;
    }
    if (this.signingClient) {
      this.signingClient.disconnect();
      this.signingClient = null;
    }
    this.txBuilder = null;
  }

  /**
   * Check if client is connected
   */
  isConnected(): boolean {
    return this.client !== null || this.signingClient !== null;
  }

  /**
   * Check if client has signing capabilities
   */
  canSign(): boolean {
    return this.signingClient !== null;
  }
}
