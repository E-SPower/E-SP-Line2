import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { WifiOff } from 'lucide-react'
import apiClient from '../api/client'

// ConnectionMonitor periodically pings the backend health endpoint. When the
// backend becomes unreachable for more than `offlineThreshold` consecutive
// checks, it shows a modal warning that the frontend/backend link is lost.
// Once connectivity is restored, the modal dismisses automatically.
export default function ConnectionMonitor() {
  const { t } = useTranslation()
  const [offline, setOffline] = useState(false)
  const offlineCount = useRef(0)
  const shownOnce = useRef(false)

  useEffect(() => {
    // Don't run while on the login page (no backend interaction required yet).
    if (window.location.pathname === '/login') return

    let cancelled = false
    let timer: ReturnType<typeof setInterval>

    const check = async () => {
      if (cancelled) return
      let ok = false
      try {
        const health = await apiClient.healthCheck()
        ok = health && health.status === 'ok'
      } catch {
        ok = false
      }

      if (ok) {
        offlineCount.current = 0
        if (offline) setOffline(false)
      } else {
        offlineCount.current += 1
        // Show the modal after 3 consecutive failed checks (~9s).
        if (offlineCount.current >= 3) {
          shownOnce.current = true
          setOffline(true)
        }
      }
    }

    check()
    timer = setInterval(check, 3000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Only show after having been online at least once (to avoid flashing the
  // modal before the backend ever became reachable on first load).
  if (!offline) return null

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center">
      <div className="fixed inset-0 bg-black bg-opacity-60" />
      <div className="relative bg-white rounded-lg shadow-2xl p-8 max-w-md w-full mx-4 text-center">
        <div className="mx-auto w-14 h-14 rounded-full bg-red-100 flex items-center justify-center mb-4">
          <WifiOff className="w-7 h-7 text-red-600" />
        </div>
        <h2 className="text-lg font-semibold text-gray-900 mb-2">
          {t('connection.lostTitle')}
        </h2>
        <p className="text-sm text-gray-600 mb-6">
          {t('connection.lostDesc')}
        </p>
        <div className="flex items-center justify-center space-x-2 text-sm text-gray-400">
          <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
          {t('connection.reconnecting')}
        </div>
      </div>
    </div>
  )
}
