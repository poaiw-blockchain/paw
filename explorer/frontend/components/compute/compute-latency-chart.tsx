'use client'

import { useMemo } from 'react'
import { differenceInSeconds } from 'date-fns'
import type { ComputeRequest } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDuration } from '@/lib/utils'
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

interface ComputeLatencyChartProps {
  jobs: ComputeRequest[]
  isLoading?: boolean
}

export function ComputeLatencyChart({ jobs, isLoading }: ComputeLatencyChartProps) {
  const chartData = useMemo(() => {
    return jobs
      .filter((job) => job.created_at && job.updated_at)
      .map((job) => {
        const created = new Date(job.created_at)
        const updated = new Date(job.updated_at)
        const duration = Math.max(0, differenceInSeconds(updated, created))
        return {
          requestId: job.request_id,
          timestamp: updated.getTime(),
          duration,
          status: job.status?.toLowerCase() || 'unknown',
        }
      })
      .sort((a, b) => a.timestamp - b.timestamp)
      .slice(-30)
  }, [jobs])

  const averageDuration = chartData.length
    ? chartData.reduce((sum, point) => sum + point.duration, 0) / chartData.length
    : 0

  if (isLoading && chartData.length === 0) {
    return <Skeleton className="h-[320px] w-full" />
  }

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>Execution Latency</CardTitle>
        <CardDescription>
          Average completion time {averageDuration ? formatDuration(Math.round(averageDuration)) : 'n/a'}
        </CardDescription>
      </CardHeader>
      <CardContent className="h-[280px]">
        {chartData.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">Not enough executed jobs yet.</div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#2563eb" stopOpacity={0.4} />
                  <stop offset="95%" stopColor="#2563eb" stopOpacity={0.05} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={(value) => new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                className="text-xs"
              />
              <YAxis
                tickFormatter={(value) => formatDuration(Math.max(0, Math.round(value as number)))}
                className="text-xs"
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--popover))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '8px',
                }}
                labelFormatter={(value) => new Date(value).toLocaleString()}
                formatter={(value: any, _name, payload) => {
                  return [formatDuration(Math.max(0, Math.round(value as number))), `Request ${payload?.payload?.requestId ?? ''}`]
                }}
              />
              <Area type="monotone" dataKey="duration" stroke="#2563eb" fill="url(#latencyGradient)" strokeWidth={2} name="Latency" />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
