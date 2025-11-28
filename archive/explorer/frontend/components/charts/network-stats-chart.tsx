'use client'

import { useQuery } from '@tanstack/react-query'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { api } from '@/lib/api'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber } from '@/lib/utils'

interface NetworkStatsChartProps {
  period?: '24h' | '7d' | '30d'
  height?: number
}

export function NetworkStatsChart({ period = '24h', height = 300 }: NetworkStatsChartProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['networkStatsChart', period],
    queryFn: () => api.getTransactionChart(period),
    refetchInterval: 30000, // Refetch every 30 seconds
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
    transactions: item.transactions || item.tx_count || 0,
    blocks: item.blocks || item.block_count || 0,
  }))

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis
          dataKey="time"
          className="text-xs"
          tickFormatter={(value) => {
            const date = new Date(value)
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
          }}
        />
        <YAxis className="text-xs" tickFormatter={(value) => formatNumber(value)} />
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
          formatter={(value: any) => formatNumber(value)}
        />
        <Legend />
        <Line type="monotone" dataKey="transactions" stroke="#3b82f6" strokeWidth={2} name="Transactions" dot={false} />
        <Line type="monotone" dataKey="blocks" stroke="#10b981" strokeWidth={2} name="Blocks" dot={false} />
      </LineChart>
    </ResponsiveContainer>
  )
}
