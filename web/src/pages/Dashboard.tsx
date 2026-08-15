import { useTranslation } from 'react-i18next'
import { Globe, Package, Server, MessageSquare } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'

interface RecentMessage {
  id: string
  sender_id: string
  sender_name?: string
  message_content: string
  created_at: string
}

export default function Dashboard() {
  const { t } = useTranslation()
  const [stats, setStats] = useState([
    { label: t('dashboard.totalPlatforms'), value: '0', icon: Globe, color: 'blue' },
    { label: t('dashboard.totalAdapters'), value: '0', icon: Package, color: 'green' },
    { label: t('dashboard.totalInstances'), value: '0', icon: Server, color: 'purple' },
    { label: t('dashboard.totalMessages'), value: '0', icon: MessageSquare, color: 'orange' },
  ])
  const [backendStatus, setBackendStatus] = useState<string>('checking')
  const [recentMessages, setRecentMessages] = useState<RecentMessage[]>([])

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
      apiClient.getMessages()
    ]).then(([platforms, adapters, instances, messages]) => {
      setStats([
        { label: t('dashboard.totalPlatforms'), value: String(platforms.data?.length || 0), icon: Globe, color: 'blue' },
        { label: t('dashboard.totalAdapters'), value: String(adapters.data?.length || 0), icon: Package, color: 'green' },
        { label: t('dashboard.totalInstances'), value: String(instances.data?.length || 0), icon: Server, color: 'purple' },
        { label: t('dashboard.totalMessages'), value: String(messages.data?.length || 0), icon: MessageSquare, color: 'orange' },
      ])
      setRecentMessages((messages.data || []).slice(0, 5))
    })
  }, [t])

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('dashboard.title')}</h1>
        <div className="flex items-center space-x-2">
          <span className="text-sm text-gray-600">{t('dashboard.systemStatus')}:</span>
          <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
            backendStatus === 'online'
              ? 'bg-green-100 text-green-700'
              : backendStatus === 'offline'
              ? 'bg-red-100 text-red-700'
              : 'bg-gray-100 text-gray-600'
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
            <div key={index} className="bg-white rounded-lg shadow p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600 mb-1">{stat.label}</p>
                  <p className="text-2xl font-bold text-gray-900">{stat.value}</p>
                </div>
                <div className={`p-3 rounded-lg bg-${stat.color}-100`}>
                  <Icon className={`w-6 h-6 text-${stat.color}-600`} />
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {/* Recent Activity - 从 API 加载真实消息记录 */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('dashboard.recentActivity')}</h2>
        {recentMessages.length === 0 ? (
          <p className="text-sm text-gray-500">{t('dashboard.noActivity')}</p>
        ) : (
          <div className="space-y-3">
            {recentMessages.map((message) => (
              <div key={message.id} className="flex items-center text-sm text-gray-600">
                <div className="w-2 h-2 bg-blue-500 rounded-full mr-3"></div>
                <span>{t('dashboard.newMessage')}: {message.sender_name || message.sender_id}</span>
                <span className="ml-auto text-gray-400">
                  {new Date(message.created_at).toLocaleString()}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
