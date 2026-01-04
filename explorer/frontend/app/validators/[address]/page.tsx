'use client'

import { use } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import {
  ArrowLeft,
  Shield,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Copy,
  ExternalLink,
  User,
  Percent,
  Coins,
  Users,
  Globe,
  Mail,
  Clock,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api, ValidatorDetail, Delegation } from '@/lib/api'
import { formatNumber, formatHash, formatPercent, formatToken } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface PageProps {
  params: Promise<{ address: string }>
}

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

export default function ValidatorDetailPage({ params }: PageProps) {
  const { address } = use(params)
  const { copy, isCopied } = useCopyToClipboard()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['validator', address],
    queryFn: () => api.getValidatorDetail(address),
    retry: 1,
  })

  if (isLoading) {
    return <ValidatorSkeleton />
  }

  if (isError || !data?.validator) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="p-12 text-center">
            <XCircle className="h-16 w-16 mx-auto mb-4 text-destructive" />
            <h2 className="text-2xl font-bold mb-2">Validator Not Found</h2>
            <p className="text-muted-foreground mb-6">
              {error instanceof Error ? error.message : 'The validator you are looking for does not exist.'}
            </p>
            <Link href="/validators">
              <Button>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Validators
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  const validator = data.validator
  const delegations = data.delegations || []
  const delegatorCount = data.delegator_count || 0
  const commissionEarned = data.commission_earned

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/validators">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <span className="text-2xl font-bold">
                {validator.moniker.slice(0, 2).toUpperCase()}
              </span>
            </div>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-3xl font-bold">{validator.moniker}</h1>
                <Badge variant={getStatusBadgeVariant(validator.status_label)}>
                  {validator.jailed ? 'Jailed' : validator.status_label}
                </Badge>
              </div>
              <div className="flex items-center gap-2 mt-1">
                <code className="text-sm font-mono text-muted-foreground">
                  {formatHash(validator.operator_address, 16, 12)}
                </code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => copy(validator.operator_address)}
                >
                  {isCopied ? (
                    <CheckCircle className="h-3 w-3 text-green-500" />
                  ) : (
                    <Copy className="h-3 w-3" />
                  )}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Overview Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Voting Power</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">
                {formatToken(validator.voting_power.toString(), 6)}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Commission</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Percent className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">
                {formatPercent(validator.commission_rate * 100, 2)}
              </p>
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Max: {formatPercent(validator.commission_max_rate * 100, 0)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Delegators</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Users className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">{formatNumber(delegatorCount)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Self Delegation</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Coins className="h-5 w-5 text-primary" />
              <p className="text-lg font-bold">
                {formatToken(validator.min_self_delegation || '0', 6)}
              </p>
            </div>
            <p className="text-xs text-muted-foreground">Min required</p>
          </CardContent>
        </Card>
      </div>

      {/* Validator Info */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Details Card */}
        <Card>
          <CardHeader>
            <CardTitle>Validator Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium flex items-center gap-2">
                <Shield className="h-4 w-4 text-muted-foreground" />
                Operator Address
              </span>
              <div className="flex items-center gap-2">
                <code className="font-mono text-xs">{formatHash(validator.operator_address, 12, 8)}</code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => copy(validator.operator_address)}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>

            {validator.website && (
              <div className="flex items-center justify-between py-3 border-b">
                <span className="text-sm font-medium flex items-center gap-2">
                  <Globe className="h-4 w-4 text-muted-foreground" />
                  Website
                </span>
                <a
                  href={validator.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline flex items-center gap-1 text-sm"
                >
                  {validator.website}
                  <ExternalLink className="h-3 w-3" />
                </a>
              </div>
            )}

            {validator.security_contact && (
              <div className="flex items-center justify-between py-3 border-b">
                <span className="text-sm font-medium flex items-center gap-2">
                  <Mail className="h-4 w-4 text-muted-foreground" />
                  Security Contact
                </span>
                <span className="text-sm text-muted-foreground">{validator.security_contact}</span>
              </div>
            )}

            {validator.identity && (
              <div className="flex items-center justify-between py-3 border-b">
                <span className="text-sm font-medium">Identity</span>
                <code className="font-mono text-xs">{validator.identity}</code>
              </div>
            )}

            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Jailed</span>
              <Badge variant={validator.jailed ? 'destructive' : 'outline'}>
                {validator.jailed ? 'Yes' : 'No'}
              </Badge>
            </div>

            {validator.unbonding_height !== '0' && validator.unbonding_height && (
              <div className="flex items-center justify-between py-3">
                <span className="text-sm font-medium flex items-center gap-2">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  Unbonding Height
                </span>
                <span className="text-sm">{formatNumber(parseInt(validator.unbonding_height))}</span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Commission Card */}
        <Card>
          <CardHeader>
            <CardTitle>Commission Rates</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Current Rate</span>
              <span className="text-lg font-bold">
                {formatPercent(validator.commission_rate * 100, 2)}
              </span>
            </div>

            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Max Rate</span>
              <span className="text-sm">
                {formatPercent(validator.commission_max_rate * 100, 2)}
              </span>
            </div>

            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Max Change Rate</span>
              <span className="text-sm">
                {formatPercent(validator.commission_max_change_rate * 100, 2)}
              </span>
            </div>

            {commissionEarned?.commission && (
              <div className="flex items-center justify-between py-3">
                <span className="text-sm font-medium">Commission Earned</span>
                <div className="text-right">
                  {commissionEarned.commission.map((c: any, i: number) => (
                    <div key={i} className="text-sm font-mono">
                      {formatToken(c.amount, 6, c.denom)}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Description */}
      {validator.details && (
        <Card>
          <CardHeader>
            <CardTitle>Description</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground whitespace-pre-wrap">
              {validator.details}
            </p>
          </CardContent>
        </Card>
      )}

      {/* Delegators List */}
      <Card>
        <CardHeader>
          <CardTitle>Delegators</CardTitle>
          <CardDescription>
            Top {Math.min(delegations.length, 50)} of {delegatorCount} delegator{delegatorCount !== 1 ? 's' : ''}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {delegations.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Delegator</TableHead>
                    <TableHead className="text-right">Delegated Amount</TableHead>
                    <TableHead className="text-right">Shares</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {delegations.map((delegation: Delegation, index: number) => (
                    <TableRow key={index} className="hover:bg-muted/50">
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <User className="h-4 w-4 text-muted-foreground" />
                          <Link href={`/account/${delegation.delegation.delegator_address}`}>
                            <Button variant="link" className="h-auto p-0 font-mono text-sm">
                              {formatHash(delegation.delegation.delegator_address, 12, 8)}
                            </Button>
                          </Link>
                        </div>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatToken(delegation.balance.amount, 6, delegation.balance.denom)}
                      </TableCell>
                      <TableCell className="text-right font-mono text-sm text-muted-foreground">
                        {formatToken(delegation.delegation.shares, 6)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <Users className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No delegations found</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function ValidatorSkeleton() {
  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex items-center gap-4">
        <Skeleton className="h-16 w-16 rounded-full" />
        <div>
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-4 w-96 mt-2" />
        </div>
      </div>
      <div className="grid gap-4 md:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        <Skeleton className="h-64" />
        <Skeleton className="h-64" />
      </div>
      <Skeleton className="h-96 w-full" />
    </div>
  )
}
