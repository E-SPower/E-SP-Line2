import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import apiClient from '../api/client'
import { Badge } from './ui/badge'

/**
 * Top-bar backend indicator. Polls `/health` so the badge reflects the real
 * connection instead of a hard-coded "connected".
 */
export function ConnectionBadge() {
  const { t } = useTranslation()
  const [online, setOnline] = useState<boolean | null>(null)

  useEffect(() => {
    let cancelled = false
    const check = () => {
      apiClient.healthCheck().then((health) => {
        if (!cancelled) setOnline(health.status === 'ok')
      })
    }
    check()
    const timer = window.setInterval(check, 30000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [])

  if (online === null) return null

  return (
    <Badge variant={online ? 'success' : 'danger'} dot className="hidden sm:inline-flex">
      {online ? t('app.connected') : t('dashboard.backendOffline')}
    </Badge>
  )
}

export default ConnectionBadge
