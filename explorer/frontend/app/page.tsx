'use client'

import { useEffect, useState, useMemo } from 'react'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { formatDistanceToNow } from 'date-fns'
import {
  Activity,
  Blocks,
  DollarSign,
  TrendingUp,
  Users,
  Zap,
  ArrowRight,
  Clock,
  CheckCircle,
  XCircle,
  Search,
  Globe,
  Box,
  Database,
} from 'lucide-react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { NetworkStatsChart } from '@/components/charts/network-stats-chart'
import { TransactionVolumeChart } from '@/components/charts/transaction-volume-chart'
import { api } from '@/lib/api'
import { formatNumber, formatCurrency, formatHash, formatGas } from '@/lib/utils'
import { useSearch } from '@/hooks/use-search'
import { useWebSocket } from '@/hooks/use-websocket'

interface NetworkStats {
  totalBlocks: number
  totalTransactions: number
  activeValidators: number
  averageBlockTime: number
  tps: number
  tvl: string
  dexVolume24h: string
  activeAccounts24h: number
}

interface Block {
  height: number
  hash: string
  proposer_address: string
  timestamp: string
  tx_count: number
  gas_used: number
}

interface Transaction {
  hash: string
  block_height: number
  type: string
  sender: string
  status: string
  timestamp: string
  fee_amount: string
  fee_denom: string
}

