'use client'

import { useState, useEffect, useMemo } from 'react'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { formatDistanceToNow, format } from 'date-fns'
import {
  Box,
  ChevronLeft,
  ChevronRight,
  Clock,
  Filter,
  RefreshCw,
  Search,
  TrendingUp,
  Zap,
  User,
  Activity,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { api, Block } from '@/lib/api'
import { formatNumber, formatHash, formatGas, cn } from '@/lib/utils'

interface BlockListProps {
  variant?: 'full' | 'compact'
  limit?: number
  showPagination?: boolean
  showSearch?: boolean
  showFilters?: boolean
  autoRefresh?: boolean
  refreshInterval?: number
}

export function BlockList({
  variant = 'full',
  limit = 20,
  showPagination = true,
  showSearch = true,
  showFilters = true,
  autoRefresh = true,
  refreshInterval = 5000,
}: BlockListProps) {
  const [page, setPage] = useState(1)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<'height' | 'txs' | 'gas'>('height')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')

  // Fetch blocks with pagination
  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['blocks', page, limit, sortBy, sortOrder],
    queryFn: () => api.getBlocks(page, limit),
    refetchInterval: autoRefresh ? refreshInterval : false,
    keepPreviousData: true,
  })

  // Filter and sort blocks
  const filteredBlocks = useMemo(() => {
    if (!data?.data) return []

    let filtered = [...data.data]

    // Apply search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (block) =>
          block.height.toString().includes(query) ||
          block.hash.toLowerCase().includes(query) ||
          block.proposer_address.toLowerCase().includes(query)
      )
    }

    // Apply sorting
    filtered.sort((a, b) => {
      let comparison = 0

      switch (sortBy) {
        case 'height':
          comparison = a.height - b.height
          break
        case 'txs':
          comparison = a.tx_count - b.tx_count
          break
        case 'gas':
          comparison = a.gas_used - b.gas_used
          break
      }

      return sortOrder === 'asc' ? comparison : -comparison
    })

    return filtered
  }, [data?.data, searchQuery, sortBy, sortOrder])

  // Calculate block time statistics
  const blockTimeStats = useMemo(() => {
    if (!filteredBlocks.length) return null

    const blockTimes: number[] = []
    for (let i = 1; i < filteredBlocks.length; i++) {
      const current = new Date(filteredBlocks[i].timestamp).getTime()
      const previous = new Date(filteredBlocks[i - 1].timestamp).getTime()
      const diff = Math.abs(current - previous) / 1000 // seconds
      if (diff > 0 && diff < 60) {
        // Filter outliers
        blockTimes.push(diff)
      }
    }

    if (blockTimes.length === 0) return null

    const average = blockTimes.reduce((sum, time) => sum + time, 0) / blockTimes.length
    const min = Math.min(...blockTimes)
    const max = Math.max(...blockTimes)

    return {
      average: average.toFixed(2),
      min: min.toFixed(2),
      max: max.toFixed(2),
    }
  }, [filteredBlocks])

  // Handle pagination
  const handlePrevPage = () => setPage((prev) => Math.max(1, prev - 1))
  const handleNextPage = () => setPage((prev) => prev + 1)
  const handleRefresh = () => refetch()

  // Toggle sort
  const handleSort = (field: 'height' | 'txs' | 'gas') => {
    if (sortBy === field) {
      setSortOrder((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortBy(field)
      setSortOrder('desc')
    }
  }

  if (isError) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="text-center text-red-500">
            <p>Failed to load blocks: {error instanceof Error ? error.message : 'Unknown error'}</p>
            <Button onClick={handleRefresh} className="mt-4">
              Try Again
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (variant === 'compact') {
    return <CompactBlockList blocks={filteredBlocks} isLoading={isLoading} />
  }

  return (
    <div className="space-y-4">
      {/* Header with search and filters */}
      {(showSearch || showFilters) && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Blocks</CardTitle>
                <CardDescription>
                  {data?.total ? `${formatNumber(data.total)} total blocks` : 'Loading...'}
                </CardDescription>
              </div>
              <Button onClick={handleRefresh} variant="outline" size="sm" disabled={isLoading}>
                <RefreshCw className={cn('h-4 w-4 mr-2', isLoading && 'animate-spin')} />
                Refresh
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col md:flex-row gap-4">
              {showSearch && (
                <div className="flex-1">
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      type="text"
                      placeholder="Search by height, hash, or proposer..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="pl-10"
                    />
                  </div>
                </div>
              )}
              {showFilters && (
                <div className="flex gap-2">
                  <Select value={sortBy} onValueChange={(value: any) => setSortBy(value)}>
                    <SelectTrigger className="w-[150px]">
                      <SelectValue placeholder="Sort by" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="height">Height</SelectItem>
                      <SelectItem value="txs">Transactions</SelectItem>
                      <SelectItem value="gas">Gas Used</SelectItem>
                    </SelectContent>
                  </Select>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => setSortOrder((prev) => (prev === 'asc' ? 'desc' : 'asc'))}
                  >
                    <TrendingUp
                      className={cn('h-4 w-4 transition-transform', sortOrder === 'asc' && 'rotate-180')}
                    />
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Block time statistics */}
      {blockTimeStats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Average Block Time</p>
                  <p className="text-2xl font-bold">{blockTimeStats.average}s</p>
                </div>
                <Clock className="h-8 w-8 text-muted-foreground" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Min Block Time</p>
                  <p className="text-2xl font-bold">{blockTimeStats.min}s</p>
                </div>
                <Zap className="h-8 w-8 text-green-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Max Block Time</p>
                  <p className="text-2xl font-bold">{blockTimeStats.max}s</p>
                </div>
                <Activity className="h-8 w-8 text-orange-500" />
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Blocks table */}
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[100px]">
                    <button
                      onClick={() => handleSort('height')}
                      className="flex items-center gap-1 hover:text-foreground"
                    >
                      Height
                      {sortBy === 'height' && (
                        <TrendingUp
                          className={cn('h-3 w-3', sortOrder === 'asc' && 'rotate-180')}
                        />
                      )}
                    </button>
                  </TableHead>
                  <TableHead>Hash</TableHead>
                  <TableHead>Proposer</TableHead>
                  <TableHead>
                    <button
                      onClick={() => handleSort('txs')}
                      className="flex items-center gap-1 hover:text-foreground"
                    >
                      Transactions
                      {sortBy === 'txs' && (
                        <TrendingUp
                          className={cn('h-3 w-3', sortOrder === 'asc' && 'rotate-180')}
                        />
                      )}
                    </button>
                  </TableHead>
                  <TableHead>
                    <button
                      onClick={() => handleSort('gas')}
                      className="flex items-center gap-1 hover:text-foreground"
                    >
                      Gas Used
                      {sortBy === 'gas' && (
                        <TrendingUp
                          className={cn('h-3 w-3', sortOrder === 'asc' && 'rotate-180')}
                        />
                      )}
                    </button>
                  </TableHead>
                  <TableHead className="text-right">Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <>
                    {[...Array(limit)].map((_, i) => (
                      <TableRow key={i}>
                        <TableCell>
                          <Skeleton className="h-4 w-16" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-32" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-32" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-12" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-24" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-20" />
                        </TableCell>
                      </TableRow>
                    ))}
                  </>
                ) : filteredBlocks.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                      No blocks found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredBlocks.map((block) => (
                    <TableRow key={block.height} className="hover:bg-muted/50">
                      <TableCell>
                        <Link
                          href={`/blocks/${block.height}`}
                          className="font-mono font-semibold hover:text-primary"
                        >
                          #{formatNumber(block.height)}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Link
                                href={`/blocks/${block.height}`}
                                className="font-mono text-sm hover:text-primary"
                              >
                                {formatHash(block.hash)}
                              </Link>
                            </TooltipTrigger>
                            <TooltipContent>
                              <p className="font-mono text-xs">{block.hash}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </TableCell>
                      <TableCell>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Link
                                href={`/validators/${block.proposer_address}`}
                                className="font-mono text-sm hover:text-primary flex items-center gap-1"
                              >
                                <User className="h-3 w-3" />
                                {formatHash(block.proposer_address)}
                              </Link>
                            </TooltipTrigger>
                            <TooltipContent>
                              <p className="font-mono text-xs">{block.proposer_address}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </TableCell>
                      <TableCell>
                        <Badge variant={block.tx_count > 0 ? 'default' : 'secondary'}>
                          {formatNumber(block.tx_count)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{formatGas(block.gas_used)}</span>
                      </TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="text-sm text-muted-foreground cursor-help">
                                {formatDistanceToNow(new Date(block.timestamp), { addSuffix: true })}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent>
                              <p className="text-xs">{format(new Date(block.timestamp), 'PPpp')}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
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
      {showPagination && data && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Showing {(page - 1) * limit + 1} to {Math.min(page * limit, data.total)} of{' '}
            {formatNumber(data.total)} blocks
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handlePrevPage} disabled={page === 1 || isLoading}>
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
              onClick={handleNextPage}
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

// Compact variant for embedding in other views
function CompactBlockList({ blocks, isLoading }: { blocks: Block[]; isLoading: boolean }) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} className="h-16 w-full" />
        ))}
      </div>
    )
  }

  if (blocks.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <Box className="h-12 w-12 mx-auto mb-2 opacity-50" />
        <p>No blocks found</p>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {blocks.map((block) => (
        <Link key={block.height} href={`/blocks/${block.height}`}>
          <Card className="hover:shadow-md transition-shadow cursor-pointer">
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <Box className="h-8 w-8 text-primary" />
                  <div>
                    <p className="font-semibold">Block #{formatNumber(block.height)}</p>
                    <p className="text-sm text-muted-foreground font-mono">{formatHash(block.hash)}</p>
                  </div>
                </div>
                <div className="text-right">
                  <Badge variant={block.tx_count > 0 ? 'default' : 'secondary'}>
                    {formatNumber(block.tx_count)} txs
                  </Badge>
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatDistanceToNow(new Date(block.timestamp), { addSuffix: true })}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  )
}

export default BlockList
