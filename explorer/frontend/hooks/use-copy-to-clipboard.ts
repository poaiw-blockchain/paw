import { useState, useCallback } from 'react'
import { copyToClipboard } from '@/lib/utils'

export function useCopyToClipboard() {
  const [isCopied, setIsCopied] = useState(false)

  const copy = useCallback(async (text: string) => {
    const success = await copyToClipboard(text)
    setIsCopied(success)

    if (success) {
      // Reset after 2 seconds
      setTimeout(() => {
        setIsCopied(false)
      }, 2000)
    }

    return success
  }, [])

  return { copy, isCopied }
}
