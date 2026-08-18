import { useTranslation } from 'react-i18next'
import { Globe, Package, Server, MessageSquare, AlertCircle, RefreshCw } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import apiClient from '../api/client'

interface RecentMessage {
  id: string
  sender_id: string
  sender_name?: string
  message_content: string
  created_at: string
}

interface DashboardInstance {
  id: string
  name: string
  adapter_id: string
  platform_id: string
  status: string
  config: string
  created_at: string
}

interface CatalogAdapter {
  id: string
  platform_code: string
  name: string
}

// Status priority for sorting: errors first, then initializing, then running.
function statusPriority(status: string): number {
  switch (status) {
    case 'error':
      return 0
    case 'initializing':
      return 1
    case 'running':
      return 2
    default:
      return 3
  }
}

function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  switch (status) {
    case 'error':
      return (
        <span className="inline-flex items-center px-2 py-1 text-xs font-medium rounded-full bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300">
          <AlertCircle className="w-3 h-3 mr-1" />
          {t('instances.statusError') || '异常'}
        </span>
      )
    case 'initializing':
      return (
        <span className="inline-flex items-center px-2 py-1 text-xs font-medium rounded-full bg-yellow-100 dark:bg-yellow-900/40 text-yellow-700 dark:text-yellow-300">
          <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
          {t('instances.initializing') || '初始化中'}
        </span>
      )
    case 'running':
      return (
        <span className="inline-flex items-center px-2 py-1 text-xs font-medium rounded-full bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300">
          <span className="w-2 h-2 rounded-full bg-green-500 mr-1"></span>
          {t('adapters.running')}
        </span>
      )
    default:
      return (
        <span className="inline-flex items-center px-2 py-1 text-xs font-medium rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300">
          <span className="w-2 h-2 rounded-full bg-gray-400 mr-1"></span>
          {t('adapters.stopped')}
        </span>
      )
  }
}

