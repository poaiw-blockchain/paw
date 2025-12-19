'use client'

import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { OraclePriceCharts } from '@/components/oracle/oracle-price-charts'
import { OracleDeviationChart } from '@/components/oracle/oracle-deviation-chart'
import { OracleValidatorPerformance } from '@/components/oracle/oracle-validator-performance'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { api, type OracleSubmission } from '@/lib/api'
import { formatBps, formatCurrency } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'

export default function OraclePage() {
  const {
    data: submissionsData,
    isLoading: loadingSubmissions,
  } = useQuery({
    queryKey: ['oracleSubmissions'],
    queryFn: () => api.getOracleSubmissions(1, 60),
    refetchInterval: 20_000,
  })

  const submissions = useMemo<OracleSubmission[]>(() => submissionsData?.data ?? [], [submissionsData])
  const recentSubmissions = useMemo<OracleSubmission[]>(() => submissions.slice(0, 12), [submissions])

  return (
    <div className="container mx-auto space-y-8 py-8">
      <div>
        <h1 className="text-4xl font-bold">Oracle Intelligence</h1>
        <p className="text-muted-foreground">
          Monitor validator proofs, deviations, and real-time pricing data.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <OracleDeviationChart submissions={submissions} isLoading={loadingSubmissions} />
        <OracleValidatorPerformance submissions={submissions} isLoading={loadingSubmissions} />
      </div>

      <OraclePriceCharts />

      <Card>
        <CardHeader>
          <CardTitle>Recent Submissions</CardTitle>
          <CardDescription>Ground-truth data feeds emitted by authorized validators.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Validator</TableHead>
                  <TableHead>Asset</TableHead>
                  <TableHead>Price</TableHead>
                  <TableHead>Deviation</TableHead>
                  <TableHead>Submitted</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentSubmissions.map((submission) => {
                  const deviation = Math.abs(parseFloat(submission.deviation || '0')) * 10000
                  const statusVariant = deviation > 50 ? 'destructive' : deviation > 25 ? 'secondary' : 'default'
                  return (
                    <TableRow key={`${submission.validator_address}-${submission.timestamp}`}>
                      <TableCell className="font-mono text-xs">{submission.validator_address}</TableCell>
                      <TableCell>{submission.asset}</TableCell>
                      <TableCell>{formatCurrency(submission.price)}</TableCell>
                      <TableCell>
                        <Badge variant={statusVariant}>{formatBps(deviation)}</Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {submission.timestamp
                          ? formatDistanceToNow(new Date(submission.timestamp), { addSuffix: true })
                          : '—'}
                      </TableCell>
                    </TableRow>
                  )
                })}
                {!loadingSubmissions && recentSubmissions.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                      No oracle submissions observed yet.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
