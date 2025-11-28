'use client'

import { use, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow, format } from 'date-fns'
import {
  ArrowLeft,
  Copy,
  Wallet,
  Activity,
  Clock,
  TrendingUp,
  TrendingDown,
  CheckCircle,
  XCircle,
  Coins,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api, Transaction } from '@/lib/api'
import { formatNumber, formatHash, formatGas, formatToken, cn, getStatusVariant } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface PageProps {
  params: Promise<{ address: string }>
}

export default function AccountPage({ params }: PageProps) {
  const { address } = use(params)
  const { copy, isCopied } = useCopyToClipboard()
  const [txPage, setTxPage] = useState(1)
  const txLimit = 20

  const { data: accountData, isLoading, isError, error } = useQuery({
    queryKey: ['account', address],
    queryFn: () => api.getAccount(address),
    retry: 1,
  })

  const { data: balancesData, isLoading: isLoadingBalances } = useQuery({
    queryKey: ['accountBalances', address],
    queryFn: () => api.getAccountBalances(address),
    enabled: !!accountData,
  })

  const { data: tokensData, isLoading: isLoadingTokens } = useQuery({
    queryKey: ['accountTokens', address],
    queryFn: () => api.getAccountTokens(address),
    enabled: !!accountData,
  })

  const { data: txsData, isLoading: isLoadingTxs } = useQuery({
    queryKey: ['accountTransactions', address, txPage],
    queryFn: () => api.getAccountTransactions(address, txPage, txLimit),
    enabled: !!accountData,
    keepPreviousData: true,
  })

  if (isLoading) {
    return <AccountSkeleton />
  }

  if (isError || !accountData?.account) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="p-12 text-center">
            <XCircle className="h-16 w-16 mx-auto mb-4 text-destructive" />
            <h2 className="text-2xl font-bold mb-2">Account Not Found</h2>
            <p className="text-muted-foreground mb-6">
              {error instanceof Error ? error.message : 'The account you are looking for does not exist.'}
            </p>
            <Link href="/">
              <Button>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Explorer
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  const account = accountData.account
  const balances = balancesData?.balances || []
  const tokens = tokensData?.tokens || []
  const transactions = txsData?.data || []

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">Account Details</h1>
          <div className="flex items-center gap-2 mt-2">
            <code className="text-sm font-mono bg-muted px-3 py-1 rounded">{address}</code>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => copy(address)}
            >
              {isCopied ? (
                <CheckCircle className="h-4 w-4 text-green-500" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* Account Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Transactions</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">{formatNumber(account.tx_count)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Received</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-green-500" />
              <p className="text-lg font-bold">{formatToken(account.total_received, 6)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Sent</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <TrendingDown className="h-5 w-5 text-red-500" />
              <p className="text-lg font-bold">{formatToken(account.total_sent, 6)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>First Seen</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-primary" />
              <p className="text-sm font-medium">
                {formatDistanceToNow(new Date(account.first_seen_at), { addSuffix: true })}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs for Balances, Tokens, and Transactions */}
      <Card>
        <Tabs defaultValue="transactions" className="w-full">
          <CardHeader>
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="transactions">
                Transactions ({account.tx_count})
              </TabsTrigger>
              <TabsTrigger value="balances">Balances ({balances.length})</TabsTrigger>
              <TabsTrigger value="tokens">Tokens ({tokens.length})</TabsTrigger>
            </TabsList>
          </CardHeader>
          <CardContent>
            {/* Transactions Tab */}
            <TabsContent value="transactions" className="space-y-4">
              {isLoadingTxs ? (
                <div className="space-y-4">
                  {[...Array(5)].map((_, i) => (
                    <Skeleton key={i} className="h-16 w-full" />
                  ))}
                </div>
              ) : transactions.length > 0 ? (
                <>
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Status</TableHead>
                          <TableHead>Hash</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead>Block</TableHead>
                          <TableHead>Gas Used</TableHead>
                          <TableHead>Time</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {transactions.map((tx: Transaction) => (
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
                              <span className="text-sm">{formatGas(tx.gas_used)}</span>
                            </TableCell>
                            <TableCell>
                              <span className="text-sm text-muted-foreground">
                                {formatDistanceToNow(new Date(tx.timestamp), { addSuffix: true })}
                              </span>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>

                  {/* Pagination */}
                  {txsData && txsData.total > txLimit && (
                    <div className="flex items-center justify-between pt-4">
                      <div className="text-sm text-muted-foreground">
                        Showing {(txPage - 1) * txLimit + 1} to {Math.min(txPage * txLimit, txsData.total)} of{' '}
                        {formatNumber(txsData.total)} transactions
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setTxPage((p) => Math.max(1, p - 1))}
                          disabled={txPage === 1}
                        >
                          <ChevronLeft className="h-4 w-4 mr-1" />
                          Previous
                        </Button>
                        <div className="flex items-center gap-1">
                          <span className="text-sm">Page</span>
                          <Badge variant="outline">{txPage}</Badge>
                          <span className="text-sm">of {Math.ceil(txsData.total / txLimit)}</span>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setTxPage((p) => p + 1)}
                          disabled={txPage >= Math.ceil(txsData.total / txLimit)}
                        >
                          Next
                          <ChevronRight className="h-4 w-4 ml-1" />
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Activity className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>No transactions found</p>
                </div>
              )}
            </TabsContent>

            {/* Balances Tab */}
            <TabsContent value="balances">
              {isLoadingBalances ? (
                <Skeleton className="h-40 w-full" />
              ) : balances.length > 0 ? (
                <div className="space-y-4">
                  {balances.map((balance: any, index: number) => (
                    <Card key={index}>
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <Coins className="h-8 w-8 text-primary" />
                            <div>
                              <p className="font-semibold">{balance.denom.toUpperCase()}</p>
                              <p className="text-sm text-muted-foreground">
                                Updated at block #{formatNumber(balance.last_updated_height)}
                              </p>
                            </div>
                          </div>
                          <div className="text-right">
                            <p className="text-2xl font-bold">{formatToken(balance.amount, 6)}</p>
                            <p className="text-xs text-muted-foreground">
                              {formatDistanceToNow(new Date(balance.last_updated_at), { addSuffix: true })}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Wallet className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>No balances found</p>
                </div>
              )}
            </TabsContent>

            {/* Tokens Tab */}
            <TabsContent value="tokens">
              {isLoadingTokens ? (
                <Skeleton className="h-40 w-full" />
              ) : tokens.length > 0 ? (
                <div className="space-y-4">
                  {tokens.map((token: any, index: number) => (
                    <Card key={index}>
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <Coins className="h-8 w-8 text-primary" />
                            <div>
                              <p className="font-semibold">
                                {token.token_name || token.token_symbol}
                              </p>
                              <p className="text-sm text-muted-foreground font-mono">
                                {token.token_denom}
                              </p>
                            </div>
                          </div>
                          <div className="text-right">
                            <p className="text-2xl font-bold">
                              {formatToken(token.amount, 6, token.token_symbol)}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {formatDistanceToNow(new Date(token.last_updated_at), { addSuffix: true })}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Coins className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>No tokens found</p>
                </div>
              )}
            </TabsContent>
          </CardContent>
        </Tabs>
      </Card>

      {/* Account Timeline */}
      <Card>
        <CardHeader>
          <CardTitle>Account Timeline</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">First Seen</span>
              <div className="text-right">
                <p className="text-sm">Block #{formatNumber(account.first_seen_height)}</p>
                <p className="text-xs text-muted-foreground">
                  {format(new Date(account.first_seen_at), 'PPpp')}
                </p>
              </div>
            </div>
            <div className="flex items-center justify-between py-3">
              <span className="text-sm font-medium">Last Seen</span>
              <div className="text-right">
                <p className="text-sm">Block #{formatNumber(account.last_seen_height)}</p>
                <p className="text-xs text-muted-foreground">
                  {format(new Date(account.last_seen_at), 'PPpp')}
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function AccountSkeleton() {
  return (
    <div className="container mx-auto py-8 space-y-6">
      <Skeleton className="h-12 w-96" />
      <div className="grid gap-4 md:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
      <Skeleton className="h-96 w-full" />
    </div>
  )
}
