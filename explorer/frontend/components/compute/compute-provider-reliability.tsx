'use client'

import { useMemo } from 'react'
import type { ComputeProvider, ComputeRequest } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { formatHash, formatNumber, formatPercent } from '@/lib/utils'

interface ComputeProviderReliabilityProps {
  providers: ComputeProvider[]
  jobs: ComputeRequest[]
  isLoading?: boolean
}

export function ComputeProviderReliability({ providers, jobs, isLoading }: ComputeProviderReliabilityProps) {
  const providerStats = useMemo(() => {
    const assignments = jobs.reduce<Map<string, number>>((map, job) => {
      if (job.provider) {
        map.set(job.provider, (map.get(job.provider) || 0) + 1)
      }
      return map
    }, new Map())

    const totalAssignments = Array.from(assignments.values()).reduce((sum, value) => sum + value, 0)

    return providers
      .map((provider) => {
        const completed = Number(provider.completed_jobs || 0)
        const failed = Number(provider.failed_jobs || 0)
        const total = completed + failed || Number(provider.total_jobs || completed)
        const reliability = total > 0 ? (completed / total) * 100 : 0
        const queueShare = totalAssignments > 0 ? ((assignments.get(provider.address) || 0) / totalAssignments) * 100 : 0

        return {
          address: provider.address,
          reliability,
          completed: completed,
          failures: failed,
          slashCount: Number(provider.slash_count || 0),
          queueShare,
          reputation: provider.reputation ?? 0,
        }
      })
      .sort((a, b) => b.reliability - a.reliability || b.reputation - a.reputation)
      .slice(0, 6)
  }, [providers, jobs])

  if (isLoading && providerStats.length === 0) {
    return <Skeleton className="h-[320px] w-full" />
  }

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>Provider Reliability</CardTitle>
        <CardDescription>Top performers ranked by completion rate and queue share.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {providerStats.length === 0 ? (
          <p className="text-sm text-muted-foreground">Provider registry is empty.</p>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Provider</TableHead>
                  <TableHead>Reliability</TableHead>
                  <TableHead>Completed</TableHead>
                  <TableHead>Slash</TableHead>
                  <TableHead>Queue Share</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {providerStats.map((provider) => (
                  <TableRow key={provider.address}>
                    <TableCell>
                      <div className="space-y-1">
                        <p className="font-mono text-xs">{formatHash(provider.address)}</p>
                        <p className="text-xs text-muted-foreground">Rep: {provider.reputation?.toFixed(2) ?? '0.00'}</p>
                      </div>
                    </TableCell>
                    <TableCell>{formatPercent(provider.reliability, 1)}</TableCell>
                    <TableCell>{formatNumber(provider.completed)}</TableCell>
                    <TableCell>{formatNumber(provider.slashCount)}</TableCell>
                    <TableCell>{formatPercent(provider.queueShare, 1)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
