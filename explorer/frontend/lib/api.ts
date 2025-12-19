import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'
const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws'
const GRAPHQL_URL = process.env.NEXT_PUBLIC_GRAPHQL_URL || 'http://localhost:8080/graphql'

// Types
export interface Block {
  height: number
  hash: string
  chain_id: string
  proposer_address: string
  timestamp: string
  tx_count: number
  gas_used: number
  gas_wanted: number
  evidence_count: number
  evidence: any
  signatures: any
  created_at: string
}

export interface Transaction {
  hash: string
  block_height: number
  tx_index: number
  type: string
  sender: string
  status: string
  code: number
  gas_used: number
  gas_wanted: number
  fee_amount: string
  fee_denom: string
  memo: string
  raw_log: string
  timestamp: string
  messages: any
  events: any
  signatures: any
  created_at: string
}

export interface ExplorerEvent {
  tx_hash: string
  block_height: number
  event_index: number
  type: string
  module: string
  attributes: any
  timestamp: string
  created_at: string
}

export interface Account {
  address: string
  first_seen_height: number
  last_seen_height: number
  tx_count: number
  total_received: string
  total_sent: string
  first_seen_at: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export interface AccountBalance {
  address: string
  denom: string
  amount: string
  last_updated_height: number
  last_updated_at: string
}

export interface AccountToken {
  address: string
  token_denom: string
  token_name: string
  token_symbol: string
  amount: string
  ibc_trace: any
  last_updated_height: number
  last_updated_at: string
}

export interface Validator {
  address: string
  consensus_address: string
  consensus_pubkey: string
  operator_address: string
  moniker: string
  identity: string
  website: string
  security_contact: string
  details: string
  voting_power: number
  commission_rate: string
  commission_max_rate: string
  commission_max_change_rate: string
  min_self_delegation: string
  jailed: boolean
  status: string
  tokens: string
  delegator_shares: string
  unbonding_height: number
  unbonding_time: string
  updated_height: number
  updated_time: string
}

export interface DEXPool {
  pool_id: string
  token_a: string
  token_b: string
  reserve_a: string
  reserve_b: string
  total_shares: string
  creator: string
  swap_fee: string
  protocol_fee: string
  volume_24h: string
  volume_7d: string
  volume_30d: string
  tvl: string
  apr: string
  block_height: number
  created_at: string
  updated_at: string
}

export interface DEXTrade {
  pool_id: string
  trader: string
  token_in: string
  token_out: string
  amount_in: string
  amount_out: string
  price: string
  fee: string
  tx_hash: string
  block_height: number
  timestamp: string
  created_at: string
}

export interface OraclePrice {
  asset: string
  price: string
  median: string
  average: string
  std_deviation: string
  num_validators: number
  num_submissions: number
  confidence_score: string
  block_height: number
  timestamp: string
  created_at: string
}

export interface OracleSubmission {
  validator_address: string
  asset: string
  price: string
  deviation?: string
  confidence?: string
  timestamp: string
  tx_hash?: string
}

export interface ComputeRequest {
  request_id: string
  requester: string
  program_hash: string
  input_data_hash: string
  reward: string
  timeout_height: number
  status: string
  provider: string
  result_hash: string
  verification_score: string
  verified: boolean
  tx_hash: string
  block_height: number
  created_at: string
  updated_at: string
}

export interface ComputeProvider {
  address: string
  stake?: string
  active?: boolean
  reputation?: number
  total_jobs?: number
  completed_jobs?: number
  failed_jobs?: number
  uptime_30d?: number
  avg_completion_time?: number
  slash_count?: number
}

export interface NetworkStats {
  totalBlocks: number
  totalTransactions: number
  activeValidators: number
  averageBlockTime: number
  tps: number
  tvl: string
  dexVolume24h: string
  activeAccounts24h: number
}

export interface PaginatedResponse<T> {
  data: T[]
  page: number
  limit: number
  total: number
  cached?: boolean
}

export interface SearchResult {
  type: 'block' | 'transaction' | 'address' | 'validator' | 'pool'
  id: string
  data: any
}

// API Client
class APIClient {
  private client: AxiosInstance

