'use client'

import Link from 'next/link'
import { Box, FileText, User, Droplet, Shield } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { SearchResult } from '@/lib/api'
import { formatHash, formatNumber } from '@/lib/utils'

interface SearchResultsProps {
  results: SearchResult[]
  onSelect?: () => void
}

export function SearchResults({ results, onSelect }: SearchResultsProps) {
  const getIcon = (type: string) => {
    switch (type) {
      case 'block':
        return <Box className="h-5 w-5 text-primary" />
      case 'transaction':
        return <FileText className="h-5 w-5 text-blue-500" />
      case 'address':
        return <User className="h-5 w-5 text-green-500" />
      case 'validator':
        return <Shield className="h-5 w-5 text-purple-500" />
      case 'pool':
        return <Droplet className="h-5 w-5 text-cyan-500" />
      default:
        return <FileText className="h-5 w-5 text-muted-foreground" />
    }
  }

  const getLink = (result: SearchResult) => {
    switch (result.type) {
      case 'block':
        return `/block/${result.id}`
      case 'transaction':
        return `/tx/${result.id}`
      case 'address':
        return `/account/${result.id}`
      case 'validator':
        return `/validator/${result.id}`
      case 'pool':
        return `/dex/pool/${result.id}`
      default:
        return '/'
    }
  }

  const getTitle = (result: SearchResult) => {
    switch (result.type) {
      case 'block':
        return `Block #${formatNumber(parseInt(result.id))}`
      case 'transaction':
        return `Transaction ${formatHash(result.id)}`
      case 'address':
        return `Address ${formatHash(result.id, 10, 8)}`
      case 'validator':
        return result.data?.moniker || `Validator ${formatHash(result.id, 10, 8)}`
      case 'pool':
        return `Pool ${result.id}`
      default:
        return result.id
    }
  }

  const getDescription = (result: SearchResult) => {
    switch (result.type) {
      case 'block':
        return `${result.data?.tx_count || 0} transactions`
      case 'transaction':
        return result.data?.status || 'Transaction'
      case 'address':
        return `${result.data?.tx_count || 0} transactions`
      case 'validator':
        return `Voting power: ${formatNumber(result.data?.voting_power || 0)}`
      case 'pool':
        return `TVL: ${result.data?.tvl || '0'}`
      default:
        return ''
    }
  }

  return (
    <Card className="absolute top-full mt-2 w-full z-50 max-h-96 overflow-y-auto shadow-lg">
      <CardContent className="p-2">
        <div className="space-y-1">
          {results.map((result, index) => (
            <Link key={index} href={getLink(result)} onClick={onSelect}>
              <div className="flex items-center gap-3 p-3 rounded-md hover:bg-accent transition-colors cursor-pointer">
                {getIcon(result.type)}
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-sm truncate">{getTitle(result)}</p>
                  <p className="text-xs text-muted-foreground">{getDescription(result)}</p>
                </div>
                <Badge variant="outline" className="text-xs">
                  {result.type}
                </Badge>
              </div>
            </Link>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
