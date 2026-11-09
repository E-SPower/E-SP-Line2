import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eraser, RefreshCw, Terminal } from 'lucide-react'
import apiClient from '../api/client'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { ConfirmDialog } from './ui/confirm-dialog'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog'
import { controlClassName } from './ui/form-field'
import { SkeletonText } from './ui/skeleton'
import { cn } from '@/lib/cn'

interface LogHeatmapEntry {
  hour: string
  debug: number
  info: number
  warning: number
  error: number
  unmarked: number
}

export interface LogInstance {
  id: string
  name: string
}

const LEVELS = ['', 'debug', 'info', 'warning', 'error'] as const

/**
 * Log drawer for one instance: level filter, ANDed keyword search, time window,
 * bounded line count, hourly level heatmap and a clear action. The lines are the
 * adapter process's own stdout/stderr, not sample data — a process that cannot
 * start writes its error here too.
 */
export function InstanceLogDrawer({ instance, onClose }: { instance: LogInstance | null; onClose: () => void }) {
  const { t } = useTranslation()
  const [logs, setLogs] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [level, setLevel] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [lines, setLines] = useState<number>(2000)
  const [heatmap, setHeatmap] = useState<LogHeatmapEntry[]>([])
  const [clearing, setClearing] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [clearError, setClearError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const fetchLogs = useCallback(
    async (target: LogInstance, filters: { level: string; keyword: string; from: string; to: string; lines: number }) => {
      setLoading(true)
      setError(null)
      const result = await apiClient.getInstanceLogs(target.id, {
        lines: filters.lines,
        level: filters.level || undefined,
        keyword: filters.keyword || undefined,
        from: filters.from || undefined,
        to: filters.to || undefined,
      })
      setLoading(false)
      if (result.error) {
        setError(result.error)
        setLogs('')
        return
      }
      // The API answers `{ data: "<lines>" }` for logs.
      setLogs(typeof result.data === 'string' ? result.data : '')

      const hm = await apiClient.getInstanceLogHeatmap(target.id, filters.level || undefined)
      if (!hm.error) setHeatmap(hm.data || [])
    },
    [],
  )

  useEffect(() => {
    setLevel('')
    setKeyword('')
    setFrom('')
    setTo('')
    setLines(2000)
    setLogs('')
    setHeatmap([])
    setError(null)
    setNotice(null)
    if (!instance) return
    void fetchLogs(instance, { level: '', keyword: '', from: '', to: '', lines: 2000 })
  }, [instance, fetchLogs])

  const reload = (patch: Partial<{ level: string; keyword: string; from: string; to: string; lines: number }> = {}) => {
    if (!instance) return
    void fetchLogs(instance, { level, keyword, from, to, lines, ...patch })
  }

  /**
   * A process started a moment ago has not flushed its first lines yet, so a
   * one-shot read shows an empty log over a live instance. Poll while the
   * drawer is open on a running instance and nothing has arrived, then stop as
   * soon as there is content (or the instance is no longer running).
   */
  useEffect(() => {
    if (!instance || logs !== '' || loading || error) return
    const timer = window.setInterval(() => reload(), 1500)
    return () => window.clearInterval(timer)
    // `reload` closes over the filters listed below; including it would retime
    // the interval on every render, so the filter deps stand in for it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instance, logs, loading, error, level, keyword, from, to, lines])

  // Debounce typing so the backend is not hit per keystroke.
  const keywordTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const changeKeyword = (value: string) => {
    setKeyword(value)
    if (keywordTimer.current) clearTimeout(keywordTimer.current)
    keywordTimer.current = setTimeout(() => reload({ keyword: value }), 300)
  }

  const clearLogs = async () => {
    if (!instance) return
    setClearing(true)
    setClearError(null)
    const result = await apiClient.clearInstanceLogs(instance.id)
    setClearing(false)
    if (result.error) {
      setClearError(result.error)
      return
    }
    setConfirmOpen(false)
    setNotice(t('logs.cleared', { bytes: result.data?.removed_bytes ?? 0 }))
    setLogs('')
    setHeatmap([])
  }

  const lineCount = logs === '' ? 0 : logs.split('\n').length

  return (
    <>
      <Dialog open={Boolean(instance)} onOpenChange={(open) => !open && onClose()}>
        <DialogContent className="w-[calc(100vw-2rem)] max-w-3xl min-w-0 gap-3">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Terminal className="size-4 shrink-0" />
              {t('logs.title')}
            </DialogTitle>
            <DialogDescription>
              {instance ? <span className="font-medium text-text-secondary">{instance.name}</span> : null}{' '}
              <span className="font-mono text-xs text-text-tertiary">{instance?.id}</span>
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-wrap items-end gap-2">
            <div role="group" aria-label={t('logs.level')} className="flex items-center gap-1">
              {LEVELS.map((item) => (
                <button
                  key={item || 'all'}
                  type="button"
                  aria-pressed={level === item}
                  onClick={() => {
                    setLevel(item)
                    reload({ level: item })
                  }}
                  className={cn(
                    'inline-flex h-7 items-center rounded-full border px-2.5 text-xs font-medium transition-colors',
                    level === item
                      ? 'border-transparent bg-primary text-primary-foreground'
                      : 'border-border text-text-tertiary hover:bg-surface-hover hover:text-text-secondary',
                  )}
                >
                  {item === '' ? t('common.all') : item}
                </button>
              ))}
            </div>

            <div className="flex min-w-0 flex-1 flex-wrap items-end gap-2">
              <input
                aria-label={t('logs.keyword')}
                placeholder={t('logs.keywordPlaceholder')}
                value={keyword}
                onChange={(event) => changeKeyword(event.target.value)}
                className={cn(controlClassName, 'h-7 w-40 text-xs')}
              />
              <input
                aria-label={t('logs.from')}
                type="datetime-local"
                value={from}
                onChange={(event) => {
                  setFrom(event.target.value)
                  reload({ from: event.target.value })
                }}
                className={cn(controlClassName, 'h-7 w-44 text-xs')}
              />
              <input
                aria-label={t('logs.to')}
                type="datetime-local"
                value={to}
                onChange={(event) => {
                  setTo(event.target.value)
                  reload({ to: event.target.value })
                }}
                className={cn(controlClassName, 'h-7 w-44 text-xs')}
              />
              <div className="flex items-center gap-1">
                {[200, 2000, 5000].map((value) => (
                  <button
                    key={value}
                    type="button"
                    aria-pressed={lines === value}
                    onClick={() => {
                      setLines(value)
                      reload({ lines: value })
                    }}
                    className={cn(
                      'tabular inline-flex h-7 items-center rounded-full border px-2 text-xs transition-colors',
                      lines === value
                        ? 'border-transparent bg-surface-inset font-medium text-text-primary'
                        : 'border-border text-text-tertiary hover:text-text-secondary',
                    )}
                  >
                    {value}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex items-center gap-1">
              <Button variant="ghost" size="icon-sm" onClick={() => reload()} aria-label={t('common.refresh')} title={t('common.refresh')}>
                <RefreshCw />
              </Button>
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-danger-fg hover:bg-danger-bg"
                onClick={() => {
                  setClearError(null)
                  setConfirmOpen(true)
                }}
                aria-label={t('logs.clear')}
                title={t('logs.clear')}
              >
                <Eraser />
              </Button>
            </div>
          </div>

          {heatmap.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-text-tertiary">{t('logs.heatmapHint')}</span>
              <div className="flex items-end gap-0.5" aria-label={t('logs.heatmap')} role="img">
                {heatmap.map((entry) => {
                  const total = entry.error * 4 + entry.warning * 3 + entry.info * 2 + entry.debug + entry.unmarked
                  return (
                    <span
                      key={entry.hour}
                      title={`${new Date(entry.hour).toLocaleString()} · debug ${entry.debug} / info ${entry.info} / warning ${entry.warning} / error ${entry.error}`}
                      className="w-2 rounded-xs bg-text-tertiary"
                      style={{ height: `${Math.min(24, 4 + total)}px`, opacity: Math.min(1, 0.25 + total / 40) }}
                    />
                  )
                })}
              </div>
            </div>
          )}

          {notice && (
            <Badge variant="success" role="status" aria-live="polite">
              {notice}
            </Badge>
          )}

          <div className="h-72 min-w-0 overflow-hidden rounded-md border border-border bg-surface-sunken">
            {loading ? (
              <div className="p-3">
                <SkeletonText lines={6} />
              </div>
            ) : error ? (
              <p className="p-3 text-sm text-danger-fg">{t('logs.loadFailed', { error })}</p>
            ) : logs === '' ? (
              <p className="p-3 text-sm text-text-tertiary">{t('logs.empty')}</p>
            ) : (
              <pre className="h-full w-full overflow-auto p-3 font-mono text-2xs leading-relaxed text-text-secondary">
                {logs}
              </pre>
            )}
          </div>

          <p className="tabular text-xs text-text-tertiary">{t('logs.lineCount', { count: lineCount })}</p>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={(open) => !clearing && setConfirmOpen(open)}
        title={t('logs.clearTitle')}
        description={t('logs.clearHint')}
        confirmLabel={t('logs.clear')}
        cancelLabel={t('common.cancel')}
        pending={clearing}
        error={clearError}
        onConfirm={() => void clearLogs()}
      />
    </>
  )
}

export default InstanceLogDrawer
