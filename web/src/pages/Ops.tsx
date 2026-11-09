import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Activity, AlertCircle, CheckCircle2, Clock, MonitorPlay, RefreshCw, Wifi, WifiOff } from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { Card, CardBody, CardHeader, CardTitle } from '../components/ui/card'
import { StatCard, StatGrid } from '../components/ui/stat-card'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { EmptyState } from '../components/ui/empty-state'
import { ErrorState } from '../components/ui/error-state'
import { Skeleton, SkeletonRows } from '../components/ui/skeleton'
import { cn } from '@/lib/cn'

interface AdapterStatus {
  id: string
  name: string
  platform: string
  status: 'running' | 'stopped' | 'error'
  lastHeartbeat?: string
  messageCount: number
  errorCount: number
}

interface SystemStats {
  totalAdapters: number
  runningAdapters: number
  stoppedAdapters: number
  errorAdapters: number
  totalMessages: number
}

type BackendStatus = 'checking' | 'online' | 'offline'

const REFRESH_MS = 5000

const statusVariant = (status: AdapterStatus['status']) => {
  if (status === 'running') return 'success' as const
  if (status === 'error') return 'danger' as const
  return 'neutral' as const
}

const statusIcon = (status: AdapterStatus['status']) => {
  if (status === 'running') return CheckCircle2
  if (status === 'error') return AlertCircle
  if (status === 'stopped') return WifiOff
  return Clock
}

/**
 * The single ops page.
 *
 * It used to be two surfaces: `/monitoring` (metrics, instance heartbeats,
 * live throughput) and the dashboard's right-rail "system status" card, which
 * repeated the adapter and message totals next to a backend badge. The rail
 * copy is gone and this page owns all of it, so the counts exist once.
 *
 * The backend badge stays in the metric strip's heading row on purpose: it is
 * a status for the numbers next to it ("can these totals be trusted?"), not a
 * metric of its own, and it must not reintroduce a second copy of a count.
 */
