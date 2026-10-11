import { useEffect } from 'react'
import { useAuth } from '@/contexts/auth-context'

export function usePageTitle(title: string) {
  const { siteTitle } = useAuth()
  const brand = siteTitle || 'Kite'

  useEffect(() => {
    const previousTitle = document.title

    if (title) {
      document.title = `${title} - ${brand}`
    }

    return () => {
      document.title = previousTitle
    }
  }, [title, brand])
}
