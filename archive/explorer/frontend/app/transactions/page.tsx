'use client'

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow } from 'date-fns'
import { CheckCircle, XCircle, ChevronLeft, ChevronRight, RefreshCw, Search } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api, Transaction } from '@/lib/api'
import { formatNumber, formatHash, formatGas, formatToken, cn } from '@/lib/utils'

export default function TransactionsPage() {
  const [page, setPage] = useState(1)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const limit = 20

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['transactions', page, limit, statusFilter],
    queryFn: () => api.getTransactions(page, limit, statusFilter === 'all' ? undefined : statusFilter),
    keepPreviousData: true,
    refetchInterval: 5000,
  })

  const filteredTransactions = useMemo(() => {
    if (!data?.data) return []

    let filtered = [...data.data]

    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (tx) =>
          tx.hash.toLowerCase().includes(query) ||
          tx.sender.toLowerCase().includes(query) ||
          tx.type.toLowerCase().includes(query)
      )
    }

    return filtered
  }, [data?.data, searchQuery])

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-3xl">All Transactions</CardTitle>
              <CardDescription>
                {data?.total ? `${formatNumber(data.total)} total transactions` : 'Loading...'}
              </CardDescription>
            </div>
            <Button onClick={() => refetch()} variant="outline" size="sm" disabled={isLoading}>
              <RefreshCw className={cn('h-4 w-4 mr-2', isLoading && 'animate-spin')} />
              Refresh
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  type="text"
                  placeholder="Search by hash, sender, or type..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[150px]">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="success">Success</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Transactions Table */}
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Hash</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Block</TableHead>
                  <TableHead>From</TableHead>
                  <TableHead>Gas Used</TableHead>
                  <TableHead>Fee</TableHead>
                  <TableHead className="text-right">Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <>
                    {[...Array(limit)].map((_, i) => (
                      <TableRow key={i}>
                        {[...Array(8)].map((_, j) => (
                          <TableCell key={j}>
                            <Skeleton className="h-4 w-20" />
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </>
                ) : filteredTransactions.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                      No transactions found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredTransactions.map((tx: Transaction) => (
                    <TableRow key={tx.hash} className="hover:bg-muted/50">
                      <TableCell>
                        {tx.status === 'success' ? (
                          <CheckCircle className="h-5 w-5 text-green-500" />
                        ) : (
                          <XCircle className="h-5 w-5 text-red-500" />
                        )}
                      </TableCell>
                      <TableCell>
                        <Link href={`/tx/${tx.hash}`}>
                          <Button variant="link" className="h-auto p-0 font-mono text-sm">
                            {formatHash(tx.hash)}
                          </Button>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {tx.type}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Link href={`/block/${tx.block_height}`}>
                          <Button variant="link" className="h-auto p-0">
                            #{formatNumber(tx.block_height)}
                          </Button>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Link href={`/account/${tx.sender}`}>
                          <Button variant="link" className="h-auto p-0 font-mono text-xs">
                            {formatHash(tx.sender, 8, 6)}
                          </Button>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{formatGas(tx.gas_used)}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm font-mono">{formatToken(tx.fee_amount, 6)}</span>
                      </TableCell>
                      <TableCell className="text-right">
                        <span className="text-sm text-muted-foreground">
                          {formatDistanceToNow(new Date(tx.timestamp), { addSuffix: true })}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Pagination */}
      {data && data.total > limit && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Showing {(page - 1) * limit + 1} to {Math.min(page * limit, data.total)} of{' '}
            {formatNumber(data.total)} transactions
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1 || isLoading}
            >
              <ChevronLeft className="h-4 w-4 mr-1" />
              Previous
            </Button>
            <div className="flex items-center gap-1">
              <span className="text-sm">Page</span>
              <Badge variant="outline">{page}</Badge>
              <span className="text-sm">of {Math.ceil(data.total / limit)}</span>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => p + 1)}
              disabled={page >= Math.ceil(data.total / limit) || isLoading}
            >
              Next
              <ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
