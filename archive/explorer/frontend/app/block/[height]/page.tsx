'use client'

import { use } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow, format } from 'date-fns'
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Copy,
  Box,
  User,
  Clock,
  Hash,
  FileText,
  Zap,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { api, Transaction } from '@/lib/api'
import { formatNumber, formatHash, formatGas, formatToken, cn, getStatusVariant } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface PageProps {
  params: Promise<{ height: string }>
}

export default function BlockPage({ params }: PageProps) {
  const { height } = use(params)
  const blockHeight = parseInt(height)
  const { copy, isCopied } = useCopyToClipboard()

  const { data: blockData, isLoading, isError, error } = useQuery({
    queryKey: ['block', blockHeight],
    queryFn: () => api.getBlock(blockHeight),
    retry: 1,
  })

  const { data: txsData, isLoading: isLoadingTxs } = useQuery({
    queryKey: ['blockTransactions', blockHeight],
    queryFn: () => api.getBlockTransactions(blockHeight),
    enabled: !!blockData,
  })

  if (isLoading) {
    return <BlockSkeleton />
  }

  if (isError || !blockData?.block) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="p-12 text-center">
            <XCircle className="h-16 w-16 mx-auto mb-4 text-destructive" />
            <h2 className="text-2xl font-bold mb-2">Block Not Found</h2>
            <p className="text-muted-foreground mb-6">
              {error instanceof Error ? error.message : 'The block you are looking for does not exist.'}
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

  const block = blockData.block
  const transactions = txsData?.transactions || []

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/">
            <Button variant="ghost" size="icon">
              <ArrowLeft className="h-5 w-5" />
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Block #{formatNumber(block.height)}</h1>
            <p className="text-muted-foreground">
              {formatDistanceToNow(new Date(block.timestamp), { addSuffix: true })}
            </p>
          </div>
        </div>

        {/* Block Navigation */}
        <div className="flex items-center gap-2">
          <Link href={`/block/${blockHeight - 1}`}>
            <Button variant="outline" size="sm" disabled={blockHeight <= 1}>
              <ChevronLeft className="h-4 w-4 mr-1" />
              Previous
            </Button>
          </Link>
          <Link href={`/block/${blockHeight + 1}`}>
            <Button variant="outline" size="sm">
              Next
              <ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          </Link>
        </div>
      </div>

      {/* Block Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Height</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Box className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">#{formatNumber(block.height)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Transactions</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">{formatNumber(block.tx_count)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Gas Used</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Zap className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">{formatGas(block.gas_used)}</p>
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {formatGas(block.gas_wanted)} wanted
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Timestamp</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-primary" />
              <p className="text-sm font-medium">{format(new Date(block.timestamp), 'PPp')}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Block Details */}
      <Card>
        <CardHeader>
          <CardTitle>Block Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            {/* Block Hash */}
            <div className="flex items-start justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <Hash className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Block Hash</span>
              </div>
              <div className="flex items-center gap-2">
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <code className="font-mono text-sm">{formatHash(block.hash)}</code>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="font-mono text-xs">{block.hash}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => copy(block.hash)}
                >
                  {isCopied ? (
                    <CheckCircle className="h-4 w-4 text-green-500" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {/* Chain ID */}
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Chain ID</span>
              <Badge variant="outline">{block.chain_id}</Badge>
            </div>

            {/* Proposer */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Proposer</span>
              </div>
              <Link href={`/validator/${block.proposer_address}`}>
                <Button variant="link" className="h-auto p-0 font-mono">
                  {formatHash(block.proposer_address, 12, 10)}
                </Button>
              </Link>
            </div>

            {/* Gas */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <Zap className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Gas</span>
              </div>
              <div className="text-right">
                <p className="text-sm font-mono">
                  {formatNumber(block.gas_used)} / {formatNumber(block.gas_wanted)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {((block.gas_used / block.gas_wanted) * 100).toFixed(2)}% utilized
                </p>
              </div>
            </div>

            {/* Evidence */}
            {block.evidence_count > 0 && (
              <div className="flex items-center justify-between py-3">
                <span className="text-sm font-medium">Evidence</span>
                <Badge variant="destructive">{block.evidence_count} evidence(s)</Badge>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Transactions */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Transactions</CardTitle>
              <CardDescription>
                {block.tx_count} transaction{block.tx_count !== 1 ? 's' : ''} in this block
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoadingTxs ? (
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : transactions.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Status</TableHead>
                    <TableHead>Hash</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>From</TableHead>
                    <TableHead>Gas Used</TableHead>
                    <TableHead className="text-right">Fee</TableHead>
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
                        <Link href={`/account/${tx.sender}`}>
                          <Button variant="link" className="h-auto p-0 font-mono text-xs">
                            {formatHash(tx.sender, 8, 6)}
                          </Button>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{formatGas(tx.gas_used)}</span>
                      </TableCell>
                      <TableCell className="text-right">
                        <span className="text-sm font-mono">
                          {formatToken(tx.fee_amount, 6)}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No transactions in this block</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function BlockSkeleton() {
  return (
    <div className="container mx-auto py-8 space-y-6">
      <Skeleton className="h-12 w-96" />
      <div className="grid gap-4 md:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
      <Skeleton className="h-64 w-full" />
      <Skeleton className="h-96 w-full" />
    </div>
  )
}
