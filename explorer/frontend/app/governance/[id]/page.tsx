'use client'

import { use, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { formatDistanceToNow, format } from 'date-fns'
import {
  ArrowLeft,
  Vote,
  CheckCircle,
  XCircle,
  AlertCircle,
  Clock,
  FileText,
  User,
  Copy,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api, Proposal, ProposalVote } from '@/lib/api'
import { formatNumber, formatHash, formatToken } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface PageProps {
  params: Promise<{ id: string }>
}

function getStatusBadgeVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status.toLowerCase()) {
    case 'voting':
      return 'default'
    case 'passed':
      return 'secondary'
    case 'rejected':
    case 'failed':
      return 'destructive'
    default:
      return 'outline'
  }
}

function getVoteColor(option: string): string {
  switch (option.toLowerCase()) {
    case 'yes':
      return 'bg-green-500'
    case 'no':
      return 'bg-red-500'
    case 'abstain':
      return 'bg-gray-500'
    case 'nowithveto':
      return 'bg-orange-500'
    default:
      return 'bg-blue-500'
  }
}

export default function ProposalDetailPage({ params }: PageProps) {
  const { id } = use(params)
  const proposalId = parseInt(id)
  const { copy, isCopied } = useCopyToClipboard()
  const [showFullDescription, setShowFullDescription] = useState(false)

  const { data: proposalData, isLoading, isError, error } = useQuery({
    queryKey: ['proposal', proposalId],
    queryFn: () => api.getProposal(proposalId),
    retry: 1,
  })

  const { data: votesData, isLoading: isLoadingVotes } = useQuery({
    queryKey: ['proposalVotes', proposalId],
    queryFn: () => api.getProposalVotes(proposalId),
    enabled: !!proposalData,
  })

  if (isLoading) {
    return <ProposalSkeleton />
  }

  if (isError || !proposalData?.proposal) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="p-12 text-center">
            <XCircle className="h-16 w-16 mx-auto mb-4 text-destructive" />
            <h2 className="text-2xl font-bold mb-2">Proposal Not Found</h2>
            <p className="text-muted-foreground mb-6">
              {error instanceof Error ? error.message : 'The proposal you are looking for does not exist.'}
            </p>
            <Link href="/governance">
              <Button>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Governance
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  const proposal = proposalData.proposal
  const tally = proposalData.tally
  const votes = votesData?.votes || []

  // Calculate tally percentages
  const yesVotes = parseInt(tally?.yes || '0')
  const noVotes = parseInt(tally?.no || '0')
  const abstainVotes = parseInt(tally?.abstain || '0')
  const noWithVetoVotes = parseInt(tally?.no_with_veto || '0')
  const totalVotes = yesVotes + noVotes + abstainVotes + noWithVetoVotes

  const yesPercent = totalVotes > 0 ? (yesVotes / totalVotes) * 100 : 0
  const noPercent = totalVotes > 0 ? (noVotes / totalVotes) * 100 : 0
  const abstainPercent = totalVotes > 0 ? (abstainVotes / totalVotes) * 100 : 0
  const noWithVetoPercent = totalVotes > 0 ? (noWithVetoVotes / totalVotes) * 100 : 0

  const description = proposal.content?.description || 'No description available'
  const truncatedDescription = description.length > 500 ? description.slice(0, 500) + '...' : description

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/governance">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-3xl font-bold">Proposal #{proposal.proposal_id}</h1>
            <Badge variant={getStatusBadgeVariant(proposal.status_label)}>
              {proposal.status_label}
            </Badge>
          </div>
          <p className="text-xl text-muted-foreground mt-1">
            {proposal.content?.title || 'Untitled Proposal'}
          </p>
        </div>
      </div>

      {/* Overview Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Voting Period</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-primary" />
              <div className="text-sm">
                {proposal.voting_end_time ? (
                  <>
                    <span className="font-medium">
                      {format(new Date(proposal.voting_end_time), 'MMM d, yyyy')}
                    </span>
                    <br />
                    <span className="text-muted-foreground">
                      {formatDistanceToNow(new Date(proposal.voting_end_time), { addSuffix: true })}
                    </span>
                  </>
                ) : (
                  'N/A'
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Votes</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Vote className="h-5 w-5 text-primary" />
              <p className="text-2xl font-bold">{formatNumber(totalVotes)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Type</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge variant="outline">
              {proposal.content?.['@type']?.split('.').pop() || 'Unknown'}
            </Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Deposit</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold">
              {proposal.total_deposit?.[0]
                ? formatToken(proposal.total_deposit[0].amount, 6, proposal.total_deposit[0].denom)
                : '0'}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Vote Tally Visualization */}
      <Card>
        <CardHeader>
          <CardTitle>Vote Tally</CardTitle>
          <CardDescription>Current voting results</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Progress Bar */}
          <div className="h-8 w-full rounded-full overflow-hidden bg-muted flex">
            {yesPercent > 0 && (
              <div
                className="bg-green-500 h-full transition-all"
                style={{ width: `${yesPercent}%` }}
                title={`Yes: ${yesPercent.toFixed(2)}%`}
              />
            )}
            {noPercent > 0 && (
              <div
                className="bg-red-500 h-full transition-all"
                style={{ width: `${noPercent}%` }}
                title={`No: ${noPercent.toFixed(2)}%`}
              />
            )}
            {abstainPercent > 0 && (
              <div
                className="bg-gray-500 h-full transition-all"
                style={{ width: `${abstainPercent}%` }}
                title={`Abstain: ${abstainPercent.toFixed(2)}%`}
              />
            )}
            {noWithVetoPercent > 0 && (
              <div
                className="bg-orange-500 h-full transition-all"
                style={{ width: `${noWithVetoPercent}%` }}
                title={`NoWithVeto: ${noWithVetoPercent.toFixed(2)}%`}
              />
            )}
          </div>

          {/* Vote Breakdown */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="p-4 rounded-lg border bg-green-500/10 border-green-500/20">
              <div className="flex items-center gap-2 mb-2">
                <CheckCircle className="h-5 w-5 text-green-500" />
                <span className="font-medium">Yes</span>
              </div>
              <p className="text-2xl font-bold text-green-500">{yesPercent.toFixed(2)}%</p>
              <p className="text-sm text-muted-foreground">{formatNumber(yesVotes)} votes</p>
            </div>

            <div className="p-4 rounded-lg border bg-red-500/10 border-red-500/20">
              <div className="flex items-center gap-2 mb-2">
                <XCircle className="h-5 w-5 text-red-500" />
                <span className="font-medium">No</span>
              </div>
              <p className="text-2xl font-bold text-red-500">{noPercent.toFixed(2)}%</p>
              <p className="text-sm text-muted-foreground">{formatNumber(noVotes)} votes</p>
            </div>

            <div className="p-4 rounded-lg border bg-gray-500/10 border-gray-500/20">
              <div className="flex items-center gap-2 mb-2">
                <AlertCircle className="h-5 w-5 text-gray-500" />
                <span className="font-medium">Abstain</span>
              </div>
              <p className="text-2xl font-bold text-gray-500">{abstainPercent.toFixed(2)}%</p>
              <p className="text-sm text-muted-foreground">{formatNumber(abstainVotes)} votes</p>
            </div>

            <div className="p-4 rounded-lg border bg-orange-500/10 border-orange-500/20">
              <div className="flex items-center gap-2 mb-2">
                <XCircle className="h-5 w-5 text-orange-500" />
                <span className="font-medium">No With Veto</span>
              </div>
              <p className="text-2xl font-bold text-orange-500">{noWithVetoPercent.toFixed(2)}%</p>
              <p className="text-sm text-muted-foreground">{formatNumber(noWithVetoVotes)} votes</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Proposal Description */}
      <Card>
        <CardHeader>
          <CardTitle>Description</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="prose prose-sm dark:prose-invert max-w-none">
            <pre className="whitespace-pre-wrap font-sans text-sm">
              {showFullDescription ? description : truncatedDescription}
            </pre>
            {description.length > 500 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowFullDescription(!showFullDescription)}
                className="mt-2"
              >
                {showFullDescription ? (
                  <>
                    <ChevronUp className="h-4 w-4 mr-1" />
                    Show Less
                  </>
                ) : (
                  <>
                    <ChevronDown className="h-4 w-4 mr-1" />
                    Show More
                  </>
                )}
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Timeline */}
      <Card>
        <CardHeader>
          <CardTitle>Timeline</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Submit Time</span>
              <span className="text-sm text-muted-foreground">
                {proposal.submit_time
                  ? format(new Date(proposal.submit_time), 'PPpp')
                  : 'N/A'}
              </span>
            </div>
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Deposit End</span>
              <span className="text-sm text-muted-foreground">
                {proposal.deposit_end_time
                  ? format(new Date(proposal.deposit_end_time), 'PPpp')
                  : 'N/A'}
              </span>
            </div>
            <div className="flex items-center justify-between py-3 border-b">
              <span className="text-sm font-medium">Voting Start</span>
              <span className="text-sm text-muted-foreground">
                {proposal.voting_start_time
                  ? format(new Date(proposal.voting_start_time), 'PPpp')
                  : 'N/A'}
              </span>
            </div>
            <div className="flex items-center justify-between py-3">
              <span className="text-sm font-medium">Voting End</span>
              <span className="text-sm text-muted-foreground">
                {proposal.voting_end_time
                  ? format(new Date(proposal.voting_end_time), 'PPpp')
                  : 'N/A'}
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Votes List */}
      <Card>
        <CardHeader>
          <CardTitle>Votes</CardTitle>
          <CardDescription>
            {votes.length} vote{votes.length !== 1 ? 's' : ''} recorded
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoadingVotes ? (
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : votes.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Voter</TableHead>
                    <TableHead>Option</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {votes.map((vote: ProposalVote, index: number) => (
                    <TableRow key={`${vote.voter}-${index}`} className="hover:bg-muted/50">
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <User className="h-4 w-4 text-muted-foreground" />
                          <Link href={`/account/${vote.voter}`}>
                            <Button variant="link" className="h-auto p-0 font-mono text-sm">
                              {formatHash(vote.voter, 12, 8)}
                            </Button>
                          </Link>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() => copy(vote.voter)}
                          >
                            {isCopied ? (
                              <CheckCircle className="h-3 w-3 text-green-500" />
                            ) : (
                              <Copy className="h-3 w-3" />
                            )}
                          </Button>
                        </div>
                      </TableCell>
                      <TableCell>
                        {vote.options?.map((opt, i) => (
                          <Badge
                            key={i}
                            variant="outline"
                            className={`${getVoteColor(opt.option_label)} bg-opacity-20`}
                          >
                            {opt.option_label}
                            {opt.weight !== '1.000000000000000000' && ` (${parseFloat(opt.weight) * 100}%)`}
                          </Badge>
                        ))}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <Vote className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No votes recorded yet</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function ProposalSkeleton() {
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
