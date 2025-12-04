'use client'

import { useState, useEffect, useRef } from 'react'
import { Search, Loader2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { SearchResults } from './search-results'
import { useSearch } from '@/hooks/use-search'
import { debounce } from '@/lib/utils'

interface SearchBarProps {
  placeholder?: string
  className?: string
  autoFocus?: boolean
}

export function SearchBar({
  placeholder = 'Search blocks, transactions, addresses...',
  className,
  autoFocus = false,
}: SearchBarProps) {
  const [query, setQuery] = useState('')
  const [showResults, setShowResults] = useState(false)
  const searchRef = useRef<HTMLDivElement>(null)
  const { search, results, isSearching, clearSearch } = useSearch()

  // Debounced search function
  const debouncedSearch = useRef(
    debounce((q: string) => {
      if (q.trim().length >= 3) {
        search(q)
      }
    }, 300)
  ).current

  useEffect(() => {
    if (query.trim().length >= 3) {
      debouncedSearch(query)
      setShowResults(true)
    } else {
      clearSearch()
      setShowResults(false)
    }
  }, [query, debouncedSearch, clearSearch])

  // Close results when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(event.target as Node)) {
        setShowResults(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      search(query.trim())
    }
  }

  return (
    <div ref={searchRef} className={className}>
      <form onSubmit={handleSubmit} className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          type="text"
          placeholder={placeholder}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => query.length >= 3 && setShowResults(true)}
          className="pl-10 pr-10"
          autoFocus={autoFocus}
        />
        {isSearching && (
          <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        )}
      </form>

      {showResults && results.length > 0 && (
        <SearchResults
          results={results}
          onSelect={() => {
            setShowResults(false)
            setQuery('')
          }}
        />
      )}
    </div>
  )
}
