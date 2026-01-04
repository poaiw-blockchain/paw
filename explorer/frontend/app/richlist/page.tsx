'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import {
  Trophy,
  Search,
  ChevronLeft,
  ChevronRight,
  Coins,
  TrendingUp,
  Users,
  RefreshCw,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api, RichListEntry } from '@/lib/api'
import { formatNumber, formatAddress, formatToken, formatPercent } from '@/lib/utils'

const ITEMS_PER_PAGE = 100

export default function RichListPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['richlist'],
    queryFn: () => api.getRichList(500, 'upaw'),
    staleTime: 600000, // 10 minutes
  })

  const richlist = data?.richlist || []
  const totalSupply = data?.total_supply || '0'
  const lastUpdated = data?.last_updated

  // Filter by search
  const filteredList = searchQuery
    ? richlist.filter((entry) =>
        entry.address.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : richlist

  // Paginate
  const totalPages = Math.ceil(filteredList.length / ITEMS_PER_PAGE)
  const paginatedList = filteredList.slice(
    (page - 1) * ITEMS_PER_PAGE,
    page * ITEMS_PER_PAGE
  )

  // Calculate stats
  const top10Balance = richlist.slice(0, 10).reduce((sum, entry) => sum + BigInt(entry.balance), BigInt(0))
  const top10Percentage = richlist.slice(0, 10).reduce((sum, entry) => sum + entry.percentage, 0)
  const top100Balance = richlist.slice(0, 100).reduce((sum, entry) => sum + BigInt(entry.balance), BigInt(0))
  const top100Percentage = richlist.slice(0, 100).reduce((sum, entry) => sum + entry.percentage, 0)

  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(e.target.value)
    setPage(1)
  }

  if (isLoading) {
    return <RichListSkeleton />
  }

  if (isError) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="p-12 text-center">
            <Trophy className="h-16 w-16 mx-auto mb-4 text-muted-foreground" />
            <h2 className="text-2xl font-bold mb-2">Failed to load Rich List</h2>
            <p className="text-muted-foreground mb-6">
              Unable to fetch token holder data.
            </p>
            <Button onClick={() => refetch()}>
              <RefreshCw className="mr-2 h-4 w-4" />
              Try Again
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-3xl flex items-center gap-3">
                <Trophy className="h-8 w-8 text-yellow-500" />
                Rich List
              </CardTitle>
              <CardDescription className="mt-2">
                Top PAW token holders by balance
              </CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>
        </CardHeader>
      </Card>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Supply</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Coins className="h-5 w-5 text-primary" />
              <p className="text-xl font-bold">{formatToken(totalSupply, 6, 'PAW')}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Holders</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Users className="h-5 w-5 text-primary" />
              <p className="text-xl font-bold">{formatNumber(richlist.length)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Top 10 Concentration</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-yellow-500" />
              <p className="text-xl font-bold">{formatPercent(top10Percentage)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Top 100 Concentration</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-orange-500" />
              <p className="text-xl font-bold">{formatPercent(top100Percentage)}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search and Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Token Holders</CardTitle>
            {lastUpdated && (
              <span className="text-sm text-muted-foreground">
                Updated: {new Date(lastUpdated).toLocaleString()}
              </span>
            )}
          </div>
          {/* Search */}
          <div className="relative mt-4">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Search by address..."
              value={searchQuery}
              onChange={handleSearch}
              className="pl-10"
            />
          </div>
        </CardHeader>
        <CardContent>
          {paginatedList.length > 0 ? (
            <>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-20">Rank</TableHead>
                      <TableHead>Address</TableHead>
                      <TableHead className="text-right">Balance</TableHead>
                      <TableHead className="text-right">% of Supply</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {paginatedList.map((entry: RichListEntry) => (
                      <TableRow key={entry.address} className="hover:bg-muted/50">
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {entry.rank <= 3 ? (
                              <Badge
                                variant={
                                  entry.rank === 1
                                    ? 'default'
                                    : entry.rank === 2
                                    ? 'secondary'
                                    : 'outline'
                                }
                                className={
                                  entry.rank === 1
                                    ? 'bg-yellow-500 hover:bg-yellow-600'
                                    : entry.rank === 2
                                    ? 'bg-gray-400 hover:bg-gray-500'
                                    : 'bg-amber-700 hover:bg-amber-800 text-white'
                                }
                              >
                                #{entry.rank}
                              </Badge>
                            ) : (
                              <span className="text-muted-foreground">#{entry.rank}</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Link href={`/account/${entry.address}`}>
                            <Button variant="link" className="h-auto p-0 font-mono text-sm">
                              {formatAddress(entry.address, 12, 10)}
                            </Button>
                          </Link>
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatToken(entry.balance, 6)} PAW
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <div className="w-16 h-2 bg-muted rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary rounded-full"
                                style={{ width: `${Math.min(entry.percentage, 100)}%` }}
                              />
                            </div>
                            <span className="text-sm">{formatPercent(entry.percentage)}</span>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between pt-4 border-t mt-4">
                  <div className="text-sm text-muted-foreground">
                    Showing {(page - 1) * ITEMS_PER_PAGE + 1} to{' '}
                    {Math.min(page * ITEMS_PER_PAGE, filteredList.length)} of{' '}
                    {formatNumber(filteredList.length)} holders
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      disabled={page === 1}
                    >
                      <ChevronLeft className="h-4 w-4 mr-1" />
                      Previous
                    </Button>
                    <div className="flex items-center gap-1">
                      <span className="text-sm">Page</span>
                      <Badge variant="outline">{page}</Badge>
                      <span className="text-sm">of {totalPages}</span>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                      disabled={page >= totalPages}
                    >
                      Next
                      <ChevronRight className="h-4 w-4 ml-1" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-12 text-muted-foreground">
              <Search className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No holders found matching your search</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function RichListSkeleton() {
  return (
    <div className="container mx-auto py-8 space-y-6">
      <Skeleton className="h-24 w-full" />
      <div className="grid gap-4 md:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
      <Skeleton className="h-96 w-full" />
    </div>
  )
}
