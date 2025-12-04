'use client'

import { useQuery } from '@tanstack/react-query'
import { ComputeJobTracker } from '@/components/compute/compute-job-tracker'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api } from '@/lib/api'
import { formatNumber } from '@/lib/utils'

export default function ComputePage() {
  const { data: providerData } = useQuery({
    queryKey: ['computeProviders'],
    queryFn: () => api.getComputeProviders(),
    refetchInterval: 30_000,
  })

  const providers = providerData?.providers ?? []

  return (
    <div className="container mx-auto space-y-8 py-8">
      <div>
        <h1 className="text-4xl font-bold">Compute Marketplace</h1>
        <p className="text-muted-foreground">
          Track execution markets, escrow states, and provider reliability.
        </p>
      </div>

      <ComputeJobTracker />

      <Card>
        <CardHeader>
          <CardTitle>Active Providers</CardTitle>
          <CardDescription>Diversified compute capacity across the PAW network.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            {providers.map((provider) => (
              <div key={provider.address} className="rounded-lg border p-4">
                <p className="text-sm text-muted-foreground">Provider</p>
                <p className="font-mono text-xs">{provider.address}</p>
                <div className="mt-4 text-sm space-y-1">
                  <p>Completed Jobs: {formatNumber(provider.completed_jobs || 0)}</p>
                  <p>Reputation: {provider.reputation || '—'}</p>
                  <p>Stake: {provider.stake || '0'} upaw</p>
                </div>
              </div>
            ))}
            {providers.length === 0 && <p className="text-sm text-muted-foreground">No providers registered.</p>}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
