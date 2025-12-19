'use client'

import { useMemo } from 'react'
import type { ComputeProvider, ComputeRequest } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber, formatPercent } from '@/lib/utils'

interface ComputeHealthOverviewProps {
  jobs: ComputeRequest[]
  providers: ComputeProvider[]
  isLoading?: boolean
}

export function ComputeHealthOverview({ jobs, providers, isLoading }: ComputeHealthOverviewProps) {
  const stats = useMemo(() => {
    const queueDepth = jobs.filter((job) => {
      const status = job.status?.toLowerCase()
      return status === 'active' || status === 'pending' || status === 'running'
    }).length

    const totalJobs = jobs.length
    const assignedJobs = jobs.filter((job) => !!job.provider).length
    const verifiedJobs = jobs.filter((job) => job.verified || job.verification_score === 'verified').length
    const assignmentRate = totalJobs > 0 ? (assignedJobs / totalJobs) * 100 : 0
    const verificationRate = totalJobs > 0 ? (verifiedJobs / totalJobs) * 100 : 0

    const totalStake = providers.reduce((sum, provider) => {
      const stakeValue = typeof provider.stake === 'string' ? parseFloat(provider.stake) : Number(provider.stake || 0)
      return sum + (isNaN(stakeValue) ? 0 : stakeValue)
    }, 0)

    const activeProviders = providers.filter((provider) => provider.active !== false).length

    return [
      {
        label: 'Queue Depth',
        value: formatNumber(queueDepth),
        helper: 'Active & pending workloads',
      },
      {
        label: 'Assignment Rate',
        value: formatPercent(assignmentRate, 1),
        helper:
          totalJobs > 0 ? `${formatNumber(assignedJobs)} of ${formatNumber(totalJobs)} workloads` : 'Awaiting queued jobs',
      },
      {
        label: 'Verification Success',
        value: formatPercent(verificationRate, 1),
        helper: totalJobs > 0 ? 'Jobs passing attestation' : 'Awaiting verification data',
      },
      {
        label: 'Collateral Locked',
        value: `${formatNumber(Math.round(totalStake))} upaw`,
        helper: `${formatNumber(activeProviders)} active providers`,
      },
    ]
  }, [jobs, providers])

  if (isLoading && jobs.length === 0 && providers.length === 0) {
    return <Skeleton className="h-[160px] w-full" />
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <Card key={stat.label}>
          <CardHeader className="pb-2">
            <CardDescription>{stat.label}</CardDescription>
            <CardTitle className="text-3xl">{stat.value}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">{stat.helper}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
