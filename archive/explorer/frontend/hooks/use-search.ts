import { useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { api, SearchResult } from '@/lib/api'
import { parseSearchQuery } from '@/lib/utils'

export function useSearch() {
  const router = useRouter()
  const [results, setResults] = useState<SearchResult[]>([])
  const [isSearching, setIsSearching] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const search = useCallback(
    async (query: string) => {
      if (!query || query.trim().length === 0) {
        setResults([])
        setError(null)
        return
      }

      setIsSearching(true)
      setError(null)

      try {
        // First, try to determine the type of search query
        const queryType = parseSearchQuery(query.trim())

        // If we can determine the type, navigate directly
        switch (queryType) {
          case 'block':
            router.push(`/block/${query.trim()}`)
            setResults([])
            setIsSearching(false)
            return
          case 'transaction':
            router.push(`/tx/${query.trim()}`)
            setResults([])
            setIsSearching(false)
            return
          case 'address':
            router.push(`/account/${query.trim()}`)
            setResults([])
            setIsSearching(false)
            return
        }

        // Otherwise, perform a general search
        const response = await api.search(query.trim())
        setResults(response.results || [])

        // If only one result, navigate to it
        if (response.results && response.results.length === 1) {
          const result = response.results[0]
          switch (result.type) {
            case 'block':
              router.push(`/block/${result.id}`)
              break
            case 'transaction':
              router.push(`/tx/${result.id}`)
              break
            case 'address':
              router.push(`/account/${result.id}`)
              break
            case 'validator':
              router.push(`/validator/${result.id}`)
              break
            case 'pool':
              router.push(`/dex/pool/${result.id}`)
              break
          }
          setResults([])
        }
      } catch (err) {
        console.error('Search error:', err)
        setError(err instanceof Error ? err.message : 'Search failed')
        setResults([])
      } finally {
        setIsSearching(false)
      }
    },
    [router]
  )

  const clearSearch = useCallback(() => {
    setResults([])
    setError(null)
  }, [])

  return {
    search,
    clearSearch,
    results,
    isSearching,
    error,
  }
}
