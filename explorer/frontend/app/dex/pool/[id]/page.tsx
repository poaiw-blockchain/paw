'use client'

import { use, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api, type DEXTrade } from '@/lib/api'
import { formatCurrency, formatHash, formatNumber, formatPercent } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'
import { ArrowLeft, TrendingUp, TrendingDown, DollarSign, Activity } from 'lucide-react'

type PageParams = {
  params: Promise<{ id: string }>
}

export default function PoolDetailPage({ params }: PageParams) {
  const resolvedParams = use(params)
  const poolId = resolvedParams.id
  const [chartPeriod, setChartPeriod] = useState<'1h' | '24h' | '7d' | '30d' | '1y'>('24h')

  // Fetch pool data
  const { data: poolData, isLoading: poolLoading } = useQuery({
    queryKey: ['dexPool', poolId],
    queryFn: () => api.getDEXPool(poolId),
    refetchInterval: 30_000,
  })

  // Fetch pool statistics
  const { data: statsData } = useQuery({
    queryKey: ['poolStats', poolId, chartPeriod],
    queryFn: () => fetch(`${process.env.NEXT_PUBLIC_API_URL}/dex/pools/${poolId}/statistics?period=${chartPeriod}`)
      .then(res => res.json()),
    refetchInterval: 60_000,
  })

  // Fetch pool price history for chart
  const { data: priceHistoryData } = useQuery({
    queryKey: ['poolPriceHistory', poolId, chartPeriod],
    queryFn: () => fetch(`${process.env.NEXT_PUBLIC_API_URL}/dex/pools/${poolId}/price-history?period=${chartPeriod}`)
      .then(res => res.json()),
    refetchInterval: 30_000,
  })

  // Fetch pool trades
  const {
    data: tradesData,
    isLoading: tradesLoading,
    isError: tradesError,
    refetch: refetchTrades,
  } = useQuery({
    queryKey: ['poolTrades', poolId],
    queryFn: () => api.getPoolTrades(poolId, 1, 10),
    refetchInterval: 15_000,
  })

  if (poolLoading) {
    return (
      <div className="container mx-auto py-8">
        <div className="animate-pulse space-y-6">
          <div className="h-12 bg-muted rounded w-1/3" />
          <div className="grid gap-4 md:grid-cols-4">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="h-32 bg-muted rounded" />
            ))}
          </div>
          <div className="h-96 bg-muted rounded" />
        </div>
      </div>
    )
  }

  const pool = poolData?.pool
  if (!pool) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="py-8">
            <p className="text-center text-muted-foreground">Pool not found</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const stats = statsData?.statistics?.[0]
  const priceChange = stats?.price_change_percent ? parseFloat(stats.price_change_percent) : 0
  const isPriceUp = priceChange >= 0

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Back button */}
      <Link href="/dex">
        <Button variant="ghost" size="sm">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to DEX
        </Button>
      </Link>

      {/* Pool Header */}
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-4xl font-bold">
            {pool.token_a.toUpperCase()} / {pool.token_b.toUpperCase()}
          </h1>
          <p className="text-muted-foreground mt-2">
            Pool #{poolId} • Fee: {formatPercent(parseFloat(pool.swap_fee) * 100)}
          </p>
        </div>

        <div className="flex gap-3">
          <Badge variant="secondary" className="h-fit">
            APR: {formatPercent(parseFloat(pool.apr) * 100)}
          </Badge>
          {isPriceUp ? (
            <Badge variant="default" className="h-fit bg-green-500">
              <TrendingUp className="mr-1 h-3 w-3" />
              {formatPercent(Math.abs(priceChange))}
            </Badge>
          ) : (
            <Badge variant="default" className="h-fit bg-red-500">
              <TrendingDown className="mr-1 h-3 w-3" />
              {formatPercent(Math.abs(priceChange))}
            </Badge>
          )}
        </div>
      </div>

      {/* Statistics Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Value Locked</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${formatNumber(pool.tvl)}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {formatNumber(pool.reserve_a)} {pool.token_a.toUpperCase()} + {formatNumber(pool.reserve_b)} {pool.token_b.toUpperCase()}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">24h Volume</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${formatNumber(pool.volume_24h)}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {stats?.trade_count || 0} trades
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">24h Fees</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${formatNumber(stats?.fees_usd || '0')}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {formatNumber(stats?.fees_collected_a || '0')} {pool.token_a.toUpperCase()}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Current Price</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {stats?.avg_price ? formatNumber(stats.avg_price) : '-'}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {pool.token_b.toUpperCase()} per {pool.token_a.toUpperCase()}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Charts Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Price History</CardTitle>
            <div className="flex gap-2">
              {(['1h', '24h', '7d', '30d', '1y'] as const).map((period) => (
                <Button
                  key={period}
                  variant={chartPeriod === period ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setChartPeriod(period)}
                >
                  {period}
                </Button>
              ))}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {priceHistoryData?.price_history && priceHistoryData.price_history.length > 0 ? (
            <div className="h-80 flex items-center justify-center bg-muted/20 rounded">
              <div className="text-center text-muted-foreground">
                <Activity className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Chart visualization would render here</p>
                <p className="text-xs mt-2">{priceHistoryData.price_history.length} data points available</p>
              </div>
            </div>
          ) : (
            <div className="h-80 flex items-center justify-center">
              <p className="text-muted-foreground">No price history data available</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Recent Trades */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Trades</CardTitle>
          <CardDescription>Latest swaps executed in this pool</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>Trader</TableHead>
                  <TableHead>Amount In</TableHead>
                  <TableHead>Amount Out</TableHead>
                  <TableHead>Time</TableHead>
                  <TableHead>Tx</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tradesLoading ? (
                  [...Array(5)].map((_, idx) => (
                    <TableRow key={idx}>
                      <TableCell colSpan={6} className="h-12">
                        <div className="h-4 w-1/3 animate-pulse rounded bg-muted" />
                      </TableCell>
                    </TableRow>
                  ))
                ) : tradesError ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-sm text-muted-foreground">
                      Failed to load pool trades.{' '}
                      <button className="text-primary underline" onClick={() => refetchTrades()}>
                        Retry
                      </button>
                    </TableCell>
                  </TableRow>
                ) : tradesData && tradesData.data && tradesData.data.length > 0 ? (
                  tradesData.data.map((trade: DEXTrade, idx: number) => (
                    <TableRow key={`${trade.tx_hash}-${idx}`}>
                      <TableCell>
                        <Badge variant="outline">
                          {trade.token_in === pool.token_a ? 'Buy' : 'Sell'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Link href={`/account/${trade.trader}`} className="hover:underline">
                          <span className="font-mono text-xs">{formatHash(trade.trader)}</span>
                        </Link>
                      </TableCell>
                      <TableCell>
                        {formatCurrency(trade.amount_in || '0')} {trade.token_in.toUpperCase()}
                      </TableCell>
                      <TableCell>
                        {formatCurrency(trade.amount_out || '0')} {trade.token_out.toUpperCase()}
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(trade.timestamp), { addSuffix: true })}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Link href={`/tx/${trade.tx_hash}`} className="hover:underline">
                          <span className="font-mono text-xs">{formatHash(trade.tx_hash)}</span>
                        </Link>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      No recent trades
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Pool Details */}
      <Card>
        <CardHeader>
          <CardTitle>Pool Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <h4 className="text-sm font-medium text-muted-foreground mb-2">Token A</h4>
              <p className="text-lg font-semibold">{pool.token_a.toUpperCase()}</p>
              <p className="text-sm text-muted-foreground">Reserve: {formatNumber(pool.reserve_a)}</p>
            </div>
            <div>
              <h4 className="text-sm font-medium text-muted-foreground mb-2">Token B</h4>
              <p className="text-lg font-semibold">{pool.token_b.toUpperCase()}</p>
              <p className="text-sm text-muted-foreground">Reserve: {formatNumber(pool.reserve_b)}</p>
            </div>
            <div>
              <h4 className="text-sm font-medium text-muted-foreground mb-2">Total Shares</h4>
              <p className="text-lg font-semibold">{formatNumber(pool.total_shares)}</p>
            </div>
            <div>
              <h4 className="text-sm font-medium text-muted-foreground mb-2">Created</h4>
              <p className="text-lg font-semibold">
                {new Date(pool.created_at).toLocaleDateString()}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
