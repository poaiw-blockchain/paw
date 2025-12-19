'use client'

import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ResponsiveContainer, BarChart, Bar, CartesianGrid, XAxis, YAxis, Tooltip, Legend, ComposedChart, Line } from 'recharts'
import { api } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { formatCurrency, formatNumber, formatPercent } from '@/lib/utils'

const SLIPPAGE_PROBE_AMOUNT = 5_000

function normalizeNumber(value?: string): number {
  const parsed = parseFloat(value || '0')
  return isNaN(parsed) ? 0 : parsed
}

export function DexAdvancedAnalytics() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['dexPools', 'advanced'],
    queryFn: () => api.getDEXPools(1, 12, 'tvl'),
    refetchInterval: 30_000,
  })

  const pools = useMemo(() => data?.data ?? [], [data])

  const stats = useMemo(() => {
    if (pools.length === 0) {
      return {
        concentration: 0,
        avgFee: 0,
        efficiency: 0,
        poolCount: 0,
      }
    }

    const totalTVL = pools.reduce((sum, pool) => sum + normalizeNumber(pool.tvl), 0)
    const concentration = pools.reduce((sum, pool) => {
      const share = totalTVL > 0 ? normalizeNumber(pool.tvl) / totalTVL : 0
      return sum + share * share
    }, 0)

    const avgFee = pools.reduce((sum, pool) => sum + normalizeNumber(pool.swap_fee || '0'), 0) / pools.length
    const efficiency =
      pools.reduce((sum, pool) => sum + (normalizeNumber(pool.volume_24h) / (normalizeNumber(pool.tvl) || 1)), 0) / pools.length

    return {
      concentration: concentration * 100,
      avgFee: avgFee * 100,
      efficiency: efficiency * 100,
      poolCount: pools.length,
    }
  }, [pools])

  const efficiencyChart = useMemo(() => {
    return pools
      .map((pool) => {
        const tvl = normalizeNumber(pool.tvl)
        const volume = normalizeNumber(pool.volume_24h)
        const efficiency = tvl > 0 ? (volume / tvl) * 100 : 0
        const apr = parseFloat(pool.apr || '0') * 100
        return {
          name: `#${pool.pool_id}`,
          pair: `${pool.token_a}/${pool.token_b}`,
          efficiency,
          apr,
        }
      })
      .sort((a, b) => b.efficiency - a.efficiency)
      .slice(0, 8)
  }, [pools])

  const volumeChartData = useMemo(() => {
    return pools.slice(0, 8).map((pool) => ({
      name: `#${pool.pool_id}`,
      tvl: normalizeNumber(pool.tvl),
      volume: normalizeNumber(pool.volume_24h),
    }))
  }, [pools])

  const shallowPools = useMemo(() => {
    return pools
      .map((pool) => {
        const reserveA = normalizeNumber(pool.reserve_a)
        const depth = normalizeNumber(pool.tvl)
        const slippage = reserveA > 0 ? Math.min(99, (SLIPPAGE_PROBE_AMOUNT / reserveA) * 100) : 0
        return {
          id: pool.pool_id,
          pair: `${pool.token_a}/${pool.token_b}`,
          depth,
          slippage,
        }
      })
      .sort((a, b) => a.depth - b.depth)
      .slice(0, 5)
  }, [pools])

  if (isLoading) {
    return <Skeleton className="h-[420px] w-full" />
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>DEX Advanced Analytics</CardTitle>
          <CardDescription>Failed to load expanded pool analytics.</CardDescription>
        </CardHeader>
        <CardContent>
          <button className="text-sm text-primary underline" onClick={() => refetch()}>
            Retry
          </button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Advanced Pool Analytics</CardTitle>
        <CardDescription>Capital efficiency, fee tiers, and shallow pool alerts.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Liquidity Concentration</CardDescription>
              <CardTitle className="text-3xl">{formatPercent(stats.concentration, 1)}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Herfindahl index of current liquidity</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Weighted Avg Fee</CardDescription>
              <CardTitle className="text-3xl">{formatPercent(stats.avgFee, 2)}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Average swap fee tier across pools</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Capital Efficiency</CardDescription>
              <CardTitle className="text-3xl">{formatPercent(stats.efficiency, 1)}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Volume/TVL ratio (24h)</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Active Pools</CardDescription>
              <CardTitle className="text-3xl">{formatNumber(stats.poolCount)}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Tracked pools in explorer index</p>
            </CardContent>
          </Card>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <div className="h-[280px]">
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart data={efficiencyChart}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="name" className="text-xs" />
                <YAxis
                  yAxisId="left"
                  tickFormatter={(value) => formatPercent(value, 1)}
                  className="text-xs"
                />
                <YAxis
                  yAxisId="right"
                  orientation="right"
                  tickFormatter={(value) => formatPercent(value, 1)}
                  className="text-xs"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--popover))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                  }}
                  formatter={(value, name) =>
                    name === 'Capital Efficiency'
                      ? [formatPercent(value as number, 2), 'Capital Efficiency']
                      : [formatPercent(value as number, 2), 'APR']
                  }
                />
                <Legend />
                <Bar yAxisId="left" dataKey="efficiency" name="Capital Efficiency" fill="#0ea5e9" radius={[6, 6, 0, 0]} />
                <Line yAxisId="right" type="monotone" dataKey="apr" name="APR" stroke="#f97316" strokeWidth={2} />
              </ComposedChart>
            </ResponsiveContainer>
          </div>
          <div className="h-[280px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={volumeChartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="name" className="text-xs" />
                <YAxis
                  tickFormatter={(value) => formatNumber(Math.round(value as number))}
                  className="text-xs"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--popover))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                  }}
                  formatter={(value) => formatCurrency(value as number)}
                />
                <Legend />
                <Bar dataKey="tvl" name="TVL" fill="#22c55e" radius={[6, 6, 0, 0]} />
                <Bar dataKey="volume" name="24h Volume" fill="#6366f1" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pool</TableHead>
                <TableHead>Depth</TableHead>
                <TableHead>Probe Slippage</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {shallowPools.map((pool) => (
                <TableRow key={pool.id}>
                  <TableCell>
                    <p className="font-medium">
                      Pool #{pool.id} <span className="text-xs text-muted-foreground">{pool.pair}</span>
                    </p>
                  </TableCell>
                  <TableCell>{formatCurrency(pool.depth)}</TableCell>
                  <TableCell>{formatPercent(pool.slippage, 2)}</TableCell>
                </TableRow>
              ))}
              {shallowPools.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                    No pool depth information available.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
