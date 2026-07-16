import { useTranslation } from 'react-i18next'
import { Globe, Package, Server, MessageSquare } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'

export default function Dashboard() {
  const { t } = useTranslation()
  const [stats, setStats] = useState([
    { label: t('dashboard.totalPlatforms'), value: '0', icon: Globe, color: 'blue' },
    { label: t('dashboard.totalAdapters'), value: '0', icon: Package, color: 'green' },
    { label: t('dashboard.totalInstances'), value: '0', icon: Server, color: 'purple' },
    { label: t('dashboard.totalMessages'), value: '0', icon: MessageSquare, color: 'orange' },
  ])
  const [backendStatus, setBackendStatus] = useState<string>('checking')

  useEffect(() => {
    // Check backend health
    apiClient.healthCheck().then(health => {
      setBackendStatus(health.status === 'ok' ? 'online' : 'offline')
    })

    // Load stats from API
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
    })
  }, [t])

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">{t('dashboard.title')}</h1>
      
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

      {/* Recent Activity */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('dashboard.recentActivity')}</h2>
        <div className="space-y-3">
          <div className="flex items-center text-sm text-gray-600">
            <div className="w-2 h-2 bg-green-500 rounded-full mr-3"></div>
            <span>闲鱼接入器实例 "shop-001" 已启动</span>
            <span className="ml-auto text-gray-400">2 分钟前</span>
          </div>
          <div className="flex items-center text-sm text-gray-600">
            <div className="w-2 h-2 bg-blue-500 rounded-full mr-3"></div>
            <span>收到新消息：来自用户 "buyer123"</span>
            <span className="ml-auto text-gray-400">5 分钟前</span>
          </div>
          <div className="flex items-center text-sm text-gray-600">
            <div className="w-2 h-2 bg-purple-500 rounded-full mr-3"></div>
            <span>路由规则 "default-route" 已更新</span>
            <span className="ml-auto text-gray-400">1 小时前</span>
          </div>
        </div>
      </div>
    </div>
  )
}
