import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Activity,
  ArrowRight,
  GitBranch,
  Globe,
  MessageSquare,
  Package,
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
    { label: t('dashboard.totalAdapters'), value: counts.adapters, icon: Package, tone: 'neutral' as const, to: '/adapters' },
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
              <Link to="/adapters?create=1">
                {t('adapters.create')}
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

      {/* Metrics — wide cards, four across, each linking to its list. */}
      <section aria-labelledby="overview-heading" className="space-y-2">
        <h2 id="overview-heading" className="text-2xs font-semibold uppercase tracking-wider text-text-tertiary">
          {t('dashboard.overview')}
        </h2>
        <StatGrid>
          {stats.map((stat) => (
            <Link
              key={stat.to}
              to={stat.to}
              className="rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
            >
              <StatCard
                label={stat.label}
                value={stat.value}
                icon={stat.icon}
                tone={stat.tone}
                loading={loading}
                className="h-full"
              />
            </Link>
          ))}
        </StatGrid>
      </section>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Activity — the widest, densest block. */}
        <Card padding="none" className="lg:col-span-2">
          <CardHeader className="flex-row items-start justify-between gap-3">
            <div className="min-w-0">
              <CardTitle>{t('dashboard.recentActivity')}</CardTitle>
              <p className="mt-1 text-xs text-text-tertiary">{t('dashboard.recentActivityHint')}</p>
            </div>
            <Link to="/monitoring" className="shrink-0 text-xs font-medium text-primary hover:underline">
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
                <li key={activity.id} className="flex items-center gap-3 px-5 py-3 text-sm">
                  <span aria-hidden className={`size-1.5 shrink-0 rounded-full ${activityDot[activity.type]}`} />
                  <span className="min-w-0 flex-1 truncate text-text-secondary">{activity.description}</span>
                  <time dateTime={activity.timestamp} className="tabular shrink-0 text-xs text-text-tertiary">
                    {new Date(activity.timestamp).toLocaleString()}
                  </time>
                </li>
              ))}
            </ul>
          )}
        </Card>

        {/* Right rail — lower density, secondary information. */}
        <div className="stack">
          <Card padding="none">
            <CardHeader>
              <div>
                <CardTitle>{t('dashboard.systemStatus')}</CardTitle>
                <p className="mt-1 text-xs text-text-tertiary">{t('dashboard.systemStatusHint')}</p>
              </div>
            </CardHeader>
            <CardBody className="space-y-3">
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm text-text-secondary">{t('dashboard.backendStatus')}</span>
                <Badge variant={backendBadge.variant} dot>
                  {backendBadge.label}
                </Badge>
              </div>
              {backendStatus === 'offline' && (
                <div className="flex items-start gap-2 rounded-md border border-warning-border bg-warning-bg px-2.5 py-2 text-xs text-warning-fg">
                  <WifiOff className="mt-px size-3.5 shrink-0" />
                  <span>{t('state.loadFailedHint')}</span>
                </div>
              )}
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm text-text-secondary">{t('monitoring.totalAdapters')}</span>
                {loading ? (
                  <Skeleton shape="line" className="h-4 w-10" />
                ) : (
                  <Badge variant={counts.adapters > 0 ? 'success' : 'neutral'}>{counts.adapters}</Badge>
                )}
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm text-text-secondary">{t('monitoring.totalMessages')}</span>
                {loading ? (
                  <Skeleton shape="line" className="h-4 w-10" />
                ) : (
                  <Badge variant="neutral">{counts.messages}</Badge>
                )}
              </div>
            </CardBody>
          </Card>

          <Card padding="none">
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
                  className="flex items-center gap-2 rounded-md px-2.5 py-2 text-sm text-text-secondary transition-colors hover:bg-surface-hover hover:text-text-primary"
                >
                  <action.icon className="size-4 text-text-tertiary" />
                  <span className="flex-1">{action.label}</span>
                  <ArrowRight className="size-3.5 text-text-disabled" />
                </Link>
              ))}
            </CardBody>
            <CardFooter className="justify-between">
              <span className="text-xs text-text-tertiary">{t('app.version')}</span>
              <Link to="/docs" className="text-xs font-medium text-primary hover:underline">
                {t('docs.title')}
              </Link>
            </CardFooter>
          </Card>
        </div>
      </div>
    </>
  )
}
