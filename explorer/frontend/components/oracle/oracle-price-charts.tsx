'use client'

import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip } from 'recharts'
import { api, type OraclePrice } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatCurrency, formatNumber } from '@/lib/utils'

const periods = [
  { label: '24H', value: '24h', interval: '1h' },
  { label: '7D', value: '7d', interval: '4h' },
  { label: '30D', value: '30d', interval: '1d' },
]

export function OraclePriceCharts() {
  const [selectedAsset, setSelectedAsset] = useState<string>()
  const [period, setPeriod] = useState(periods[0])

  const {
    data: prices,
    isLoading: isLoadingAssets,
    isError,
  } = useQuery({
    queryKey: ['oraclePrices'],
    queryFn: () => api.getOraclePrices(),
    refetchInterval: 20_000,
  })

  const { data: chartData, isLoading: isLoadingChart } = useQuery({
    enabled: !!selectedAsset,
    queryKey: ['oraclePriceChart', selectedAsset, period.value],
    queryFn: () => api.getAssetPriceChart(selectedAsset!, period.value, period.interval),
    refetchInterval: 20_000,
  })

  const assets = useMemo(() => prices?.prices ?? [], [prices?.prices])

  useEffect(() => {
    if (!selectedAsset && assets.length > 0) {
      setSelectedAsset(assets[0].asset)
    }
  }, [assets, selectedAsset])

  const selectedPrice = useMemo<OraclePrice | undefined>(
    () => assets.find((price) => price.asset === selectedAsset),
    [assets, selectedAsset]
  )

  if (isLoadingAssets) {
    return <Skeleton className="h-[420px] w-full" />
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Oracle Price Feeds</CardTitle>
          <CardDescription>Failed to load oracle state. Ensure the indexer is reachable.</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <CardTitle>Oracle Price Intelligence</CardTitle>
          <CardDescription>Validator submissions, drift, and volatility for high-value assets.</CardDescription>
        </div>
        <Tabs defaultValue={period.value} value={period.value} onValueChange={(value) => setPeriod(periods.find((p) => p.value === value) || periods[0])}>
          <TabsList>
            {periods.map((p) => (
              <TabsTrigger key={p.value} value={p.value}>
                {p.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex gap-2 overflow-x-auto pb-2">
          {assets.map((asset) => (
            <button
              key={asset.asset}
              onClick={() => setSelectedAsset(asset.asset)}
              className={`rounded-lg border px-4 py-2 text-left transition ${
                selectedAsset === asset.asset ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50'
              }`}
            >
              <p className="text-sm text-muted-foreground">{asset.asset}</p>
              <p className="text-xl font-semibold">{formatCurrency(asset.price)}</p>
              <p className="text-xs text-muted-foreground">
                Median {formatCurrency(asset.median)} • Sources {asset.num_validators}
              </p>
            </button>
          ))}
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Oracle Median</CardDescription>
              <CardTitle className="text-2xl">{formatCurrency(selectedPrice?.median || '0')}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">Avg submissions {formatNumber(selectedPrice?.num_submissions || 0)}</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Standard Deviation</CardDescription>
              <CardTitle className="text-2xl">
                {formatCurrency(selectedPrice?.std_deviation || '0')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">Confidence {formatCurrency(selectedPrice?.confidence_score || '0')}</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Validator Coverage</CardDescription>
              <CardTitle className="text-2xl">{selectedPrice?.num_validators || 0} feeders</CardTitle>
            </CardHeader>
            <CardContent>
              <Badge variant="secondary">live</Badge>
            </CardContent>
          </Card>
        </div>

        <div className="h-[320px]">
          {isLoadingChart ? (
            <Skeleton className="h-full w-full" />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={chartData?.chart || []}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="timestamp"
                  tickFormatter={(value) => new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                />
                <YAxis tickFormatter={(value) => formatCurrency(value)} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--popover))',
                    border: '1px solid hsl(var(--border))',
                  }}
                  formatter={(value) => formatCurrency(value as number)}
                />
                <Line type="monotone" dataKey="price" stroke="#f97316" dot={false} strokeWidth={2} />
                <Line type="monotone" dataKey="median" stroke="#6366f1" dot={false} strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
