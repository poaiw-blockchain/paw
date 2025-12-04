import axios from 'axios';
import { SigningStargateClient, GasPrice } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';

export class ApiService {
  constructor() {
    this.apiEndpoint = this.getApiEndpoint();
  }

  getApiEndpoint() {
    if (window.electron?.store) {
      return window.electron.store.get('apiEndpoint').then(endpoint =>
        endpoint || 'http://localhost:1317'
      );
    }
    return Promise.resolve('http://localhost:1317');
  }

  async getEndpoint() {
    return await this.apiEndpoint;
  }

  /**
   * Get account balance
   */
  async getBalance(address) {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(
        `${endpoint}/cosmos/bank/v1beta1/balances/${address}`
      );
      return response.data;
    } catch (error) {
      console.error('Failed to get balance:', error);
      throw new Error(error.response?.data?.message || 'Failed to fetch balance');
    }
  }

  /**
   * Get account information (sequence, account number)
   */
  async getAccount(address) {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(
        `${endpoint}/cosmos/auth/v1beta1/accounts/${address}`
      );
      return response.data.account;
    } catch (error) {
      console.error('Failed to get account:', error);
      throw new Error(error.response?.data?.message || 'Failed to fetch account');
    }
  }

  /**
   * Get transaction history
   */
  async getTransactions(address, limit = 50) {
    try {
      const endpoint = await this.getEndpoint();

      // Try to get transactions from the tx search endpoint
      const response = await axios.get(
        `${endpoint}/cosmos/tx/v1beta1/txs`,
        {
          params: {
            events: `message.sender='${address}'`,
            'pagination.limit': limit,
            order_by: 'ORDER_BY_DESC'
          }
        }
      );

      return response.data.txs || response.data.tx_responses || [];
    } catch (error) {
      console.error('Failed to get transactions:', error);
      // Return empty array if endpoint doesn't exist or fails
      return [];
    }
  }

  /**
   * Send tokens to another address
   */
  async sendTokens(fromAddress, toAddress, amount, denom, memo, privateKey) {
    try {
      const endpoint = await this.getEndpoint();
      const rpcEndpoint = endpoint.replace('1317', '26657').replace('/cosmos', '');

      // Create wallet from private key
      const wallet = await DirectSecp256k1HdWallet.fromMnemonic(privateKey, {
        prefix: 'paw'
      });

      // Get signing client
      const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        wallet,
        {
          gasPrice: GasPrice.fromString('0.025upaw')
        }
      );

      // Send transaction
      const result = await client.sendTokens(
        fromAddress,
        toAddress,
        [{ denom, amount: amount.toString() }],
        {
          amount: [{ denom: 'upaw', amount: '5000' }],
          gas: '200000'
        },
        memo
      );

      return result;
    } catch (error) {
      console.error('Failed to send tokens:', error);
      throw new Error(error.message || 'Failed to send transaction');
    }
  }

  /**
   * Get validator list
   */
  async getValidators() {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(
        `${endpoint}/cosmos/staking/v1beta1/validators`
      );
      return response.data.validators || [];
    } catch (error) {
      console.error('Failed to get validators:', error);
      throw new Error(error.response?.data?.message || 'Failed to fetch validators');
    }
  }

  /**
   * Delegate tokens to a validator
   */
  async delegate(delegatorAddress, validatorAddress, amount, denom, memo, privateKey) {
    try {
      const endpoint = await this.getEndpoint();
      const rpcEndpoint = endpoint.replace('1317', '26657').replace('/cosmos', '');

      const wallet = await DirectSecp256k1HdWallet.fromMnemonic(privateKey, {
        prefix: 'paw'
      });

      const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        wallet,
        {
          gasPrice: GasPrice.fromString('0.025upaw')
        }
      );

      const result = await client.delegateTokens(
        delegatorAddress,
        validatorAddress,
        { denom, amount: amount.toString() },
        {
          amount: [{ denom: 'upaw', amount: '5000' }],
          gas: '200000'
        },
        memo
      );

      return result;
    } catch (error) {
      console.error('Failed to delegate:', error);
      throw new Error(error.message || 'Failed to delegate tokens');
    }
  }

  /**
   * Get node information
   */
  async getNodeInfo() {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(
        `${endpoint}/cosmos/base/tendermint/v1beta1/node_info`
      );
      return response.data.node_info;
    } catch (error) {
      console.error('Failed to get node info:', error);
      throw new Error(error.response?.data?.message || 'Failed to fetch node info');
    }
  }

  /**
   * Get latest block
   */
  async getLatestBlock() {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(
        `${endpoint}/cosmos/base/tendermint/v1beta1/blocks/latest`
      );
      return response.data.block;
    } catch (error) {
      console.error('Failed to get latest block:', error);
      throw new Error(error.response?.data?.message || 'Failed to fetch latest block');
    }
  }

  /**
   * Get DEX pools (PAW-specific)
   */
  async getDexPools() {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(`${endpoint}/paw/dex/v1/pools`);
      return response.data.pools || [];
    } catch (error) {
      console.error('Failed to get DEX pools:', error);
      // Return empty array if DEX module not available
      return [];
    }
  }

  /**
   * Get oracle prices (PAW-specific)
   */
  async getOraclePrices() {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(`${endpoint}/paw/oracle/v1/prices`);
      return response.data.prices || [];
    } catch (error) {
      console.error('Failed to get oracle prices:', error);
      // Return empty array if oracle module not available
      return [];
    }
  }

  /**
   * Fetch bank denom metadata to determine decimals/symbols
   */
  async getDenomMetadata(denom) {
    try {
      const endpoint = await this.getEndpoint();
      const response = await axios.get(`${endpoint}/cosmos/bank/v1beta1/denoms_metadata/${denom}`);
      return response.data.metadata || null;
    } catch (error) {
      console.warn('Failed to fetch denom metadata:', denom, error?.response?.status || error.message);
      return null;
    }
  }
}
