import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import BigNumber from 'bignumber.js'
import numeral from 'numeral'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// Format numbers with commas
export function formatNumber(value: number | string): string {
  if (typeof value === 'string') {
    const num = parseFloat(value)
    if (isNaN(num)) return value
    return numeral(num).format('0,0')
  }
  return numeral(value).format('0,0')
}

// Format currency values
export function formatCurrency(value: string | number, decimals: number = 2): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '$0.00'

  if (num >= 1e9) {
    return `$${numeral(num).format('0.00a').toUpperCase()}`
  } else if (num >= 1e6) {
    return `$${numeral(num).format('0.00a').toUpperCase()}`
  } else if (num >= 1e3) {
    return `$${numeral(num).format('0,0.00')}`
  }

  return `$${numeral(num).format('0,0.' + '0'.repeat(decimals))}`
}

// Format hash to short form
export function formatHash(hash: string, startChars: number = 8, endChars: number = 6): string {
  if (!hash || hash.length <= startChars + endChars) return hash
  return `${hash.slice(0, startChars)}...${hash.slice(-endChars)}`
}

// Format address to short form
export function formatAddress(address: string, startChars: number = 10, endChars: number = 8): string {
  if (!address || address.length <= startChars + endChars) return address
  return `${address.slice(0, startChars)}...${address.slice(-endChars)}`
}

// Format gas amounts
export function formatGas(gas: number | string): string {
  const gasNum = typeof gas === 'string' ? parseInt(gas) : gas
  if (isNaN(gasNum)) return '0'

  if (gasNum >= 1e6) {
    return numeral(gasNum / 1e6).format('0.00') + 'M'
  } else if (gasNum >= 1e3) {
    return numeral(gasNum / 1e3).format('0.00') + 'K'
  }

  return numeral(gasNum).format('0,0')
}

// Format token amounts with proper decimals
export function formatToken(amount: string | number, decimals: number = 6, symbol?: string): string {
  const bn = new BigNumber(amount)
  const divisor = new BigNumber(10).pow(decimals)
  const formatted = bn.dividedBy(divisor).toFixed(decimals)

  // Remove trailing zeros
  const cleaned = parseFloat(formatted).toString()

  return symbol ? `${cleaned} ${symbol}` : cleaned
}

// Format percentage
export function formatPercent(value: number | string, decimals: number = 2): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '0%'
  return `${numeral(num).format('0,0.' + '0'.repeat(decimals))}%`
}

// Format basis points (1 bp = 0.01%)
export function formatBps(value: number | string, decimals: number = 1): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '0 bps'
  return `${numeral(num).format('0,0.' + '0'.repeat(decimals))} bps`
}

// Format APR/APY
export function formatAPR(apr: string | number): string {
  const num = typeof apr === 'string' ? parseFloat(apr) : apr
  if (isNaN(num)) return '0%'
  return formatPercent(num * 100, 2)
}

// Format time duration
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
  return `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`
}

// Format bytes
export function formatBytes(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return '0 Bytes'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

// Truncate text with ellipsis
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength - 3) + '...'
}

// Copy to clipboard
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    } else {
      // Fallback for older browsers
      const textArea = document.createElement('textarea')
      textArea.value = text
      textArea.style.position = 'fixed'
      textArea.style.left = '-999999px'
      document.body.appendChild(textArea)
      textArea.select()
      try {
        document.execCommand('copy')
        textArea.remove()
        return true
      } catch (error) {
        console.error('Failed to copy:', error)
        textArea.remove()
        return false
      }
    }
  } catch (error) {
    console.error('Failed to copy to clipboard:', error)
    return false
  }
}

// Validate address format
export function isValidAddress(address: string): boolean {
  // Basic validation for Cosmos addresses (bech32)
  const bech32Regex = /^[a-z0-9]+1[a-z0-9]{38,58}$/
  return bech32Regex.test(address)
}

// Validate transaction hash format
export function isValidTxHash(hash: string): boolean {
  // Hex hash validation (64 characters)
  const hexRegex = /^[A-Fa-f0-9]{64}$/
  return hexRegex.test(hash)
}

// Validate block height
export function isValidBlockHeight(height: string): boolean {
  const num = parseInt(height)
  return !isNaN(num) && num > 0
}

// Parse search query type
export function parseSearchQuery(query: string): 'block' | 'transaction' | 'address' | 'unknown' {
  if (isValidBlockHeight(query)) return 'block'
  if (isValidTxHash(query)) return 'transaction'
  if (isValidAddress(query)) return 'address'
  return 'unknown'
}

// Get status color
export function getStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'success':
    case 'confirmed':
    case 'active':
      return 'text-green-500'
    case 'pending':
    case 'processing':
      return 'text-yellow-500'
    case 'failed':
    case 'error':
    case 'rejected':
      return 'text-red-500'
    case 'inactive':
    case 'jailed':
      return 'text-gray-500'
    default:
      return 'text-blue-500'
  }
}

// Get status badge variant
export function getStatusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status.toLowerCase()) {
    case 'success':
    case 'confirmed':
    case 'active':
      return 'default'
    case 'pending':
    case 'processing':
      return 'secondary'
    case 'failed':
    case 'error':
    case 'rejected':
      return 'destructive'
    default:
      return 'outline'
  }
}

// Format voting power
export function formatVotingPower(power: number | string, total?: number): string {
  const num = typeof power === 'string' ? parseFloat(power) : power
  if (isNaN(num)) return '0'

  const formatted = formatNumber(num)

  if (total && total > 0) {
    const percentage = (num / total) * 100
    return `${formatted} (${formatPercent(percentage, 2)})`
  }

  return formatted
}

// Calculate price change
export function calculatePriceChange(current: number, previous: number): { value: number; percentage: number } {
  const value = current - previous
  const percentage = previous > 0 ? (value / previous) * 100 : 0
  return { value, percentage }
}

// Format price change
export function formatPriceChange(current: number, previous: number): string {
  const { value, percentage } = calculatePriceChange(current, previous)
  const sign = value >= 0 ? '+' : ''
  return `${sign}${formatCurrency(value)} (${sign}${formatPercent(percentage, 2)})`
}

// Generate random color for charts
export function generateColor(index: number): string {
  const colors = [
    '#3b82f6', // blue
    '#10b981', // green
    '#f59e0b', // amber
    '#ef4444', // red
    '#8b5cf6', // violet
    '#ec4899', // pink
    '#06b6d4', // cyan
    '#f97316', // orange
  ]
  return colors[index % colors.length]
}

// Debounce function
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null

  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null
      func(...args)
    }

    if (timeout) {
      clearTimeout(timeout)
    }
    timeout = setTimeout(later, wait)
  }
}

// Throttle function
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean

  return function executedFunction(...args: Parameters<T>) {
    if (!inThrottle) {
      func(...args)
      inThrottle = true
      setTimeout(() => (inThrottle = false), limit)
    }
  }
}

// Format relative time with more precision
export function formatRelativeTime(date: Date | string): string {
  const now = new Date()
  const then = typeof date === 'string' ? new Date(date) : date
  const diffInSeconds = Math.floor((now.getTime() - then.getTime()) / 1000)

  if (diffInSeconds < 60) return `${diffInSeconds} seconds ago`
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)} minutes ago`
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)} hours ago`
  if (diffInSeconds < 2592000) return `${Math.floor(diffInSeconds / 86400)} days ago`
  if (diffInSeconds < 31536000) return `${Math.floor(diffInSeconds / 2592000)} months ago`
  return `${Math.floor(diffInSeconds / 31536000)} years ago`
}