export default function HomePage() {
  const [searchQuery, setSearchQuery] = useState('')
  const { search, results, isSearching } = useSearch()
  const { lastMessage, isConnected } = useWebSocket()

  // Fetch network stats
  const { data: stats, isLoading: isLoadingStats } = useQuery<NetworkStats>({
    queryKey: ['networkStats'],
    queryFn: () => api.getNetworkStats(),
    refetchInterval: 10000, // Refetch every 10 seconds
  })

  // Fetch latest blocks
  const { data: blocksData, isLoading: isLoadingBlocks } = useQuery({
    queryKey: ['latestBlocks'],
    queryFn: () => api.getLatestBlocks(10),
    refetchInterval: 5000, // Refetch every 5 seconds
  })

  // Fetch latest transactions
  const { data: txsData, isLoading: isLoadingTxs } = useQuery({
    queryKey: ['latestTransactions'],
    queryFn: () => api.getLatestTransactions(10),
    refetchInterval: 5000,
  })

  const { data: oracleOverview } = useQuery({
    queryKey: ['homeOraclePrices'],
    queryFn: () => api.getOraclePrices(),
    refetchInterval: 20000,
  })

  const { data: activeCompute } = useQuery({
    queryKey: ['homeComputeJobs'],
    queryFn: () => api.getComputeRequests(1, 50, 'active'),
    refetchInterval: 20000,
  })

  const { data: dexTrades } = useQuery({
    queryKey: ['homeDexTrades'],
    queryFn: () => api.getDEXTrades(1, 50),
    refetchInterval: 20000,
  })

  const oracleFeedCount = oracleOverview?.prices?.length ?? 0
  const totalOracleSubmissions = useMemo(
    () =>
      oracleOverview?.prices?.reduce((acc, price) => acc + (price.num_submissions || 0), 0) ??
      0,
    [oracleOverview]
  )
  const activeJobs = activeCompute?.data?.length ?? 0
  const recentDexTrades = dexTrades?.data?.length ?? 0

  // Handle WebSocket updates for real-time data
  useEffect(() => {
    if (lastMessage) {
      console.log('Received WebSocket message:', lastMessage)
      // Handle real-time updates here
    }
  }, [lastMessage])

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      await search(searchQuery)
    }
  }

  return (
    <div className="container mx-auto py-8 space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-4xl font-bold tracking-tight">PAW Chain Explorer</h1>
            <p className="text-muted-foreground mt-2">
              Real-time blockchain data and analytics
            </p>
          </div>
          <div className="flex items-center gap-2">
            <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
            <span className="text-sm text-muted-foreground">
              {isConnected ? 'Live' : 'Disconnected'}
            </span>
          </div>
        </div>

        {/* Search Bar */}
        <form onSubmit={handleSearch} className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search blocks, transactions, addresses, validators..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 pr-4 py-6 text-lg"
          />
          {isSearching && (
            <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
              <div className="animate-spin h-5 w-5 border-2 border-primary border-t-transparent rounded-full" />
            </div>
          )}
        </form>
      </div>

      {/* Network Statistics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Blocks</CardTitle>
            <Blocks className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoadingStats ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <>
                <div className="text-2xl font-bold">{formatNumber(stats?.totalBlocks || 0)}</div>
                <p className="text-xs text-muted-foreground">
                  ~{stats?.averageBlockTime || 0}s block time
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Transactions</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoadingStats ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <>
                <div className="text-2xl font-bold">
                  {formatNumber(stats?.totalTransactions || 0)}
                </div>
                <p className="text-xs text-muted-foreground">
                  {stats?.tps || 0} TPS current
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Validators</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoadingStats ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <>
                <div className="text-2xl font-bold">{stats?.activeValidators || 0}</div>
                <p className="text-xs text-muted-foreground">Securing the network</p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Value Locked</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoadingStats ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <>
                <div className="text-2xl font-bold">{formatCurrency(stats?.tvl || '0')}</div>
                <p className="text-xs text-muted-foreground">
                  {formatCurrency(stats?.dexVolume24h || '0')} 24h volume
                </p>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Charts */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Network Activity</CardTitle>
            <CardDescription>Transaction volume over time</CardDescription>
          </CardHeader>
          <CardContent>
            <TransactionVolumeChart period="24h" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Network Health</CardTitle>
            <CardDescription>Block production and validator performance</CardDescription>
          </CardHeader>
          <CardContent>
            <NetworkStatsChart />
          </CardContent>
        </Card>
      </div>

      {/* Quick Links */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="hover:shadow-lg transition-shadow cursor-pointer">
          <Link href="/dex">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                DEX Analytics
              </CardTitle>
              <CardDescription>Explore liquidity pools and trading activity</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">24h Volume</p>
                  <p className="text-2xl font-bold">{formatCurrency(stats?.dexVolume24h || '0')}</p>
                </div>
                <ArrowRight className="h-5 w-5 text-muted-foreground" />
              </div>
            </CardContent>
          </Link>
        </Card>

        <Card className="hover:shadow-lg transition-shadow cursor-pointer">
          <Link href="/oracle">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Zap className="h-5 w-5" />
                Oracle Prices
              </CardTitle>
              <CardDescription>Real-time price feeds from validators</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Price Feeds</p>
                  <p className="text-2xl font-bold">{oracleFeedCount}</p>
                </div>
                <ArrowRight className="h-5 w-5 text-muted-foreground" />
              </div>
            </CardContent>
          </Link>
        </Card>

        <Card className="hover:shadow-lg transition-shadow cursor-pointer">
          <Link href="/compute">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Database className="h-5 w-5" />
                Compute Jobs
              </CardTitle>
              <CardDescription>Decentralized computation marketplace</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Active Jobs</p>
                  <p className="text-2xl font-bold">{activeJobs}</p>
                </div>
                <ArrowRight className="h-5 w-5 text-muted-foreground" />
              </div>
            </CardContent>
          </Link>
        </Card>
      </div>

      {/* Latest Blocks and Transactions */}
      <div className="grid gap-4 md:grid-cols-2">
        {/* Latest Blocks */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Latest Blocks</CardTitle>
              <Link href="/blocks">
                <Button variant="ghost" size="sm">
                  View all
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingBlocks ? (
              <div className="space-y-4">
                {[...Array(5)].map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                {blocksData?.blocks?.map((block: Block) => (
                  <Link
                    key={block.height}
                    href={`/blocks/${block.height}`}
                    className="flex items-center justify-between p-4 border rounded-lg hover:bg-accent transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <Box className="h-8 w-8 text-primary" />
                      <div>
                        <p className="font-semibold">Block #{formatNumber(block.height)}</p>
                        <p className="text-sm text-muted-foreground">
                          {formatDistanceToNow(new Date(block.timestamp), { addSuffix: true })}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium">{block.tx_count} txs</p>
                      <p className="text-xs text-muted-foreground">
                        {formatGas(block.gas_used)} gas
                      </p>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Latest Transactions */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Latest Transactions</CardTitle>
              <Link href="/transactions">
                <Button variant="ghost" size="sm">
                  View all
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {isLoadingTxs ? (
              <div className="space-y-4">
                {[...Array(5)].map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                {txsData?.transactions?.map((tx: Transaction) => (
                  <Link
                    key={tx.hash}
                    href={`/transactions/${tx.hash}`}
                    className="flex items-center justify-between p-4 border rounded-lg hover:bg-accent transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      {tx.status === 'success' ? (
                        <CheckCircle className="h-5 w-5 text-green-500" />
                      ) : (
                        <XCircle className="h-5 w-5 text-red-500" />
                      )}
                      <div>
                        <p className="font-mono text-sm">{formatHash(tx.hash)}</p>
                        <p className="text-sm text-muted-foreground">
                          {formatDistanceToNow(new Date(tx.timestamp), { addSuffix: true })}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <Badge variant={tx.status === 'success' ? 'default' : 'destructive'}>
                        {tx.status}
                      </Badge>
                      <p className="text-xs text-muted-foreground mt-1">
                        Block #{formatNumber(tx.block_height)}
                      </p>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Additional Statistics */}
      <Card>
        <CardHeader>
          <CardTitle>Network Statistics (24h)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-4">
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Active Accounts</p>
              <p className="text-2xl font-bold">{formatNumber(stats?.activeAccounts24h || 0)}</p>
              <p className="text-xs text-green-500">+12.5% from yesterday</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Average TPS</p>
              <p className="text-2xl font-bold">{stats?.tps || 0}</p>
              <p className="text-xs text-green-500">+5.2% from yesterday</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">DEX Trades</p>
              <p className="text-2xl font-bold">{formatNumber(recentDexTrades)}</p>
              <p className="text-xs text-green-500">Live past hour</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Oracle Updates</p>
              <p className="text-2xl font-bold">{formatNumber(totalOracleSubmissions || 0)}</p>
              <p className="text-xs text-blue-500">Last reporting window</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
