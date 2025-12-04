'use client'

import { useQuery } from '@tanstack/react-query'
import { OraclePriceCharts } from '@/components/oracle/oracle-price-charts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import { formatCurrency, formatNumber } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'

export default function OraclePage() {
  const { data: submissionsData } = useQuery({
    queryKey: ['oracleSubmissions'],
    queryFn: () => api.getOracleSubmissions(1, 12),
    refetchInterval: 20_000,
  })

  const submissions = submissionsData?.data ?? []

  return (
    <div className="container mx-auto space-y-8 py-8">
      <div>
        <h1 className="text-4xl font-bold">Oracle Intelligence</h1>
        <p className="text-muted-foreground">
          Monitor validator proofs, deviations, and real-time pricing data.
        </p>
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
                {submissions.map((submission) => (
                  <TableRow key={`${submission.validator_address}-${submission.timestamp}`}>
                    <TableCell className="font-mono text-xs">{submission.validator_address}</TableCell>
                    <TableCell>{submission.asset}</TableCell>
                    <TableCell>{formatCurrency(submission.price)}</TableCell>
                    <TableCell>
                      <Badge variant={Math.abs(parseFloat(submission.deviation || '0')) > 0.02 ? 'destructive' : 'secondary'}>
                        {formatNumber(parseFloat(submission.deviation || '0') * 100)} bps
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {submission.timestamp
                        ? formatDistanceToNow(new Date(submission.timestamp), { addSuffix: true })
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
