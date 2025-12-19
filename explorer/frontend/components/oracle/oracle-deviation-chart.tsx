'use client'

import { useMemo } from 'react'
import type { OracleSubmission } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatBps, formatNumber } from '@/lib/utils'
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

interface OracleDeviationChartProps {
  submissions: OracleSubmission[]
  isLoading?: boolean
}

export function OracleDeviationChart({ submissions, isLoading }: OracleDeviationChartProps) {
  const chartData = useMemo(() => {
    const aggregates = submissions.reduce<Map<string, { asset: string; total: number; max: number; count: number }>>(
      (map, submission) => {
        const deviation = Math.abs(parseFloat(submission.deviation || '0'))
        if (isNaN(deviation)) {
          return map
        }
        const deviationBps = deviation * 10000
        const current = map.get(submission.asset) || { asset: submission.asset, total: 0, max: 0, count: 0 }
        current.total += deviationBps
        current.max = Math.max(current.max, deviationBps)
        current.count += 1
        map.set(submission.asset, current)
        return map
      },
      new Map()
    )

    return Array.from(aggregates.values())
      .map((entry) => ({
        asset: entry.asset,
        avgBps: entry.count ? entry.total / entry.count : 0,
        maxBps: entry.max,
        submissions: entry.count,
      }))
      .sort((a, b) => b.avgBps - a.avgBps)
      .slice(0, 6)
  }, [submissions])

  if (isLoading && chartData.length === 0) {
    return <Skeleton className="h-[320px] w-full" />
  }

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>Deviation Radar</CardTitle>
        <CardDescription>Average + max drift per asset (basis points).</CardDescription>
      </CardHeader>
      <CardContent className="h-[280px]">
        {chartData.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            Waiting for oracle submissions.
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis dataKey="asset" className="text-xs" />
              <YAxis className="text-xs" tickFormatter={(value) => formatNumber(Math.round(value as number))} />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--popover))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '8px',
                }}
                formatter={(value: any, name) => {
                  if (name === 'avgBps') {
                    return [formatBps(value as number), 'Average Drift']
                  }
                  if (name === 'maxBps') {
                    return [formatBps(value as number), 'Max Drift']
                  }
                  return value
                }}
                labelFormatter={(asset) => `${asset}`}
              />
              <Bar dataKey="avgBps" fill="#f59e0b" radius={[4, 4, 0, 0]} name="avgBps" />
              <Bar dataKey="maxBps" fill="#ef4444" radius={[4, 4, 0, 0]} name="maxBps" />
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
