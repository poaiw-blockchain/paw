'use client'

import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, BarChart, Bar } from 'recharts'
import { api, type DEXPool } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { formatCurrency, formatNumber, formatPercent } from '@/lib/utils'

type DexMetric = 'tvl' | 'volume'

const metricTabs: { label: string; value: DexMetric; description: string }[] = [
  { label: 'Total Value Locked', value: 'tvl', description: 'Liquidity depth across top pools' },
  { label: '24h Volume', value: 'volume', description: 'Execution liquidity over the last 24h' },
]

export function DexPoolVisualization() {
  const [metric, setMetric] = useState<DexMetric>('tvl')

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['dexPools', metric],
    queryFn: () => api.getDEXPools(1, 8, metric === 'tvl' ? 'tvl' : 'volume_24h'),
    refetchInterval: 30_000,
  })

  const pools = data?.data ?? []
  const dominantPool = pools[0]

  const chartData = useMemo(() => {
    return pools.map((pool) => {
      const aprValue = parseFloat(pool.apr || '0')
      return {
        name: `Pool #${pool.pool_id}`,
        pair: `${pool.token_a}/${pool.token_b}`,
        tvl: parseFloat(pool.tvl || '0'),
        volume: parseFloat(pool.volume_24h || '0'),
        aprPercent: aprValue * 100,
      }
    })
  }, [pools])

  const totalTVL = pools.reduce((acc, pool) => acc + (parseFloat(pool.tvl || '0') || 0), 0)

  if (isLoading) {
    return <Skeleton className="h-[420px] w-full" />
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>DEX Liquidity</CardTitle>
          <CardDescription>Unable to load DEX analytics at the moment.</CardDescription>
        </CardHeader>
        <CardContent>
          <button
            className="text-sm text-primary underline"
            onClick={() => refetch()}
          >
            Retry
          </button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="border-primary/20">
      <CardHeader className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>DEX Liquidity Topology</CardTitle>
            <CardDescription>Live pool analytics with composable liquidity heatmaps.</CardDescription>
          </div>
          {dominantPool && (
            <div className="text-right">
              <p className="text-xs text-muted-foreground">Dominant Pool</p>
              <p className="font-semibold">
                #{dominantPool.pool_id} {dominantPool.token_a}/{dominantPool.token_b}
              </p>
              <p className="text-sm text-muted-foreground">
                {totalTVL > 0
                  ? formatPercent((parseFloat(dominantPool.tvl || '0') / totalTVL) * 100, 1)
                  : '0%'}{' '}
                TVL share
              </p>
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <Tabs value={metric} onValueChange={(value) => setMetric(value as DexMetric)}>
          <TabsList>
            {metricTabs.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>
          {metricTabs.map((tab) => (
            <TabsContent key={tab.value} value={tab.value}>
              <p className="text-sm text-muted-foreground mb-4">{tab.description}</p>
              <div className="grid gap-4 lg:grid-cols-2">
                <div className="h-[260px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={chartData}>
                      <defs>
                        <linearGradient id="dexColor" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#7aa2f7" stopOpacity={0.5} />
                          <stop offset="95%" stopColor="#7aa2f7" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                      <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                      <YAxis
                        tickFormatter={(value) => formatNumber(value)}
                        tick={{ fontSize: 12 }}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--popover))',
                          border: '1px solid hsl(var(--border))',
                        }}
                        formatter={(value) => formatCurrency(value as number)}
                      />
                      <Legend />
                      <Area
                        type="monotone"
                        dataKey={tab.value === 'tvl' ? 'tvl' : 'volume'}
                        stroke="#7aa2f7"
                        fillOpacity={1}
                        fill="url(#dexColor)"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>

                <div className="h-[260px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={chartData}>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                      <XAxis dataKey="pair" tick={{ fontSize: 12 }} />
                      <YAxis tickFormatter={(value) => formatPercent(value, 2)} tick={{ fontSize: 12 }} />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--popover))',
                          border: '1px solid hsl(var(--border))',
                        }}
                        formatter={(value) => `${formatPercent(value as number, 2)} APR`}
                      />
                      <Legend />
                      <Bar dataKey="aprPercent" name="APR" fill="#10b981" radius={[6, 6, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>
            </TabsContent>
          ))}
        </Tabs>

        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pool</TableHead>
                <TableHead>Pair</TableHead>
                <TableHead className="text-right">TVL</TableHead>
                <TableHead className="text-right">24h Volume</TableHead>
                <TableHead className="text-right">APR</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pools.map((pool) => (
                <TableRow key={pool.pool_id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">#{pool.pool_id}</Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p className="font-medium">
                        {pool.token_a}/{pool.token_b}
                      </p>
                      <p className="text-xs text-muted-foreground">Fee {formatPercent(parseFloat(pool.swap_fee || '0') * 100)}</p>
                    </div>
                  </TableCell>
                  <TableCell className="text-right">{formatCurrency(pool.tvl || '0')}</TableCell>
                  <TableCell className="text-right">{formatCurrency(pool.volume_24h || '0')}</TableCell>
                  <TableCell className="text-right">
                    <Badge variant="secondary">{formatPercent(parseFloat(pool.apr || '0') * 100)}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
