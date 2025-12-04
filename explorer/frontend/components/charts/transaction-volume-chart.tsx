'use client'

import { useQuery } from '@tanstack/react-query'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { api } from '@/lib/api'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber, formatCurrency } from '@/lib/utils'

interface TransactionVolumeChartProps {
  period?: '24h' | '7d' | '30d'
  height?: number
}

export function TransactionVolumeChart({ period = '24h', height = 300 }: TransactionVolumeChartProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['transactionVolumeChart', period],
    queryFn: () => api.getVolumeChart(period),
    refetchInterval: 30000,
  })

  if (isLoading) {
    return <Skeleton className="w-full" style={{ height }} />
  }

  if (isError || !data?.chart) {
    return (
      <div className="flex items-center justify-center" style={{ height }}>
        <p className="text-sm text-muted-foreground">Failed to load chart data</p>
      </div>
    )
  }

  const chartData = data.chart.map((item: any) => ({
    time: item.time || item.timestamp,
    volume: parseFloat(item.volume || item.tx_volume || 0),
    count: item.count || item.tx_count || 0,
  }))

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
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
          yAxisId="left"
          className="text-xs"
          tickFormatter={(value) => formatCurrency(value, 0)}
        />
        <YAxis
          yAxisId="right"
          orientation="right"
          className="text-xs"
          tickFormatter={(value) => formatNumber(value)}
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
          formatter={(value: any, name: string) => {
            if (name === 'Volume') return formatCurrency(value)
            return formatNumber(value)
          }}
        />
        <Legend />
        <Bar yAxisId="left" dataKey="volume" fill="#3b82f6" name="Volume" radius={[8, 8, 0, 0]} />
        <Bar yAxisId="right" dataKey="count" fill="#10b981" name="Count" radius={[8, 8, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}