  constructor(baseURL: string) {
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add auth token if available
        const token = this.getAuthToken()
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response) {
          console.error('API Error:', error.response.status, error.response.data)
        } else if (error.request) {
          console.error('Network Error:', error.message)
        } else {
          console.error('Error:', error.message)
        }
        return Promise.reject(error)
      }
    )
  }

  private getAuthToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('auth_token')
    }
    return null
  }

  private async request<T>(config: AxiosRequestConfig): Promise<T> {
    const response: AxiosResponse<T> = await this.client.request(config)
    return response.data
  }

  // Blocks API
  async getBlocks(page = 1, limit = 20): Promise<PaginatedResponse<Block>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/blocks',
      params: { page, limit },
    })
    return {
      data: response.blocks || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
      cached: response.cached,
    }
  }

  async getLatestBlocks(limit = 10): Promise<{ blocks: Block[] }> {
    return this.request({
      method: 'GET',
      url: '/blocks/latest',
      params: { limit },
    })
  }

  async getBlock(height: number): Promise<{ block: Block }> {
    return this.request({
      method: 'GET',
      url: `/blocks/${height}`,
    })
  }

  async getBlockTransactions(height: number): Promise<{ transactions: Transaction[]; count: number }> {
    return this.request({
      method: 'GET',
      url: `/blocks/${height}/transactions`,
    })
  }

  // Transactions API
  async getTransactions(page = 1, limit = 20, status?: string, type?: string): Promise<PaginatedResponse<Transaction>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/transactions',
      params: { page, limit, status, type },
    })
    return {
      data: response.transactions || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getLatestTransactions(limit = 10): Promise<{ transactions: Transaction[] }> {
    return this.request({
      method: 'GET',
      url: '/transactions/latest',
      params: { limit },
    })
  }

  async getTransaction(hash: string): Promise<{ transaction: Transaction }> {
    return this.request({
      method: 'GET',
      url: `/transactions/${hash}`,
    })
  }

  async getTransactionEvents(hash: string): Promise<{ events: ExplorerEvent[]; count: number }> {
    return this.request({
      method: 'GET',
      url: `/transactions/${hash}/events`,
    })
  }

  // Accounts API
  async getAccount(address: string): Promise<{ account: Account }> {
    return this.request({
      method: 'GET',
      url: `/accounts/${address}`,
    })
  }

  async getAccountTransactions(address: string, page = 1, limit = 20): Promise<PaginatedResponse<Transaction>> {
    const response = await this.request<any>({
      method: 'GET',
      url: `/accounts/${address}/transactions`,
      params: { page, limit },
    })
    return {
      data: response.transactions || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getAccountBalances(address: string): Promise<{ balances: AccountBalance[] }> {
    return this.request({
      method: 'GET',
      url: `/accounts/${address}/balances`,
    })
  }

  async getAccountTokens(address: string): Promise<{ tokens: AccountToken[] }> {
    return this.request({
      method: 'GET',
      url: `/accounts/${address}/tokens`,
    })
  }

  // Validators API
  async getValidators(page = 1, limit = 20, status?: string): Promise<PaginatedResponse<Validator>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/validators',
      params: { page, limit, status },
    })
    return {
      data: response.validators || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getActiveValidators(): Promise<{ validators: Validator[]; count: number }> {
    return this.request({
      method: 'GET',
      url: '/validators/active',
    })
  }

  async getValidator(address: string): Promise<{ validator: Validator }> {
    return this.request({
      method: 'GET',
      url: `/validators/${address}`,
    })
  }

  async getValidatorUptime(address: string, days = 30): Promise<{ uptime: any }> {
    return this.request({
      method: 'GET',
      url: `/validators/${address}/uptime`,
      params: { days },
    })
  }

  async getValidatorRewards(address: string, page = 1, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.request<any>({
      method: 'GET',
      url: `/validators/${address}/rewards`,
      params: { page, limit },
    })
    return {
      data: response.rewards || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  // DEX API
  async getDEXPools(page = 1, limit = 20, sortBy = 'tvl'): Promise<PaginatedResponse<DEXPool>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/dex/pools',
      params: { page, limit, sort: sortBy },
    })
    return {
      data: response.pools || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getDEXPool(poolId: string): Promise<{ pool: DEXPool }> {
    return this.request({
      method: 'GET',
      url: `/dex/pools/${poolId}`,
    })
  }

  async getPoolTrades(poolId: string, page = 1, limit = 20): Promise<PaginatedResponse<DEXTrade>> {
    const response = await this.request<any>({
      method: 'GET',
      url: `/dex/pools/${poolId}/trades`,
      params: { page, limit },
    })
    return {
      data: response.trades || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getPoolLiquidity(poolId: string, page = 1, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.request<any>({
      method: 'GET',
      url: `/dex/pools/${poolId}/liquidity`,
      params: { page, limit },
    })
    return {
      data: response.liquidity || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getPoolChart(poolId: string, period = '24h'): Promise<{ chart: any }> {
    return this.request({
      method: 'GET',
      url: `/dex/pools/${poolId}/chart`,
      params: { period },
    })
  }

  async getDEXTrades(page = 1, limit = 20): Promise<PaginatedResponse<DEXTrade>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/dex/trades',
      params: { page, limit },
    })
    return {
      data: response.trades || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getLatestDEXTrades(limit = 10): Promise<{ trades: DEXTrade[] }> {
    return this.request({
      method: 'GET',
      url: '/dex/trades/latest',
      params: { limit },
    })
  }

  // Oracle API
  async getOraclePrices(): Promise<{ prices: OraclePrice[] }> {
    return this.request({
      method: 'GET',
      url: '/oracle/prices',
    })
  }

  async getAssetPrice(asset: string): Promise<{ price: OraclePrice }> {
    return this.request({
      method: 'GET',
      url: `/oracle/prices/${asset}`,
    })
  }

  async getAssetPriceHistory(asset: string, period = '24h'): Promise<{ history: OraclePrice[] }> {
    return this.request({
      method: 'GET',
      url: `/oracle/prices/${asset}/history`,
      params: { period },
    })
  }

  async getAssetPriceChart(asset: string, period = '24h', interval = '1h'): Promise<{ chart: any }> {
    return this.request({
      method: 'GET',
      url: `/oracle/prices/${asset}/chart`,
      params: { period, interval },
    })
  }

  async getOracleSubmissions(page = 1, limit = 20, asset?: string): Promise<PaginatedResponse<OracleSubmission>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/oracle/submissions',
      params: { page, limit, asset },
    })
    return {
      data: response.submissions || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getOracleSlashes(page = 1, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/oracle/slashes',
      params: { page, limit },
    })
    return {
      data: response.slashes || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  // Compute API
  async getComputeRequests(page = 1, limit = 20, status?: string): Promise<PaginatedResponse<ComputeRequest>> {
    const response = await this.request<any>({
      method: 'GET',
      url: '/compute/requests',
      params: { page, limit, status },
    })
    return {
      data: response.requests || [],
      page: response.page || page,
      limit: response.limit || limit,
      total: response.total || 0,
    }
  }

  async getComputeRequest(requestId: string): Promise<{ request: ComputeRequest }> {
    return this.request({
      method: 'GET',
      url: `/compute/requests/${requestId}`,
    })
  }

  async getComputeResults(requestId: string): Promise<{ results: any[] }> {
    return this.request({
      method: 'GET',
      url: `/compute/requests/${requestId}/results`,
    })
  }

  async getComputeVerifications(requestId: string): Promise<{ verifications: any[] }> {
    return this.request({
      method: 'GET',
      url: `/compute/requests/${requestId}/verifications`,
    })
  }

  async getComputeProviders(): Promise<{ providers: ComputeProvider[] }> {
    return this.request({
      method: 'GET',
      url: '/compute/providers',
    })
  }

  async getComputeProvider(address: string): Promise<{ provider: ComputeProvider }> {
    return this.request({
      method: 'GET',
      url: `/compute/providers/${address}`,
    })
  }

  // Statistics API
  async getNetworkStats(): Promise<NetworkStats> {
    const response = await this.request<{ stats: any }>({
      method: 'GET',
      url: '/stats/network',
    })
    return response.stats
  }

  async getTransactionChart(period = '24h'): Promise<{ chart: any }> {
    return this.request({
      method: 'GET',
      url: '/stats/charts/transactions',
      params: { period },
    })
  }

  async getAddressChart(period = '30d'): Promise<{ chart: any }> {
    return this.request({
      method: 'GET',
      url: '/stats/charts/addresses',
      params: { period },
    })
  }

  async getVolumeChart(period = '7d'): Promise<{ chart: any }> {
    return this.request({
      method: 'GET',
      url: '/stats/charts/volume',
      params: { period },
    })
  }

  async getGasChart(period = '24h'): Promise<{ chart: any }> {
    return this.request({
      method: 'GET',
      url: '/stats/charts/gas',
      params: { period },
    })
  }

  // Search API
  async search(query: string): Promise<{ results: SearchResult[]; query: string }> {
    return this.request({
      method: 'GET',
      url: '/search',
      params: { q: query },
    })
  }

  // Export API
  async exportTransactions(
    address: string,
    format: 'csv' | 'json' = 'csv',
    startDate?: string,
    endDate?: string
  ): Promise<Blob> {
    const response = await this.client.request({
      method: 'GET',
      url: '/export/transactions',
      params: { address, format, start_date: startDate, end_date: endDate },
      responseType: 'blob',
    })
    return response.data
  }

  async exportTrades(poolId?: string, format: 'csv' | 'json' = 'csv', startDate?: string, endDate?: string): Promise<Blob> {
    const response = await this.client.request({
      method: 'GET',
      url: '/export/trades',
      params: { pool_id: poolId, format, start_date: startDate, end_date: endDate },
      responseType: 'blob',
    })
    return response.data
  }

  // WebSocket connection
  connectWebSocket(onMessage: (message: any) => void, onError?: (error: Event) => void): WebSocket {
    const ws = new WebSocket(WS_URL)

    ws.onopen = () => {
      console.log('WebSocket connected')
      // Subscribe to channels
      ws.send(
        JSON.stringify({
          action: 'subscribe',
          channels: ['blocks', 'transactions', 'dex_trades'],
        })
      )
    }

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        onMessage(message)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      if (onError) {
        onError(error)
      }
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
    }

    return ws
  }

  // GraphQL query
  async graphql(query: string, variables?: Record<string, any>): Promise<any> {
    const response = await axios.post(
      GRAPHQL_URL,
      {
        query,
        variables,
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )
    return response.data
  }
}

// Create singleton instance
export const api = new APIClient(API_BASE_URL)

// Export utility functions
export const downloadFile = (blob: Blob, filename: string) => {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', filename)
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

export default api