export default function Ops() {
  const { t } = useTranslation()
  const [adapters, setAdapters] = useState<AdapterStatus[]>([])
  const [stats, setStats] = useState<SystemStats>({
    totalAdapters: 0,
    runningAdapters: 0,
    stoppedAdapters: 0,
    errorAdapters: 0,
    totalMessages: 0,
  })
  const [backendStatus, setBackendStatus] = useState<BackendStatus>('checking')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null)
  const mounted = useRef(true)

  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  const loadOpsData = useCallback(async () => {
    // Fire the health probe alongside the list call so both land on one tick.
    const [health, result] = await Promise.all([apiClient.healthCheck(), apiClient.getInstances()])
    if (!mounted.current) return

    setBackendStatus(health.status === 'ok' ? 'online' : 'offline')

    if (result.error) {
      setError(result.error)
      setLoading(false)
      return
    }

    setError(null)
    const statuses: AdapterStatus[] = (result.data || []).map((instance: any) => ({
      id: instance.id,
      name: instance.name,
      platform: instance.platform_id,
      status: instance.status === 'running' ? 'running' : instance.status === 'error' ? 'error' : 'stopped',
      lastHeartbeat: instance.last_heartbeat,
      messageCount: instance.message_count || 0,
      errorCount: instance.error_count || 0,
    }))

    setAdapters(statuses)
    setStats({
      totalAdapters: statuses.length,
      runningAdapters: statuses.filter((entry) => entry.status === 'running').length,
      stoppedAdapters: statuses.filter((entry) => entry.status === 'stopped').length,
      errorAdapters: statuses.filter((entry) => entry.status === 'error').length,
      totalMessages: statuses.reduce((sum, entry) => sum + entry.messageCount, 0),
    })
    setUpdatedAt(new Date())
    setLoading(false)
  }, [])

  useEffect(() => {
    void loadOpsData()
    const timer = window.setInterval(() => void loadOpsData(), REFRESH_MS)
    return () => window.clearInterval(timer)
  }, [loadOpsData])

  const totalErrors = adapters.reduce((sum, entry) => sum + entry.errorCount, 0)
  const errorRate = stats.totalMessages > 0 ? ((totalErrors / stats.totalMessages) * 100).toFixed(2) : '0.00'

  const backendBadge =
    backendStatus === 'online'
      ? { variant: 'success' as const, label: t('ops.backendOnline') }
      : backendStatus === 'offline'
        ? { variant: 'danger' as const, label: t('ops.backendOffline') }
        : { variant: 'neutral' as const, label: t('ops.backendChecking') }

  return (
    <>
      <PageHeader
        title={t('ops.title')}
        description={t('ops.subtitle')}
        eyebrow={t('ops.description')}
        actions={
          <Button variant="secondary" size="sm" onClick={() => void loadOpsData()}>
            <RefreshCw />
            {t('common.refresh')}
          </Button>
        }
      />

      {error ? (
        <ErrorState
          title={t('state.loadFailed')}
          description={error || t('state.loadFailedHint')}
          onRetry={loadOpsData}
          retryLabel={t('common.retry')}
        />
      ) : (
        <>
          {/* One metrics block: the four counts live here and nowhere else, and
              the backend badge rides the heading row beside them. */}
          <section aria-labelledby="ops-metrics-heading" className="min-w-0 space-y-2">
            <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
              <h2
                id="ops-metrics-heading"
                className="min-w-0 text-2xs font-semibold uppercase tracking-wider text-text-tertiary"
              >
                {t('ops.metricsHeading')}
              </h2>
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="min-w-0 text-xs text-text-tertiary">{t('ops.backendConnectivity')}</span>
                <Badge variant={backendBadge.variant} dot role="status" aria-live="polite">
                  {backendBadge.label}
                </Badge>
              </div>
            </div>

            {backendStatus === 'offline' && (
              <div className="flex items-start gap-2 rounded-md border border-warning-border bg-warning-bg px-2.5 py-2 text-xs text-warning-fg">
                <WifiOff className="mt-px size-3.5 shrink-0" />
                <span className="min-w-0 break-words">{t('state.loadFailedHint')}</span>
              </div>
            )}

            <StatGrid>
              <StatCard
                label={t('ops.totalAdapters')}
                value={stats.totalAdapters}
                icon={Activity}
                tone="primary"
                loading={loading}
              />
              <StatCard
                label={t('ops.runningAdapters')}
                value={stats.runningAdapters}
                icon={Wifi}
                tone="success"
                loading={loading}
              />
              <StatCard
                label={t('ops.errorAdapters')}
                value={stats.errorAdapters}
                icon={AlertCircle}
                tone="danger"
                loading={loading}
              />
              <StatCard
                label={t('ops.totalMessages')}
                value={stats.totalMessages}
                icon={MonitorPlay}
                tone="info"
                loading={loading}
              />
            </StatGrid>
          </section>

          <Card padding="none">
            <CardHeader className="flex-row items-start justify-between gap-3">
              <div>
                <CardTitle>{t('ops.adapterStatus')}</CardTitle>
                <p className="mt-1 text-xs text-text-tertiary">{t('ops.adapterStatusHint')}</p>
              </div>
              {updatedAt && (
                <span className="tabular shrink-0 text-xs text-text-tertiary">
                  {t('ops.updatedAt', { time: updatedAt.toLocaleTimeString() })}
                </span>
              )}
            </CardHeader>

            {loading ? (
              <SkeletonRows rows={4} columns={4} />
            ) : adapters.length === 0 ? (
              <EmptyState
                icon={MonitorPlay}
                size="page"
                aurora
                title={t('ops.noAdapters')}
                description={t('ops.noAdaptersHint')}
              />
            ) : (
              <ul className="divide-y divide-border">
                {adapters.map((adapter) => {
                  const StatusIcon = statusIcon(adapter.status)
                  return (
                    <li
                      key={adapter.id}
                      className={cn(
                        'flex flex-col gap-2 px-5 py-3 sm:flex-row sm:items-center sm:justify-between',
                        adapter.status === 'error' && 'bg-danger-bg/40',
                      )}
                    >
                      <div className="flex min-w-0 items-center gap-2.5">
                        <StatusIcon
                          aria-hidden
                          className={cn(
                            'size-4 shrink-0',
                            adapter.status === 'running' && 'text-success',
                            adapter.status === 'error' && 'text-danger',
                            adapter.status === 'stopped' && 'text-text-disabled',
                          )}
                        />
                        <div className="min-w-0">
                          <p className="truncate text-sm font-medium text-text-primary">{adapter.name}</p>
                          <p className="truncate font-mono text-xs text-text-tertiary">{adapter.id}</p>
                        </div>
                      </div>

                      <div className="flex flex-wrap items-center gap-4 sm:justify-end">
                        <Metric label={t('ops.messages')} value={adapter.messageCount} />
                        <Metric
                          label={t('ops.errors')}
                          value={adapter.errorCount}
                          tone={adapter.errorCount > 0 ? 'danger' : 'neutral'}
                        />
                        <Metric
                          label={t('ops.lastHeartbeat')}
                          value={adapter.lastHeartbeat ? new Date(adapter.lastHeartbeat).toLocaleTimeString() : '—'}
                        />
                        <Badge variant={statusVariant(adapter.status)} dot>
                          {t(`state.status.${adapter.status}`)}
                        </Badge>
                      </div>
                    </li>
                  )
                })}
              </ul>
            )}
          </Card>

          {/* Default padding, not `padding="none"`: with `none` the card's
              `gap-0` butts the header straight against the body, and at phone
              widths the title collides with the first panel. */}
          <Card>
            <CardHeader>
              <CardTitle>{t('ops.realtimeMetrics')}</CardTitle>
            </CardHeader>
            <CardBody className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <MetricPanel label={t('ops.messagesPerMinute')} tone="info">
                {Math.round(stats.totalMessages / 60)}
              </MetricPanel>
              {/* One interpolated string, not `{errorRate}%`: two children make
                  `children` an array, which `MetricPanel` mistook for the
                  loading placeholder, so the number never rendered. */}
              <MetricPanel label={t('ops.errorRate')} tone="danger">
                {`${errorRate}%`}
              </MetricPanel>
              <MetricPanel label={t('ops.uptime')} tone="success">
                {'99.9%'}
              </MetricPanel>
            </CardBody>
          </Card>
        </>
      )}
    </>
  )
}

function Metric({
  label,
  value,
  tone = 'neutral',
}: {
  label: string
  value: string | number
  tone?: 'neutral' | 'danger'
}) {
  return (
    <div className="min-w-14 text-left sm:text-right">
      <p className="text-2xs uppercase tracking-wide text-text-tertiary">{label}</p>
      <p className={cn('tabular text-sm font-semibold', tone === 'danger' ? 'text-danger-fg' : 'text-text-primary')}>
        {value}
      </p>
    </div>
  )
}

function MetricPanel({
  label,
  tone,
  children,
}: {
  label: string
  tone: 'info' | 'danger' | 'success'
  children: React.ReactNode
}) {
  const toneClass = {
    info: 'text-info-fg',
    danger: 'text-danger-fg',
    success: 'text-success-fg',
  }[tone]

  return (
    <div className="rounded-md border border-border-subtle bg-surface-sunken px-3 py-2.5">
      <p className="text-xs text-text-tertiary">{label}</p>
      {typeof children === 'string' || typeof children === 'number' ? (
        <p className={cn('tabular mt-1 text-2xl font-semibold', toneClass)}>{children}</p>
      ) : (
        <Skeleton shape="line" className="mt-2 h-6 w-16" />
      )}
    </div>
  )
}
