'use client'

import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { DexPoolVisualization } from '@/components/dex/pool-visualization'
import { api, type DEXTrade } from '@/lib/api'
import { formatCurrency, formatHash, formatNumber } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'

export default function DexPage() {
  const { data: tradesData, isLoading } = useQuery({
    queryKey: ['dexTrades'],
    queryFn: () => api.getDEXTrades(1, 12),
    refetchInterval: 15_000,
  })

  const trades = tradesData?.data ?? []

  return (
    <div className="container mx-auto space-y-8 py-8">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-4xl font-bold">PAW DEX Analytics</h1>
          <p className="text-muted-foreground">
            Inspect liquidity, flow, and capital efficiency across the on-chain exchange.
          </p>
        </div>
        <Button asChild>
          <Link href="https://docs.pawchain.network/dex" target="_blank">
            DEX Documentation
          </Link>
        </Button>
      </div>

      <DexPoolVisualization />

      <Card>
        <CardHeader className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <CardTitle>Recent Trades</CardTitle>
            <CardDescription>High-frequency swaps executed in the last blocks.</CardDescription>
          </div>
          <Button variant="ghost" onClick={() => location.reload()}>
            Refresh
          </Button>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Pool</TableHead>
                  <TableHead>Trader</TableHead>
                  <TableHead>In</TableHead>
                  <TableHead>Out</TableHead>
                  <TableHead>Fee</TableHead>
                  <TableHead>Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  [...Array(6)].map((_, idx) => (
                    <TableRow key={idx}>
                      <TableCell colSpan={6} className="h-12">
                        <div className="animate-pulse rounded bg-muted h-6 w-1/3" />
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  trades.map((trade: DEXTrade) => (
                    <TableRow key={`${trade.tx_hash}-${trade.pool_id}-${trade.timestamp}`}>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-semibold">Pool #{trade.pool_id}</span>
                          <span className="text-xs text-muted-foreground">
                            {trade.token_in}/{trade.token_out}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs">{formatHash(trade.trader)}</span>
                      </TableCell>
                      <TableCell>
                        {formatCurrency(trade.amount_in || '0')} {trade.token_in.toUpperCase()}
                      </TableCell>
                      <TableCell>
                        {formatCurrency(trade.amount_out || '0')} {trade.token_out.toUpperCase()}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{formatNumber(trade.fee || '0')} bps</Badge>
                      </TableCell>
                      <TableCell>
                        <p className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(trade.timestamp), { addSuffix: true })}
                        </p>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
