import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Activity,
  ArrowRight,
  GitBranch,
  Globe,
  MessageSquare,
  Server,
  ShieldCheck,
  WifiOff,
} from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { StatCard, StatGrid } from '../components/ui/stat-card'
import { Card, CardBody, CardFooter, CardHeader, CardTitle } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { EmptyState } from '../components/ui/empty-state'
import { Skeleton, SkeletonText } from '../components/ui/skeleton'

interface RecentActivity {
  id: string
  type: 'adapter' | 'message' | 'route'
  description: string
  timestamp: string
}

type BackendStatus = 'checking' | 'online' | 'offline'

const activityDot: Record<RecentActivity['type'], string> = {
  adapter: 'bg-success',
  message: 'bg-info',
  route: 'bg-text-tertiary',
}

export default function Dashboard() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [backendStatus, setBackendStatus] = useState<BackendStatus>('checking')
  const [counts, setCounts] = useState({ platforms: 0, adapters: 0, instances: 0, messages: 0 })
  const [recentActivities, setRecentActivities] = useState<RecentActivity[]>([])
  const [error, setError] = useState<string | null>(null)

  const [reloadToken, setReloadToken] = useState(0)
  const [refreshed, setRefreshed] = useState(false)

  const refresh = useCallback(() => {
    setRefreshed(false)
    setReloadToken((value) => value + 1)
  }, [])

  useEffect(() => {
    let cancelled = false

    apiClient.healthCheck().then((health) => {
      if (cancelled) return
      setBackendStatus(health.status === 'ok' ? 'online' : 'offline')
    })

    Promise.all([
      apiClient.getPlatforms(),
      apiClient.getAdapters(),
      apiClient.getInstances(),
      apiClient.getMessages(),
    ])
      .then(([platforms, adapters, instances, messages]) => {
        if (cancelled) return

        // A failed list call must show up, not silently read as "0".
        const failure = platforms.error || adapters.error || instances.error || messages.error
        setError(failure || null)

        setCounts({
          platforms: platforms.data?.length || 0,
          adapters: adapters.data?.length || 0,
          instances: instances.data?.length || 0,
          messages: messages.data?.length || 0,
        })

        const activities: RecentActivity[] = []

        adapters.data?.slice(0, 3).forEach((adapter: any) => {
          activities.push({
            id: `adapter-${adapter.id}`,
            type: 'adapter',
            description: `${t('dashboard.adapterCreated')}: ${adapter.name}`,
            timestamp: adapter.created_at,
          })
        })

        messages.data?.slice(0, 3).forEach((message: any) => {
          activities.push({
            id: `message-${message.id}`,
            type: 'message',
            description: `${t('dashboard.messageReceived')}: ${message.sender_name || message.sender_id}`,
            timestamp: message.created_at,
          })
        })

        activities.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
        setRecentActivities(activities.slice(0, 5))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [t, reloadToken])

  const stats = [
    { label: t('dashboard.totalPlatforms'), value: counts.platforms, icon: Globe, tone: 'neutral' as const, to: '/platforms' },
    { label: t('dashboard.totalInstances'), value: counts.instances, icon: Server, tone: 'neutral' as const, to: '/instances' },
    { label: t('dashboard.totalMessages'), value: counts.messages, icon: MessageSquare, tone: 'neutral' as const, to: '/messages' },
  ]

  const backendBadge =
    backendStatus === 'online'
      ? { variant: 'success' as const, label: t('dashboard.backendOnline') }
      : backendStatus === 'offline'
        ? { variant: 'danger' as const, label: t('dashboard.backendOffline') }
        : { variant: 'neutral' as const, label: t('dashboard.backendChecking') }

  return (
    <>
      <PageHeader
        title={t('dashboard.title')}
        description={t('dashboard.subtitle')}
        actions={
          <>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                refresh()
                setRefreshed(true)
              }}
              disabled={loading}
            >
              <Activity />
              {t('common.refresh')}
            </Button>
            <Button variant="primary" size="sm" asChild>
              <Link to="/instances?create=1">
                {t('instances.create')}
                <ArrowRight />
              </Link>
            </Button>
          </>
        }
      />

      {error && (
        <p
          role="alert"
          className="flex items-start gap-2 rounded-md border border-danger-border bg-danger-bg px-2.5 py-2 text-xs text-danger-fg"
        >
          <WifiOff className="mt-px size-3.5 shrink-0" />
          <span className="min-w-0 flex-1 break-words">
            {t('state.loadFailed')} · {error}
          </span>
          <button type="button" className="shrink-0 font-medium underline" onClick={refresh}>
            {t('common.retry')}
          </button>
        </p>
      )}

      {refreshed && !loading && !error && (
        <Badge variant="success" role="status" aria-live="polite">
          {t('dashboard.refreshed')}
        </Badge>
      )}

      {/* Metrics — each tile links to its list. Three tracks from `md` all the
          way up: with only three metrics (adapters merged into instances) any
          two-column step lays them out 2+1 and leaves a hole in the second row.
          Tracks are `minmax(0, 1fr)`, so a long word widens the label, never
          the grid; below `md` a tile per row, where three tiles would break
          every label mid-word. */}
      <section aria-labelledby="overview-heading" className="min-w-0 space-y-2">
        <h2 id="overview-heading" className="text-2xs font-semibold uppercase tracking-wider text-text-tertiary">
          {t('dashboard.overview')}
        </h2>
        {/* `lg:` is repeated on purpose: `StatGrid` hard-codes `lg:grid-cols-4`
            and a fourth metric no longer exists. */}
        <StatGrid className="gap-6 grid-cols-1 md:grid-cols-[repeat(3,minmax(0,1fr))] lg:grid-cols-[repeat(3,minmax(0,1fr))]">
          {stats.map((stat) => (
            <Link
              key={stat.to}
              to={stat.to}
              className="min-w-0 break-words rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
            >
              <StatCard
                label={stat.label}
                value={stat.value}
                icon={stat.icon}
                tone={stat.tone}
                loading={loading}
                className="h-full min-w-0 [&>div]:min-w-0 [&>div]:gap-2 [&_p]:min-w-0 [&_p]:break-words max-xl:!px-3"
              />
            </Link>
          ))}
        </StatGrid>
      </section>

      <div className="grid min-w-0 grid-cols-1 items-start gap-6 md:grid-cols-3">
        {/* Activity — the widest, densest block. */}
        <Card padding="none" className="min-w-0 md:col-span-2">
          {/* One column while the card is phone-narrow, then two shrinkable
              tracks: the title may collapse and `运行监控` may wrap instead of
              pushing the header past the card. */}
          <CardHeader className="min-w-0 grid-cols-1 gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,auto)] sm:gap-3">
            <div className="min-w-0">
              <CardTitle>{t('dashboard.recentActivity')}</CardTitle>
              <p className="mt-1 min-w-0 break-words text-xs text-text-tertiary">{t('dashboard.recentActivityHint')}</p>
            </div>
            <Link
              to="/monitoring"
              className="min-w-0 break-words text-xs font-medium text-primary hover:underline sm:justify-self-end sm:text-end"
            >
              {t('nav.monitoring')}
            </Link>
          </CardHeader>
          {loading ? (
            <CardBody>
              <SkeletonText lines={4} />
            </CardBody>
          ) : recentActivities.length === 0 ? (
            <EmptyState
              icon={Activity}
              title={t('dashboard.noRecentActivity')}
              description={t('dashboard.noRecentActivityHint')}
            />
          ) : (
            <ul className="divide-y divide-border">
              {recentActivities.map((activity) => (
                <li key={activity.id} className="flex min-w-0 items-center gap-3 px-5 py-3 text-sm">
                  <span aria-hidden className={`size-1.5 shrink-0 rounded-full ${activityDot[activity.type]}`} />
                  <span className="min-w-0 flex-1 truncate text-text-secondary" title={activity.description}>
                    {activity.description}
                  </span>
                  {/* Yields width rather than starving the description; at big
                      root sizes it wraps to two short lines instead of forcing
                      the row wider. */}
                  <time
                    dateTime={activity.timestamp}
                    className="tabular min-w-0 shrink break-words text-end text-xs text-text-tertiary"
                  >
                    {new Date(activity.timestamp).toLocaleString()}
                  </time>
                </li>
              ))}
            </ul>
          )}
        </Card>

        {/* Right rail — lower density, secondary information. `min-w-0` on the
            rail and wrapping labels let the status rows give way instead of
            pushing a badge out of the card when the column is narrow. */}
        <div className="stack min-w-0">
          <Card padding="none" className="min-w-0">
            <CardHeader>
              <div className="min-w-0">
                <CardTitle>{t('dashboard.systemStatus')}</CardTitle>
                <p className="mt-1 text-xs text-text-tertiary">{t('dashboard.systemStatusHint')}</p>
              </div>
            </CardHeader>
            <CardBody className="space-y-3">
              {/* `flex-wrap` + `break-words`: on a narrow rail the badge drops
                  to the next line instead of squeezing the label down to one
                  letter. No fixed widths, so nothing can escape the card. */}
              <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-2 gap-y-1">
                <span className="min-w-0 break-words text-sm text-text-secondary">
                  {t('dashboard.backendStatus')}
                </span>
                <Badge variant={backendBadge.variant} dot>
                  {backendBadge.label}
                </Badge>
              </div>
              {backendStatus === 'offline' && (
                <div className="flex items-start gap-2 rounded-md border border-warning-border bg-warning-bg px-2.5 py-2 text-xs text-warning-fg">
                  <WifiOff className="mt-px size-3.5 shrink-0" />
                  <span className="min-w-0 break-words">{t('state.loadFailedHint')}</span>
                </div>
              )}
              <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-2 gap-y-1">
                <span className="min-w-0 break-words text-sm text-text-secondary">
                  {t('dashboard.totalAdapters')}
                </span>
                {loading ? (
                  <Skeleton shape="line" className="h-4 w-10" />
                ) : (
                  <Badge variant={counts.adapters > 0 ? 'success' : 'neutral'}>{counts.adapters}</Badge>
                )}
              </div>
              <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-2 gap-y-1">
                <span className="min-w-0 break-words text-sm text-text-secondary">
                  {t('monitoring.totalMessages')}
                </span>
                {loading ? (
                  <Skeleton shape="line" className="h-4 w-10" />
                ) : (
                  <Badge variant="neutral">{counts.messages}</Badge>
                )}
              </div>
            </CardBody>
          </Card>

          <Card padding="none" className="min-w-0">
            <CardHeader>
              <CardTitle>{t('dashboard.quickActions')}</CardTitle>
            </CardHeader>
            <CardBody className="flex flex-col gap-1 p-2">
              {[
                { to: '/platforms', icon: Globe, label: t('nav.platforms') },
                { to: '/routes', icon: GitBranch, label: t('nav.routes') },
                { to: '/messages', icon: ShieldCheck, label: t('messages.ack') },
              ].map((action) => (
                <Link
                  key={action.to}
                  to={action.to}
                  className="flex min-w-0 items-center gap-2 rounded-md px-2.5 py-2 text-sm text-text-secondary transition-colors hover:bg-surface-hover hover:text-text-primary"
                >
                  <action.icon className="size-4 shrink-0 text-text-tertiary" />
                  <span className="min-w-0 flex-1 truncate">{action.label}</span>
                  <ArrowRight className="size-3.5 shrink-0 text-text-disabled" />
                </Link>
              ))}
            </CardBody>
            {/* Version and docs share a row until the rail is too narrow, then
                they stack rather than overhang the card. */}
            <CardFooter className="flex-wrap justify-between gap-x-2 gap-y-1">
              <span className="min-w-0 text-xs text-text-tertiary">{t('app.version')}</span>
              <Link to="/docs" className="min-w-0 text-xs font-medium text-primary hover:underline">
                {t('docs.title')}
              </Link>
            </CardFooter>
          </Card>
        </div>
      </div>
    </>
  )
}
