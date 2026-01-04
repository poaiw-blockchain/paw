'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow, format } from 'date-fns'
import {
  Coins,
  Search,
  Shield,
  Clock,
  TrendingUp,
  Wallet,
  AlertCircle,
  ArrowRight,
  Gift,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api, Delegation, UnbondingDelegation, DelegationReward } from '@/lib/api'
import { formatNumber, formatHash, formatToken, formatDuration } from '@/lib/utils'

export default function StakingPage() {
  const [address, setAddress] = useState('')
  const [searchAddress, setSearchAddress] = useState('')

  // Get staking pool info
  const { data: poolData, isLoading: loadingPool } = useQuery({
    queryKey: ['stakingPool'],
    queryFn: () => api.getStakingPool(),
    refetchInterval: 60000,
  })

  // Get delegations for entered address
  const { data: delegationsData, isLoading: loadingDelegations, isError: delegationsError } = useQuery({
    queryKey: ['stakingDelegations', searchAddress],
    queryFn: () => api.getStakingDelegations(searchAddress),
    enabled: !!searchAddress && searchAddress.length > 10,
  })

  // Get rewards for entered address
  const { data: rewardsData, isLoading: loadingRewards } = useQuery({
    queryKey: ['stakingRewards', searchAddress],
    queryFn: () => api.getStakingRewards(searchAddress),
    enabled: !!searchAddress && searchAddress.length > 10,
  })

  const pool = poolData?.pool
  const params = poolData?.params
  const delegations = delegationsData?.delegations || []
  const unbonding = delegationsData?.unbonding || []
  const rewards = rewardsData?.rewards || []
  const totalRewards = rewardsData?.total || []

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (address.trim()) {
      setSearchAddress(address.trim())
    }
  }

  // Calculate totals
  const totalDelegated = delegations.reduce((sum, d) => {
    return sum + parseInt(d.balance.amount || '0')
  }, 0)

  const totalUnbonding = unbonding.reduce((sum, u) => {
    return sum + u.entries.reduce((s, e) => s + parseInt(e.balance || '0'), 0)
  }, 0)

  const totalRewardAmount = totalRewards.reduce((sum, r) => {
    return sum + parseFloat(r.amount || '0')
  }, 0)

  // Calculate pool percentages
  const bondedTokens = parseInt(pool?.bonded_tokens || '0')
  const notBondedTokens = parseInt(pool?.not_bonded_tokens || '0')
  const totalStaked = bondedTokens + notBondedTokens
  const bondedPercent = totalStaked > 0 ? (bondedTokens / totalStaked) * 100 : 0

  // Parse unbonding time
  const unbondingSeconds = params?.unbonding_time ? parseInt(params.unbonding_time) / 1_000_000_000 : 0

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <Card>
        <CardHeader>
          <CardTitle className="text-3xl flex items-center gap-2">
            <Coins className="h-8 w-8 text-primary" />
            Staking Dashboard
          </CardTitle>
          <CardDescription>
            View staking information, delegations, and rewards on PAW Chain
          </CardDescription>
        </CardHeader>
      </Card>

      {/* Network Staking Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Bonded Tokens</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-primary" />
              <p className="text-xl font-bold">{formatToken(bondedTokens.toString(), 6)}</p>
            </div>
            <div className="mt-2">
              <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary transition-all"
                  style={{ width: `${bondedPercent}%` }}
                />
              </div>
              <p className="text-xs text-muted-foreground mt-1">{bondedPercent.toFixed(2)}% bonded</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Not Bonded Tokens</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Wallet className="h-5 w-5 text-muted-foreground" />
              <p className="text-xl font-bold">{formatToken(notBondedTokens.toString(), 6)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Unbonding Period</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-primary" />
              <p className="text-xl font-bold">
                {unbondingSeconds > 0 ? formatDuration(unbondingSeconds) : 'N/A'}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Max Validators</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-primary" />
              <p className="text-xl font-bold">{params?.max_validators || 'N/A'}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Address Search */}
      <Card>
        <CardHeader>
          <CardTitle>View Account Staking</CardTitle>
          <CardDescription>
            Enter an address to view delegations and rewards
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSearch} className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Enter PAW address (e.g., paw1...)"
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                className="pl-9"
              />
            </div>
            <Button type="submit" disabled={!address.trim()}>
              <Search className="h-4 w-4 mr-2" />
              Search
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* Account Staking Info */}
      {searchAddress && (
        <>
          {/* Account Stats */}
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardHeader className="pb-3">
                <CardDescription>Total Delegated</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <TrendingUp className="h-5 w-5 text-green-500" />
                  <p className="text-2xl font-bold">{formatToken(totalDelegated.toString(), 6)}</p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardDescription>Unbonding</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <Clock className="h-5 w-5 text-yellow-500" />
                  <p className="text-2xl font-bold">{formatToken(totalUnbonding.toString(), 6)}</p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardDescription>Pending Rewards</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <Gift className="h-5 w-5 text-primary" />
                  <p className="text-2xl font-bold">{formatToken(totalRewardAmount.toFixed(0), 6)}</p>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Tabs for Delegations, Unbonding, Rewards */}
          <Card>
            <Tabs defaultValue="delegations" className="w-full">
              <CardHeader>
                <TabsList className="grid w-full grid-cols-3">
                  <TabsTrigger value="delegations">
                    Delegations ({delegations.length})
                  </TabsTrigger>
                  <TabsTrigger value="unbonding">
                    Unbonding ({unbonding.length})
                  </TabsTrigger>
                  <TabsTrigger value="rewards">
                    Rewards ({rewards.length})
                  </TabsTrigger>
                </TabsList>
              </CardHeader>
              <CardContent>
                {/* Delegations Tab */}
                <TabsContent value="delegations" className="space-y-4">
                  {loadingDelegations ? (
                    <div className="space-y-4">
                      {[...Array(3)].map((_, i) => (
                        <Skeleton key={i} className="h-16 w-full" />
                      ))}
                    </div>
                  ) : delegationsError ? (
                    <div className="text-center py-8 text-muted-foreground">
                      <AlertCircle className="h-12 w-12 mx-auto mb-2 opacity-50" />
                      <p>Failed to load delegations or address not found</p>
                    </div>
                  ) : delegations.length > 0 ? (
                    <div className="overflow-x-auto">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Validator</TableHead>
                            <TableHead className="text-right">Amount</TableHead>
                            <TableHead className="text-right">Shares</TableHead>
                            <TableHead></TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {delegations.map((delegation: Delegation, index: number) => (
                            <TableRow key={index} className="hover:bg-muted/50">
                              <TableCell>
                                <Link href={`/validators/${delegation.delegation.validator_address}`}>
                                  <Button variant="link" className="h-auto p-0 font-mono text-sm">
                                    {formatHash(delegation.delegation.validator_address, 16, 8)}
                                  </Button>
                                </Link>
                              </TableCell>
                              <TableCell className="text-right font-mono">
                                {formatToken(delegation.balance.amount, 6, delegation.balance.denom)}
                              </TableCell>
                              <TableCell className="text-right font-mono text-sm text-muted-foreground">
                                {formatToken(delegation.delegation.shares, 6)}
                              </TableCell>
                              <TableCell>
                                <Link href={`/validators/${delegation.delegation.validator_address}`}>
                                  <Button variant="ghost" size="sm">
                                    <ArrowRight className="h-4 w-4" />
                                  </Button>
                                </Link>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      <Coins className="h-12 w-12 mx-auto mb-2 opacity-50" />
                      <p>No delegations found for this address</p>
                    </div>
                  )}
                </TabsContent>

                {/* Unbonding Tab */}
                <TabsContent value="unbonding" className="space-y-4">
                  {loadingDelegations ? (
                    <Skeleton className="h-32 w-full" />
                  ) : unbonding.length > 0 ? (
                    <div className="space-y-4">
                      {unbonding.map((u: UnbondingDelegation, index: number) => (
                        <Card key={index}>
                          <CardHeader className="pb-2">
                            <CardDescription className="flex items-center justify-between">
                              <span>Validator</span>
                              <Link href={`/validators/${u.validator_address}`}>
                                <Button variant="link" className="h-auto p-0 font-mono text-xs">
                                  {formatHash(u.validator_address, 12, 8)}
                                </Button>
                              </Link>
                            </CardDescription>
                          </CardHeader>
                          <CardContent>
                            <div className="space-y-3">
                              {u.entries.map((entry, i) => (
                                <div key={i} className="flex items-center justify-between py-2 border-b last:border-0">
                                  <div>
                                    <p className="font-mono">
                                      {formatToken(entry.balance, 6)}
                                    </p>
                                    <p className="text-xs text-muted-foreground">
                                      Started at block #{entry.creation_height}
                                    </p>
                                  </div>
                                  <div className="text-right">
                                    <Badge variant="outline">
                                      <Clock className="h-3 w-3 mr-1" />
                                      {formatDistanceToNow(new Date(entry.completion_time), { addSuffix: true })}
                                    </Badge>
                                    <p className="text-xs text-muted-foreground mt-1">
                                      {format(new Date(entry.completion_time), 'PPp')}
                                    </p>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      <Clock className="h-12 w-12 mx-auto mb-2 opacity-50" />
                      <p>No unbonding delegations</p>
                    </div>
                  )}
                </TabsContent>

                {/* Rewards Tab */}
                <TabsContent value="rewards" className="space-y-4">
                  {loadingRewards ? (
                    <Skeleton className="h-32 w-full" />
                  ) : rewards.length > 0 ? (
                    <>
                      {/* Total Rewards */}
                      {totalRewards.length > 0 && (
                        <Card className="bg-primary/5 border-primary/20">
                          <CardContent className="pt-6">
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-2">
                                <Gift className="h-5 w-5 text-primary" />
                                <span className="font-medium">Total Claimable Rewards</span>
                              </div>
                              <div className="text-right">
                                {totalRewards.map((r, i) => (
                                  <p key={i} className="text-lg font-bold">
                                    {formatToken(parseFloat(r.amount).toFixed(0), 6, r.denom)}
                                  </p>
                                ))}
                              </div>
                            </div>
                          </CardContent>
                        </Card>
                      )}

                      {/* Per-Validator Rewards */}
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Validator</TableHead>
                              <TableHead className="text-right">Pending Rewards</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {rewards.map((reward: DelegationReward, index: number) => (
                              <TableRow key={index} className="hover:bg-muted/50">
                                <TableCell>
                                  <Link href={`/validators/${reward.validator_address}`}>
                                    <Button variant="link" className="h-auto p-0 font-mono text-sm">
                                      {formatHash(reward.validator_address, 16, 8)}
                                    </Button>
                                  </Link>
                                </TableCell>
                                <TableCell className="text-right">
                                  {reward.reward.map((r, i) => (
                                    <span key={i} className="font-mono text-sm">
                                      {formatToken(parseFloat(r.amount).toFixed(0), 6, r.denom)}
                                    </span>
                                  ))}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    </>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      <Gift className="h-12 w-12 mx-auto mb-2 opacity-50" />
                      <p>No pending rewards</p>
                    </div>
                  )}
                </TabsContent>
              </CardContent>
            </Tabs>
          </Card>
        </>
      )}

      {/* Quick Links */}
      {!searchAddress && (
        <Card>
          <CardHeader>
            <CardTitle>Quick Links</CardTitle>
          </CardHeader>
          <CardContent className="flex gap-4">
            <Link href="/validators">
              <Button variant="outline">
                <Shield className="h-4 w-4 mr-2" />
                View All Validators
              </Button>
            </Link>
            <Link href="/governance">
              <Button variant="outline">
                <TrendingUp className="h-4 w-4 mr-2" />
                Governance Proposals
              </Button>
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