export default function Dashboard() {
  const { t, i18n } = useTranslation()
  const [stats, setStats] = useState([
    { label: t('dashboard.totalPlatforms'), value: '0', icon: Globe, color: 'blue' },
    { label: t('dashboard.totalAdapters'), value: '0', icon: Package, color: 'green' },
    { label: t('dashboard.totalInstances'), value: '0', icon: Server, color: 'purple' },
    { label: t('dashboard.totalMessages'), value: '0', icon: MessageSquare, color: 'orange' },
  ])
  const [backendStatus, setBackendStatus] = useState<string>('checking')
  const [recentMessages, setRecentMessages] = useState<RecentMessage[]>([])
  const [instances, setInstances] = useState<DashboardInstance[]>([])
  const [adapters, setAdapters] = useState<Record<string, string>>({})
  const [refreshKey, setRefreshKey] = useState(0)

  useEffect(() => {
    // Check backend health
    apiClient.healthCheck().then(health => {
      setBackendStatus(health.status === 'ok' ? 'online' : 'offline')
    })

    // Load stats and recent activity from API
    Promise.all([
      apiClient.getPlatforms(),
      apiClient.getAdapters(),
      apiClient.getInstances(),
      apiClient.getMessages(),
      apiClient.getAdapterCatalog(),
    ]).then(([platforms, adaptersResp, instancesResp, messages, catalog]) => {
      setStats([
        { label: t('dashboard.totalPlatforms'), value: String(platforms.data?.length || 0), icon: Globe, color: 'blue' },
        { label: t('dashboard.totalAdapters'), value: String(adaptersResp.data?.length || 0), icon: Package, color: 'green' },
        { label: t('dashboard.totalInstances'), value: String(instancesResp.data?.length || 0), icon: Server, color: 'purple' },
        { label: t('dashboard.totalMessages'), value: String(messages.data?.length || 0), icon: MessageSquare, color: 'orange' },
      ])
      setRecentMessages((messages.data || []).slice(0, 5))

      const catalogList: CatalogAdapter[] = catalog.data || []
      const adapterMap: Record<string, string> = {}
      catalogList.forEach((a) => {
        adapterMap[a.id] = a.name
      })
      setAdapters(adapterMap)

      // Sort instances: errors first, then initializing, then running.
      const sorted = [...(instancesResp.data || [])].sort(
        (a, b) => statusPriority(a.status) - statusPriority(b.status)
      )
      setInstances(sorted)
    })
  }, [t, refreshKey])

  const errorCount = instances.filter((i) => i.status === 'error').length
  const runningCount = instances.filter((i) => i.status === 'running').length
  const initializingCount = instances.filter((i) => i.status === 'initializing').length

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('dashboard.title')}</h1>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setRefreshKey((k) => k + 1)}
            className="flex items-center px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <RefreshCw className="w-4 h-4 mr-1" />
            {t('common.refresh')}
          </button>
          <span className="text-sm text-gray-600 dark:text-gray-300">{t('dashboard.systemStatus')}:</span>
          <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
            backendStatus === 'online'
              ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300'
              : backendStatus === 'offline'
              ? 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300'
              : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300'
          }`}>
            <span className={`w-2 h-2 rounded-full mr-2 ${
              backendStatus === 'online'
                ? 'bg-green-500'
                : backendStatus === 'offline'
                ? 'bg-red-500'
                : 'bg-gray-400 animate-pulse'
            }`}></span>
            {backendStatus === 'online'
              ? t('dashboard.statusOnline')
              : backendStatus === 'offline'
              ? t('dashboard.statusOffline')
              : t('dashboard.statusChecking')}
          </span>
        </div>
      </div>
      
      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {stats.map((stat, index) => {
          const Icon = stat.icon
          return (
            <div key={index} className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">{stat.label}</p>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stat.value}</p>
                </div>
                <div className={`p-3 rounded-lg bg-${stat.color}-100 dark:bg-gray-700`}>
                  <Icon className={`w-6 h-6 text-${stat.color}-600`} />
                </div>
              </div>
            </div>
          )
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Instance Activity Status - 出错优先 */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('dashboard.instanceStatus')}</h2>
            <Link to="/instances" className="text-sm text-blue-600 hover:text-blue-800">
              {t('dashboard.viewAll')} →
            </Link>
          </div>

          <div className="flex flex-wrap gap-3 mb-4">
            <span className="px-2.5 py-1 text-xs font-medium rounded-full bg-red-50 dark:bg-red-900/40 text-red-700 dark:text-red-300">
              {t('instances.statusError') || '异常'}: {errorCount}
            </span>
            <span className="px-2.5 py-1 text-xs font-medium rounded-full bg-green-50 dark:bg-green-900/40 text-green-700 dark:text-green-300">
              {t('adapters.running')}: {runningCount}
            </span>
            <span className="px-2.5 py-1 text-xs font-medium rounded-full bg-yellow-50 dark:bg-yellow-900/40 text-yellow-700 dark:text-yellow-300">
              {t('instances.initializing') || '初始化中'}: {initializingCount}
            </span>
          </div>

          {instances.length === 0 ? (
            <p className="text-sm text-gray-500">{t('dashboard.noInstances')}</p>
          ) : (
            <div className="space-y-2 max-h-72 overflow-y-auto">
              {instances.map((instance) => (
                <Link
                  key={instance.id}
                  to="/instances"
                  className="flex items-center justify-between p-3 rounded-lg border border-gray-100 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                >
                  <div className="flex items-center min-w-0">
                    <div>
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{instance.name}</p>
                      <p className="text-xs text-gray-400 dark:text-gray-500 truncate">
                        {adapters[instance.adapter_id] || instance.adapter_id}
                      </p>
                    </div>
                  </div>
                  <StatusBadge status={instance.status} />
                </Link>
              ))}
            </div>
          )}
        </div>

        {/* Recent Activity - 从 API 加载真实消息记录 */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{t('dashboard.recentActivity')}</h2>
          {recentMessages.length === 0 ? (
            <p className="text-sm text-gray-500">{t('dashboard.noActivity')}</p>
          ) : (
            <div className="space-y-3">
              {recentMessages.map((message) => (
                <div key={message.id} className="flex items-center text-sm text-gray-600 dark:text-gray-300">
                  <div className="w-2 h-2 bg-blue-500 rounded-full mr-3"></div>
                  <span>{t('dashboard.newMessage')}: {message.sender_name || message.sender_id}</span>
                  <span className="ml-auto text-gray-400 dark:text-gray-500">
                    {new Date(message.created_at).toLocaleString(i18n.language)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
