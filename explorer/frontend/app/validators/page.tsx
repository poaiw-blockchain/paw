'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import {
  Shield,
  Search,
  ArrowUpDown,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Filter,
  ExternalLink,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api, ValidatorDetail } from '@/lib/api'
import { formatNumber, formatPercent, formatToken } from '@/lib/utils'

const statusFilters = [
  { value: 'all', label: 'All Validators' },
  { value: 'bonded', label: 'Active (Bonded)' },
  { value: 'unbonding', label: 'Unbonding' },
  { value: 'unbonded', label: 'Inactive (Unbonded)' },
]

const sortOptions = [
  { value: 'voting_power', label: 'Voting Power' },
  { value: 'commission', label: 'Commission' },
  { value: 'moniker', label: 'Name' },
]

function getStatusBadgeVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status.toLowerCase()) {
    case 'active':
      return 'default'
    case 'unbonding':
      return 'secondary'
    case 'inactive':
      return 'destructive'
    default:
      return 'outline'
  }
}

function getStatusIcon(status: string, jailed: boolean) {
  if (jailed) {
    return <AlertTriangle className="h-4 w-4 text-orange-500" />
  }
  switch (status.toLowerCase()) {
    case 'active':
      return <CheckCircle className="h-4 w-4 text-green-500" />
    case 'inactive':
      return <XCircle className="h-4 w-4 text-red-500" />
    default:
      return <AlertTriangle className="h-4 w-4 text-yellow-500" />
  }
}

export default function ValidatorsPage() {
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<string>('voting_power')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [searchQuery, setSearchQuery] = useState('')

  const { data, isLoading, isError } = useQuery({
    queryKey: ['validators', statusFilter, sortBy, sortOrder],
    queryFn: () =>
      api.getValidatorsList({
        status: statusFilter === 'all' ? undefined : statusFilter,
        sort: sortBy,
        order: sortOrder,
      }),
    refetchInterval: 60000,
  })

  const validators = data?.validators || []

  // Filter by search query
  const filteredValidators = validators.filter((v) =>
    v.moniker.toLowerCase().includes(searchQuery.toLowerCase()) ||
    v.operator_address.toLowerCase().includes(searchQuery.toLowerCase())
  )

  // Calculate totals
  const totalVotingPower = validators.reduce((sum, v) => sum + v.voting_power, 0)
  const activeCount = validators.filter((v) => v.status_label === 'Active').length
  const jailedCount = validators.filter((v) => v.jailed).length

  const toggleSortOrder = () => {
    setSortOrder((prev) => (prev === 'asc' ? 'desc' : 'asc'))
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between flex-wrap gap-4">
            <div>
              <CardTitle className="text-3xl flex items-center gap-2">
                <Shield className="h-8 w-8 text-primary" />
                Validators
              </CardTitle>
              <CardDescription>
                Network validators securing the PAW Chain
              </CardDescription>
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search validators..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-40">
                  <Filter className="h-4 w-4 mr-2" />
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  {statusFilters.map((filter) => (
                    <SelectItem key={filter.value} value={filter.value}>
                      {filter.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={sortBy} onValueChange={setSortBy}>
                <SelectTrigger className="w-40">
                  <SelectValue placeholder="Sort by" />
                </SelectTrigger>
                <SelectContent>
                  {sortOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button variant="outline" size="icon" onClick={toggleSortOrder}>
                <ArrowUpDown className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Stats Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Validators</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{formatNumber(validators.length)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Active Validators</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-green-500">{formatNumber(activeCount)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Voting Power</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{formatToken(totalVotingPower.toString(), 6)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Jailed</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-orange-500">{formatNumber(jailedCount)}</p>
          </CardContent>
        </Card>
      </div>

      {/* Validators Table */}
      <Card>
        <CardHeader>
          <CardTitle>Validator Set</CardTitle>
          <CardDescription>
            {filteredValidators.length} validator{filteredValidators.length !== 1 ? 's' : ''} found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[...Array(10)].map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : isError ? (
            <div className="text-center py-8 text-muted-foreground">
              <AlertTriangle className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>Failed to load validators</p>
            </div>
          ) : filteredValidators.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">Rank</TableHead>
                    <TableHead>Validator</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Voting Power</TableHead>
                    <TableHead className="text-right">Share</TableHead>
                    <TableHead className="text-right">Commission</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredValidators.map((validator: ValidatorDetail) => {
                    const powerShare = totalVotingPower > 0
                      ? (validator.voting_power / totalVotingPower) * 100
                      : 0

                    return (
                      <TableRow key={validator.operator_address} className="hover:bg-muted/50">
                        <TableCell className="font-mono font-bold">
                          #{validator.rank}
                        </TableCell>
                        <TableCell>
                          <Link href={`/validators/${validator.operator_address}`}>
                            <div className="flex items-center gap-3">
                              <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                                <span className="text-xs font-bold">
                                  {validator.moniker.slice(0, 2).toUpperCase()}
                                </span>
                              </div>
                              <div>
                                <Button variant="link" className="h-auto p-0 font-medium">
                                  {validator.moniker}
                                </Button>
                                {validator.website && (
                                  <a
                                    href={validator.website}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="block text-xs text-muted-foreground hover:text-primary"
                                    onClick={(e) => e.stopPropagation()}
                                  >
                                    <ExternalLink className="h-3 w-3 inline mr-1" />
                                    Website
                                  </a>
                                )}
                              </div>
                            </div>
                          </Link>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getStatusIcon(validator.status_label, validator.jailed)}
                            <Badge variant={getStatusBadgeVariant(validator.status_label)}>
                              {validator.jailed ? 'Jailed' : validator.status_label}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatToken(validator.voting_power.toString(), 6)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <div className="w-16 h-2 bg-muted rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary transition-all"
                                style={{ width: `${Math.min(powerShare, 100)}%` }}
                              />
                            </div>
                            <span className="text-sm">{powerShare.toFixed(2)}%</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <span className={validator.commission_rate > 0.1 ? 'text-orange-500' : ''}>
                            {formatPercent(validator.commission_rate * 100, 2)}
                          </span>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <Shield className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No validators found</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
