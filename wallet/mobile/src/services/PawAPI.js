import axios from 'axios';

const DEFAULT_BASE_URL = 'http://localhost:1317';

class PawAPIService {
  constructor() {
    this.baseURL = DEFAULT_BASE_URL;
    this.client = axios.create({
      baseURL: this.baseURL,
      timeout: 10000,
    });
  }

  setBaseURL(url) {
    this.baseURL = url;
    this.client = axios.create({
      baseURL: url,
      timeout: 10000,
    });
  }

  formatError(error) {
    if (error?.response?.data?.message) {
      throw new Error(error.response.data.message || 'API Error');
    }
    if (error?.request) {
      throw new Error('Network error');
    }
    throw new Error(error?.message || 'Request error');
  }

  async request(method, path, config = {}) {
    try {
      const response = await this.client[method](path, config);
      return response.data;
    } catch (error) {
      this.formatError(error);
    }
  }

  async getNodeInfo() {
    return this.request('get', '/cosmos/base/tendermint/v1beta1/node_info');
  }

  async getBalance(address) {
    return this.request('get', `/cosmos/bank/v1beta1/balances/${address}`);
  }

  async getAccount(address) {
    const data = await this.request(
      'get',
      `/cosmos/auth/v1beta1/accounts/${address}`,
    );
    return data?.account;
  }

  async getTransaction(hash) {
    return this.request('get', `/cosmos/tx/v1beta1/txs/${hash}`);
  }

  async getTransactionsByAddress(address, limit = 20, page = 1) {
    const data = await this.request('get', '/cosmos/tx/v1beta1/txs', {
      params: {
        'pagination.limit': limit,
        'pagination.offset': (page - 1) * limit,
        events: `message.sender='${address}'`,
      },
    });
    return data?.txs || [];
  }

  async getDexPools() {
    const data = await this.request('get', '/paw/dex/v1/pools');
    return data?.pools || [];
  }

  async getPool(poolId) {
    const data = await this.request('get', `/paw/dex/v1/pools/${poolId}`);
    return data?.pool;
  }

  async getOraclePrices() {
    const data = await this.request('get', '/paw/oracle/v1/prices');
    return data?.prices || [];
  }

  async getValidators() {
    const data = await this.request(
      'get',
      '/cosmos/staking/v1beta1/validators',
    );
    return data?.validators || [];
  }

  async getDelegations(address) {
    const data = await this.request(
      'get',
      `/cosmos/staking/v1beta1/delegations/${address}`,
    );
    return data?.delegation_responses || [];
  }

  async getRewards(address) {
    return this.request(
      'get',
      `/cosmos/distribution/v1beta1/delegators/${address}/rewards`,
    );
  }

  /**
   * Broadcast a signed transaction to the chain
   * @param {string} txBytes - Base64 encoded signed transaction bytes
   * @param {string} mode - Broadcast mode: BROADCAST_MODE_SYNC, BROADCAST_MODE_ASYNC, BROADCAST_MODE_BLOCK
   * @returns {Promise<Object>} Transaction response with hash
   */
  async broadcastTransaction(txBytes, mode = 'BROADCAST_MODE_SYNC') {
    try {
      const response = await this.client.post('/cosmos/tx/v1beta1/txs', {
        tx_bytes: txBytes,
        mode: mode,
      });
      const txResponse = response.data?.tx_response;
      if (txResponse?.code !== 0 && txResponse?.code !== undefined) {
        throw new Error(txResponse.raw_log || `Transaction failed with code ${txResponse.code}`);
      }
      return txResponse;
    } catch (error) {
      if (error?.response?.data?.message) {
        throw new Error(error.response.data.message);
      }
      throw error;
    }
  }

  /**
   * Simulate a transaction to estimate gas
   * @param {string} txBytes - Base64 encoded transaction bytes
   * @returns {Promise<Object>} Simulation result with gas estimate
   */
  async simulateTransaction(txBytes) {
    try {
      const response = await this.client.post('/cosmos/tx/v1beta1/simulate', {
        tx_bytes: txBytes,
      });
      return response.data;
    } catch (error) {
      if (error?.response?.data?.message) {
        throw new Error(error.response.data.message);
      }
      throw error;
    }
  }
}

const PawAPI = new PawAPIService();
export default PawAPI;
