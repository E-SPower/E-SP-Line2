import { useTranslation } from 'react-i18next'
import { Activity, AlertCircle, CheckCircle, Clock, Wifi, WifiOff } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'

interface AdapterStatus {
  id: string
  name: string
  platform: string
  status: 'running' | 'stopped' | 'error'
  connectedAt?: string
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
  messagesPerMinute: number
  avgResponseTime: number
}

export default function Monitoring() {
  const { t } = useTranslation()
  const [adapters, setAdapters] = useState<AdapterStatus[]>([])
  const [stats, setStats] = useState<SystemStats>({
    totalAdapters: 0,
    runningAdapters: 0,
    stoppedAdapters: 0,
    errorAdapters: 0,
    totalMessages: 0,
    messagesPerMinute: 0,
    avgResponseTime: 0
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadMonitoringData()
    const interval = setInterval(loadMonitoringData, 5000) // Refresh every 5 seconds
    return () => clearInterval(interval)
  }, [])

  const loadMonitoringData = async () => {
    try {
      // Load adapter instances
      const instancesRes = await apiClient.getInstances()
      if (instancesRes.data) {
        const adapterStatuses: AdapterStatus[] = instancesRes.data.map((instance: any) => ({
          id: instance.id,
          name: instance.name,
          platform: instance.platform_id,
          status: instance.status,
          connectedAt: instance.connected_at,
          lastHeartbeat: instance.last_heartbeat,
          messageCount: instance.message_count || 0,
          errorCount: instance.error_count || 0
        }))
        setAdapters(adapterStatuses)

        // Calculate stats
        const running = adapterStatuses.filter(a => a.status === 'running').length
        const stopped = adapterStatuses.filter(a => a.status === 'stopped').length
        const errors = adapterStatuses.filter(a => a.status === 'error').length
        const totalMessages = adapterStatuses.reduce((sum, a) => sum + a.messageCount, 0)

        setStats({
          totalAdapters: adapterStatuses.length,
          runningAdapters: running,
          stoppedAdapters: stopped,
          errorAdapters: errors,
          totalMessages,
          messagesPerMinute: Math.round(totalMessages / 60),
          avgResponseTime: 150 // Mock value
        })
      }
    } catch (error) {
      console.error('Failed to load monitoring data:', error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="w-5 h-5 text-green-500" />
      case 'stopped':
        return <WifiOff className="w-5 h-5 text-gray-400" />
      case 'error':
        return <AlertCircle className="w-5 h-5 text-red-500" />
      default:
        return <Clock className="w-5 h-5 text-yellow-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-green-50 border-green-200'
      case 'stopped':
        return 'bg-gray-50 border-gray-200'
      case 'error':
        return 'bg-red-50 border-red-200'
      default:
        return 'bg-yellow-50 border-yellow-200'
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">{t('monitoring.title')}</h1>
        <button
          onClick={loadMonitoringData}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          {t('common.refresh')}
        </button>
      </div>

      {/* System Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">{t('monitoring.totalAdapters')}</p>
              <p className="text-2xl font-bold text-gray-900">{stats.totalAdapters}</p>
            </div>
            <Activity className="w-8 h-8 text-blue-500" />
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">{t('monitoring.runningAdapters')}</p>
              <p className="text-2xl font-bold text-green-600">{stats.runningAdapters}</p>
            </div>
            <Wifi className="w-8 h-8 text-green-500" />
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">{t('monitoring.totalMessages')}</p>
              <p className="text-2xl font-bold text-gray-900">{stats.totalMessages}</p>
            </div>
            <Activity className="w-8 h-8 text-purple-500" />
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 mb-1">{t('monitoring.avgResponseTime')}</p>
              <p className="text-2xl font-bold text-gray-900">{stats.avgResponseTime}ms</p>
            </div>
            <Clock className="w-8 h-8 text-orange-500" />
          </div>
        </div>
      </div>

      {/* Adapter Status List */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">{t('monitoring.adapterStatus')}</h2>
        </div>
        <div className="divide-y divide-gray-200">
          {adapters.length === 0 ? (
            <div className="px-6 py-12 text-center text-gray-500">
              {t('monitoring.noAdapters')}
            </div>
          ) : (
            adapters.map(adapter => (
              <div key={adapter.id} className={`px-6 py-4 ${getStatusColor(adapter.status)}`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-4">
                    {getStatusIcon(adapter.status)}
                    <div>
                      <h3 className="font-medium text-gray-900">{adapter.name}</h3>
                      <p className="text-sm text-gray-500">
                        {t(`platforms.${adapter.platform}`)} • {adapter.id}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="flex items-center space-x-6 text-sm">
                      <div>
                        <p className="text-gray-600">{t('monitoring.messages')}</p>
                        <p className="font-semibold text-gray-900">{adapter.messageCount}</p>
                      </div>
                      <div>
                        <p className="text-gray-600">{t('monitoring.errors')}</p>
                        <p className="font-semibold text-gray-900">{adapter.errorCount}</p>
                      </div>
                      {adapter.lastHeartbeat && (
                        <div>
                          <p className="text-gray-600">{t('monitoring.lastHeartbeat')}</p>
                          <p className="font-semibold text-gray-900">
                            {new Date(adapter.lastHeartbeat).toLocaleTimeString()}
                          </p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Real-time Metrics */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('monitoring.realtimeMetrics')}</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border border-gray-200 rounded-lg p-4">
            <p className="text-sm text-gray-600 mb-2">{t('monitoring.messagesPerMinute')}</p>
            <p className="text-3xl font-bold text-blue-600">{stats.messagesPerMinute}</p>
          </div>
          <div className="border border-gray-200 rounded-lg p-4">
            <p className="text-sm text-gray-600 mb-2">{t('monitoring.errorRate')}</p>
            <p className="text-3xl font-bold text-red-600">
              {stats.totalMessages > 0 
                ? ((adapters.reduce((sum, a) => sum + a.errorCount, 0) / stats.totalMessages) * 100).toFixed(2)
                : 0}%
            </p>
          </div>
          <div className="border border-gray-200 rounded-lg p-4">
            <p className="text-sm text-gray-600 mb-2">{t('monitoring.uptime')}</p>
            <p className="text-3xl font-bold text-green-600">99.9%</p>
          </div>
        </div>
      </div>
    </div>
  )
}
