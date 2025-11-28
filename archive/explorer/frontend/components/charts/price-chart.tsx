'use client'

import { useQuery } from '@tanstack/react-query'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { api } from '@/lib/api'
import { Skeleton } from '@/components/ui/skeleton'
import { formatCurrency } from '@/lib/utils'

interface PriceChartProps {
  asset: string
  period?: '24h' | '7d' | '30d'
  height?: number
}

export function PriceChart({ asset, period = '24h', height = 300 }: PriceChartProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['priceChart', asset, period],
    queryFn: () => api.getAssetPriceChart(asset, period, '1h'),
    refetchInterval: 30000,
  })

  if (isLoading) {
    return <Skeleton className="w-full" style={{ height }} />
  }

  if (isError || !data?.chart) {
    return (
      <div className="flex items-center justify-center" style={{ height }}>
        <p className="text-sm text-muted-foreground">Failed to load price data</p>
      </div>
    )
  }

  const chartData = data.chart.map((item: any) => ({
    time: item.timestamp || item.time,
    price: parseFloat(item.price || item.median || 0),
  }))

  const minPrice = Math.min(...chartData.map((d: any) => d.price))
  const maxPrice = Math.max(...chartData.map((d: any) => d.price))
  const priceChange = chartData.length > 0 ? chartData[chartData.length - 1].price - chartData[0].price : 0
  const isPositive = priceChange >= 0

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
        <defs>
          <linearGradient id={`colorPrice-${asset}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={isPositive ? '#10b981' : '#ef4444'} stopOpacity={0.3} />
            <stop offset="95%" stopColor={isPositive ? '#10b981' : '#ef4444'} stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis
          dataKey="time"
          className="text-xs"
          tickFormatter={(value) => {
            const date = new Date(value)
            if (period === '24h') {
              return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
            }
            return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
          }}
        />
        <YAxis
          className="text-xs"
          domain={[minPrice * 0.99, maxPrice * 1.01]}
          tickFormatter={(value) => formatCurrency(value)}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'hsl(var(--popover))',
            border: '1px solid hsl(var(--border))',
            borderRadius: '8px',
          }}
          labelFormatter={(value) => {
            const date = new Date(value)
            return date.toLocaleString()
          }}
          formatter={(value: any) => [formatCurrency(value), 'Price']}
        />
        <Area
          type="monotone"
          dataKey="price"
          stroke={isPositive ? '#10b981' : '#ef4444'}
          fillOpacity={1}
          fill={`url(#colorPrice-${asset})`}
          strokeWidth={2}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
