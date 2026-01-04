'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow, format } from 'date-fns'
import {
  Vote,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  FileText,
  Filter,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api, Proposal } from '@/lib/api'
import { formatNumber } from '@/lib/utils'

const statusFilters = [
  { value: 'all', label: 'All Proposals' },
  { value: 'voting', label: 'Voting Period' },
  { value: 'passed', label: 'Passed' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'deposit', label: 'Deposit Period' },
]

function getStatusBadgeVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status.toLowerCase()) {
    case 'voting':
      return 'default'
    case 'passed':
      return 'secondary'
    case 'rejected':
    case 'failed':
      return 'destructive'
    default:
      return 'outline'
  }
}

function getStatusIcon(status: string) {
  switch (status.toLowerCase()) {
    case 'voting':
      return <Vote className="h-4 w-4" />
    case 'passed':
      return <CheckCircle className="h-4 w-4 text-green-500" />
    case 'rejected':
    case 'failed':
      return <XCircle className="h-4 w-4 text-red-500" />
    case 'deposit':
      return <Clock className="h-4 w-4 text-yellow-500" />
    default:
      return <AlertCircle className="h-4 w-4" />
  }
}

export default function GovernancePage() {
  const [statusFilter, setStatusFilter] = useState<string>('all')

  const { data, isLoading, isError } = useQuery({
    queryKey: ['proposals', statusFilter],
    queryFn: () => api.getProposals(statusFilter === 'all' ? undefined : statusFilter),
    refetchInterval: 60000,
  })

  const proposals = data?.proposals || []

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-3xl flex items-center gap-2">
                <Vote className="h-8 w-8 text-primary" />
                Governance
              </CardTitle>
              <CardDescription>
                View and track governance proposals on PAW Chain
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="Filter by status" />
                </SelectTrigger>
                <SelectContent>
                  {statusFilters.map((filter) => (
                    <SelectItem key={filter.value} value={filter.value}>
                      {filter.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Stats Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Proposals</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{formatNumber(proposals.length)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Active Voting</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-primary">
              {formatNumber(proposals.filter((p) => p.status_label === 'Voting').length)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Passed</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-green-500">
              {formatNumber(proposals.filter((p) => p.status_label === 'Passed').length)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Rejected</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-red-500">
              {formatNumber(proposals.filter((p) => p.status_label === 'Rejected').length)}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Proposals Table */}
      <Card>
        <CardHeader>
          <CardTitle>Proposals</CardTitle>
          <CardDescription>
            {proposals.length} proposal{proposals.length !== 1 ? 's' : ''} found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : isError ? (
            <div className="text-center py-8 text-muted-foreground">
              <AlertCircle className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>Failed to load proposals</p>
            </div>
          ) : proposals.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">ID</TableHead>
                    <TableHead>Title</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Submit Time</TableHead>
                    <TableHead>Voting End</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {proposals.map((proposal: Proposal) => (
                    <TableRow key={proposal.proposal_id} className="hover:bg-muted/50">
                      <TableCell className="font-mono">
                        #{proposal.proposal_id}
                      </TableCell>
                      <TableCell>
                        <Link href={`/governance/${proposal.proposal_id}`}>
                          <Button variant="link" className="h-auto p-0 text-left font-medium">
                            {proposal.content?.title || 'Untitled Proposal'}
                          </Button>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={getStatusBadgeVariant(proposal.status_label)}
                          className="flex items-center gap-1 w-fit"
                        >
                          {getStatusIcon(proposal.status_label)}
                          {proposal.status_label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {proposal.content?.['@type']?.split('.').pop() || 'Unknown'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-muted-foreground">
                          {proposal.submit_time
                            ? formatDistanceToNow(new Date(proposal.submit_time), { addSuffix: true })
                            : 'N/A'}
                        </span>
                      </TableCell>
                      <TableCell>
                        {proposal.voting_end_time ? (
                          <div className="text-sm">
                            <span className="text-muted-foreground">
                              {format(new Date(proposal.voting_end_time), 'MMM d, yyyy')}
                            </span>
                            <br />
                            <span className="text-xs text-muted-foreground">
                              {formatDistanceToNow(new Date(proposal.voting_end_time), { addSuffix: true })}
                            </span>
                          </div>
                        ) : (
                          <span className="text-sm text-muted-foreground">N/A</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No proposals found</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
