'use client'

import { useQuery } from '@tanstack/react-query'
import { ComputeJobTracker } from '@/components/compute/compute-job-tracker'
import { ComputeHealthOverview } from '@/components/compute/compute-health-overview'
import { ComputeLatencyChart } from '@/components/compute/compute-latency-chart'
import { ComputeProviderReliability } from '@/components/compute/compute-provider-reliability'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api } from '@/lib/api'
import { formatNumber } from '@/lib/utils'

export default function ComputePage() {
  const {
    data: providerData,
    isLoading: loadingProviders,
  } = useQuery({
    queryKey: ['computeProviders'],
    queryFn: () => api.getComputeProviders(),
    refetchInterval: 30_000,
  })

  const {
    data: jobsData,
    isLoading: loadingJobs,
  } = useQuery({
    queryKey: ['computeRequests', 'all'],
    queryFn: () => api.getComputeRequests(1, 50),
    refetchInterval: 20_000,
  })

  const providers = providerData?.providers ?? []
  const jobs = jobsData?.data ?? []

  return (
    <div className="container mx-auto space-y-8 py-8">
      <div>
        <h1 className="text-4xl font-bold">Compute Marketplace</h1>
        <p className="text-muted-foreground">
          Track execution markets, escrow states, and provider reliability.
        </p>
      </div>

      <ComputeHealthOverview jobs={jobs} providers={providers} isLoading={loadingJobs || loadingProviders} />

      <div className="grid gap-6 lg:grid-cols-2">
        <ComputeLatencyChart jobs={jobs} isLoading={loadingJobs} />
        <ComputeProviderReliability providers={providers} jobs={jobs} isLoading={loadingProviders} />
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
            {!loadingProviders && providers.length === 0 && (
              <p className="text-sm text-muted-foreground">No providers registered.</p>
            )}
            {loadingProviders && providers.length === 0 && (
              <p className="text-sm text-muted-foreground">Loading provider registry…</p>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
