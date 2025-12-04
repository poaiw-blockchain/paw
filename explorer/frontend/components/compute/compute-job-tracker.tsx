'use client'

import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { formatHash, formatNumber } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'

const statusFilters = [
  { label: 'Active', value: 'active' },
  { label: 'Pending', value: 'pending' },
  { label: 'Completed', value: 'completed' },
  { label: 'Failed', value: 'failed' },
]

const statusVariant: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  active: 'default',
  pending: 'secondary',
  completed: 'default',
  failed: 'destructive',
  default: 'secondary',
}

export function ComputeJobTracker() {
  const [status, setStatus] = useState<string>('active')

  const {
    data,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['computeJobs', status],
    queryFn: () => api.getComputeRequests(1, 20, status === 'active' ? 'active' : status),
    refetchInterval: 15_000,
  })

  const jobs = data?.data ?? []

  const summary = useMemo(() => {
    return statusFilters.map((filter) => ({
      ...filter,
      count: jobs.filter((job) => job.status?.toLowerCase() === filter.value).length,
    }))
  }, [jobs])

  if (isLoading) {
    return <Skeleton className="h-[420px] w-full" />
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Compute Job Tracker</CardTitle>
          <CardDescription>Cannot connect to the compute indexer.</CardDescription>
        </CardHeader>
        <CardContent>
          <button className="text-sm text-primary underline" onClick={() => refetch()}>
            Retry
          </button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle>Decentralized Compute Pipeline</CardTitle>
          <CardDescription>Monitor escrowed workloads and provider attestations in real-time.</CardDescription>
        </div>
        <Tabs value={status} onValueChange={setStatus}>
          <TabsList>
            {statusFilters.map((filter) => (
              <TabsTrigger key={filter.value} value={filter.value}>
                {filter.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 md:grid-cols-4">
          {summary.map((item) => (
            <Card key={item.value}>
              <CardHeader className="pb-2">
                <CardDescription>{item.label}</CardDescription>
                <CardTitle className="text-2xl">{formatNumber(item.count)}</CardTitle>
              </CardHeader>
            </Card>
          ))}
        </div>

        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Request</TableHead>
                <TableHead>Requester</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Reward</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Updated</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {jobs.map((job) => (
                <TableRow key={job.request_id}>
                  <TableCell>
                    <div>
                      <p className="font-mono text-sm">{formatHash(job.request_id)}</p>
                      <p className="text-xs text-muted-foreground">Program {formatHash(job.program_hash)}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <p className="font-mono text-xs">{formatHash(job.requester)}</p>
                  </TableCell>
                  <TableCell>
                    {job.provider ? <p className="font-mono text-xs">{formatHash(job.provider)}</p> : <span className="text-muted-foreground text-xs">Unassigned</span>}
                  </TableCell>
                  <TableCell>{job.reward} upaw</TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[job.status?.toLowerCase() || 'default'] || 'secondary'}>
                      {job.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <p className="text-xs text-muted-foreground">
                      {job.updated_at ? formatDistanceToNow(new Date(job.updated_at), { addSuffix: true }) : '—'}
                    </p>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
