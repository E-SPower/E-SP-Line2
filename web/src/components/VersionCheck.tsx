import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, X } from 'lucide-react'

// VersionCheck periodically compares the frontend version against the backend
// version. When they differ, it shows a NON-BLOCKING banner at the top of the
// page (no full-screen modal, no "refresh" button) so the user can keep
// working. It only reports a mismatch when the backend is actually reachable
// and returns a version — if the backend is down, ConnectionMonitor handles
// that case and this component stays silent.
export default function VersionCheck() {
  const { t } = useTranslation()
  const [mismatch, setMismatch] = useState(false)

  useEffect(() => {
    const checkVersion = async () => {
      try {
        const response = await fetch('/api/version')
        if (!response.ok) {
          // Backend reachable but no version endpoint — not a version mismatch.
          setMismatch(false)
          return
        }
        const data = await response.json()
        const frontendVersion = '1.0.0'
        const backendVersion = data.backend || data.version
        if (backendVersion && backendVersion !== frontendVersion) {
          setMismatch(true)
        } else {
          setMismatch(false)
        }
      } catch {
        // Backend unreachable — ConnectionMonitor handles that case.
        setMismatch(false)
      }
    }

    checkVersion()
    const interval = setInterval(checkVersion, 30000)
    return () => clearInterval(interval)
  }, [])

  if (!mismatch) return null

  return (
    <div className="fixed top-0 left-0 right-0 z-[90] flex items-center justify-center px-4">
      <div className="mt-3 flex items-center gap-3 bg-amber-50 dark:bg-amber-900/40 border border-amber-200 dark:border-amber-700 rounded-lg shadow-lg px-4 py-2.5 max-w-2xl w-full">
        <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0" />
        <p className="text-sm text-amber-800 dark:text-amber-200 flex-1">
          {t('version.mismatch')}
        </p>
        <button
          onClick={() => setMismatch(false)}
          title={t('common.close')}
          className="text-amber-600 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-200 shrink-0"
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}
