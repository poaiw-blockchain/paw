'use client'

import { use } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow, format } from 'date-fns'
import {
  ArrowLeft,
  CheckCircle,
  XCircle,
  Copy,
  ExternalLink,
  Clock,
  Hash,
  Box,
  User,
  FileText,
  Zap,
  AlertCircle,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import { formatNumber, formatHash, formatGas, formatToken, cn, getStatusVariant } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface PageProps {
  params: Promise<{ hash: string }>
}

export default function TransactionPage({ params }: PageProps) {
  const { hash } = use(params)
  const { copy, isCopied } = useCopyToClipboard()

  const { data: txData, isLoading, isError, error } = useQuery({
    queryKey: ['transaction', hash],
    queryFn: () => api.getTransaction(hash),
    retry: 1,
  })

  const { data: eventsData, isLoading: isLoadingEvents } = useQuery({
    queryKey: ['transactionEvents', hash],
    queryFn: () => api.getTransactionEvents(hash),
    enabled: !!txData,
  })

  if (isLoading) {
    return <TransactionSkeleton />
  }

  if (isError || !txData?.transaction) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="p-12 text-center">
            <XCircle className="h-16 w-16 mx-auto mb-4 text-destructive" />
            <h2 className="text-2xl font-bold mb-2">Transaction Not Found</h2>
            <p className="text-muted-foreground mb-6">
              {error instanceof Error ? error.message : 'The transaction you are looking for does not exist.'}
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

  const tx = txData.transaction
  const events = eventsData?.events || []
  const isSuccess = tx.status === 'success' || tx.code === 0

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold">Transaction Details</h1>
          <p className="text-muted-foreground">View transaction information and events</p>
        </div>
      </div>

      {/* Status Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {isSuccess ? (
                <CheckCircle className="h-8 w-8 text-green-500" />
              ) : (
                <XCircle className="h-8 w-8 text-red-500" />
              )}
              <div>
                <CardTitle>
                  {isSuccess ? 'Transaction Successful' : 'Transaction Failed'}
                </CardTitle>
                <CardDescription>
                  {formatDistanceToNow(new Date(tx.timestamp), { addSuffix: true })}
                </CardDescription>
              </div>
            </div>
            <Badge variant={getStatusVariant(tx.status)} className="text-base px-4 py-2">
              {tx.status.toUpperCase()}
            </Badge>
          </div>
        </CardHeader>
      </Card>

      {/* Transaction Details */}
      <Card>
        <CardHeader>
          <CardTitle>Overview</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            {/* Transaction Hash */}
            <div className="flex items-start justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <Hash className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Transaction Hash</span>
              </div>
              <div className="flex items-center gap-2">
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <code className="font-mono text-sm">{formatHash(tx.hash)}</code>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="font-mono text-xs">{tx.hash}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => copy(tx.hash)}
                >
                  {isCopied ? (
                    <CheckCircle className="h-4 w-4 text-green-500" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {/* Block Height */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <Box className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Block Height</span>
              </div>
              <Link href={`/block/${tx.block_height}`}>
                <Button variant="link" className="h-auto p-0">
                  #{formatNumber(tx.block_height)}
                </Button>
              </Link>
            </div>

            {/* Timestamp */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Timestamp</span>
              </div>
              <div className="text-right">
                <p className="text-sm font-mono">{format(new Date(tx.timestamp), 'PPpp')}</p>
                <p className="text-xs text-muted-foreground">
                  {formatDistanceToNow(new Date(tx.timestamp), { addSuffix: true })}
                </p>
              </div>
            </div>

            {/* Sender */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">From</span>
              </div>
              <Link href={`/account/${tx.sender}`}>
                <Button variant="link" className="h-auto p-0 font-mono">
                  {formatHash(tx.sender, 12, 10)}
                </Button>
              </Link>
            </div>

            {/* Type */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <FileText className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Type</span>
              </div>
              <Badge variant="outline">{tx.type}</Badge>
            </div>

            {/* Gas Used */}
            <div className="flex items-center justify-between py-3 border-b">
              <div className="flex items-center gap-2">
                <Zap className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">Gas Used</span>
              </div>
              <div className="text-right">
                <p className="text-sm font-mono">{formatGas(tx.gas_used)} / {formatGas(tx.gas_wanted)}</p>
                <p className="text-xs text-muted-foreground">
                  {((tx.gas_used / tx.gas_wanted) * 100).toFixed(2)}% utilized
                </p>
              </div>
            </div>

            {/* Fee */}
            <div className="flex items-center justify-between py-3">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">Transaction Fee</span>
              </div>
              <p className="text-sm font-mono">
                {formatToken(tx.fee_amount, 6, tx.fee_denom)}
              </p>
            </div>

            {/* Memo */}
            {tx.memo && (
              <div className="flex items-start justify-between py-3 border-t">
                <span className="text-sm font-medium">Memo</span>
                <p className="text-sm text-muted-foreground max-w-md text-right">{tx.memo}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Tabs for Messages, Events, and Raw Log */}
      <Card>
        <Tabs defaultValue="messages" className="w-full">
          <CardHeader>
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="messages">Messages ({tx.messages?.length || 0})</TabsTrigger>
              <TabsTrigger value="events">Events ({events.length})</TabsTrigger>
              <TabsTrigger value="raw">Raw Log</TabsTrigger>
            </TabsList>
          </CardHeader>
          <CardContent>
            <TabsContent value="messages" className="space-y-4">
              {tx.messages && tx.messages.length > 0 ? (
                tx.messages.map((msg: any, index: number) => (
                  <Card key={index}>
                    <CardHeader>
                      <CardTitle className="text-base">Message #{index + 1}</CardTitle>
                      <CardDescription>{msg['@type'] || msg.type}</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <pre className="bg-muted p-4 rounded-md overflow-x-auto text-xs">
                        {JSON.stringify(msg, null, 2)}
                      </pre>
                    </CardContent>
                  </Card>
                ))
              ) : (
                <p className="text-center text-muted-foreground py-8">No messages</p>
              )}
            </TabsContent>

            <TabsContent value="events">
              {isLoadingEvents ? (
                <Skeleton className="h-40 w-full" />
              ) : events.length > 0 ? (
                <div className="space-y-4">
                  {events.map((event: any, index: number) => (
                    <Card key={index}>
                      <CardHeader>
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-base">{event.type}</CardTitle>
                          <Badge variant="outline">{event.module}</Badge>
                        </div>
                      </CardHeader>
                      <CardContent>
                        <pre className="bg-muted p-4 rounded-md overflow-x-auto text-xs">
                          {JSON.stringify(event.attributes, null, 2)}
                        </pre>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <p className="text-center text-muted-foreground py-8">No events</p>
              )}
            </TabsContent>

            <TabsContent value="raw">
              {tx.raw_log ? (
                <pre className="bg-muted p-4 rounded-md overflow-x-auto text-xs">
                  {typeof tx.raw_log === 'string' ? tx.raw_log : JSON.stringify(tx.raw_log, null, 2)}
                </pre>
              ) : (
                <p className="text-center text-muted-foreground py-8">No raw log available</p>
              )}
            </TabsContent>
          </CardContent>
        </Tabs>
      </Card>

      {/* Error message if failed */}
      {!isSuccess && tx.raw_log && (
        <Card className="border-destructive">
          <CardHeader>
            <div className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-destructive" />
              <CardTitle className="text-destructive">Error Details</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <pre className="bg-destructive/10 p-4 rounded-md overflow-x-auto text-sm text-destructive">
              {typeof tx.raw_log === 'string' ? tx.raw_log : JSON.stringify(tx.raw_log, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function TransactionSkeleton() {
  return (
    <div className="container mx-auto py-8 space-y-6">
      <Skeleton className="h-12 w-96" />
      <Skeleton className="h-32 w-full" />
      <Skeleton className="h-96 w-full" />
      <Skeleton className="h-64 w-full" />
    </div>
  )
}
