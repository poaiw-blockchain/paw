'use client'

import { useMemo } from 'react'
import type { OracleSubmission } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { formatBps, formatNumber, formatHash } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'
import { Badge } from '@/components/ui/badge'

interface OracleValidatorPerformanceProps {
  submissions: OracleSubmission[]
  isLoading?: boolean
}

export function OracleValidatorPerformance({ submissions, isLoading }: OracleValidatorPerformanceProps) {
  const validatorStats = useMemo(() => {
    const aggregates = submissions.reduce<
      Map<string, { validator: string; totalDeviation: number; count: number; assets: Set<string>; lastTimestamp?: Date }>
    >((map, submission) => {
      const deviation = Math.abs(parseFloat(submission.deviation || '0'))
      if (isNaN(deviation)) {
        return map
      }

      const record =
        map.get(submission.validator_address) ||
        { validator: submission.validator_address, totalDeviation: 0, count: 0, assets: new Set<string>(), lastTimestamp: undefined }

      record.totalDeviation += deviation * 10000
      record.count += 1
      if (submission.asset) {
        record.assets.add(submission.asset)
      }
      if (submission.timestamp) {
        const ts = new Date(submission.timestamp)
        if (!record.lastTimestamp || ts > record.lastTimestamp) {
          record.lastTimestamp = ts
        }
      }

      map.set(submission.validator_address, record)
      return map
    }, new Map())

    return Array.from(aggregates.values())
      .map((entry) => ({
        validator: entry.validator,
        avgDeviation: entry.count ? entry.totalDeviation / entry.count : 0,
        submissionCount: entry.count,
        assetCount: entry.assets.size,
        lastTimestamp: entry.lastTimestamp,
      }))
      .sort((a, b) => b.submissionCount - a.submissionCount || a.avgDeviation - b.avgDeviation)
      .slice(0, 6)
  }, [submissions])

  if (isLoading && validatorStats.length === 0) {
    return <Skeleton className="h-[320px] w-full" />
  }

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>Validator Reliability</CardTitle>
        <CardDescription>Submission cadence and drift for top oracle validators.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {validatorStats.length === 0 ? (
          <p className="text-sm text-muted-foreground">No validator submissions captured yet.</p>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Validator</TableHead>
                  <TableHead>Avg Drift</TableHead>
                  <TableHead>Submissions</TableHead>
                  <TableHead>Assets</TableHead>
                  <TableHead>Last Update</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {validatorStats.map((validator) => (
                  <TableRow key={validator.validator}>
                    <TableCell>
                      <div className="space-y-1">
                        <p className="font-mono text-xs">{formatHash(validator.validator)}</p>
                        <Badge variant={validator.avgDeviation < 25 ? 'default' : 'secondary'} className="text-xs">
                          {validator.avgDeviation < 25 ? 'Stable' : 'Watch'}
                        </Badge>
                      </div>
                    </TableCell>
                    <TableCell>{formatBps(validator.avgDeviation)}</TableCell>
                    <TableCell>{formatNumber(validator.submissionCount)}</TableCell>
                    <TableCell>{formatNumber(validator.assetCount)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {validator.lastTimestamp ? formatDistanceToNow(validator.lastTimestamp, { addSuffix: true }) : '—'}
                    </TableCell>
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
