import { useTranslation } from 'react-i18next'
import { Plus, Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import apiClient from '../api/client'

interface CatalogAdapter {
  id: string
  platform_code: string
  name: string
  version: string
  runtime_type: string
  description: string
  hidden: boolean
  icon?: string
  capabilities?: string[]
  config_schema: Record<string, any>
}

export default function Adapters() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [adapters, setAdapters] = useState<CatalogAdapter[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadCatalog()
  }, [])

  const loadCatalog = async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getAdapterCatalog()
    if (result.error) {
      setError(result.error)
      setAdapters([])
    } else {
      setAdapters(result.data || [])
    }
    setLoading(false)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-6 h-6 text-blue-500 animate-spin" />
        <span className="ml-2 text-gray-500">{t('common.loading')}</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="text-red-800">Error: {error}</div>
        <button
          onClick={loadCatalog}
          className="mt-2 text-sm text-red-600 hover:text-red-800 underline"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('adapters.title')}</h1>
        <button
          onClick={() => navigate('/instances')}
          className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4 mr-2" />
          {t('instances.create')}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {adapters.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500">
            {t('adapters.noAdapters') || '暂无可用接入器，请在 adapters/ 目录添加 adapter.yaml'}
          </div>
        ) : (
          adapters.map((adapter) => (
            <div key={adapter.id} className="bg-white rounded-lg shadow p-6">
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center">
                  <span className="text-3xl mr-3">{adapter.icon || '🔌'}</span>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">{adapter.name}</h3>
                    <p className="text-xs text-gray-400 font-mono">{adapter.id}</p>
                  </div>
                </div>
                <span className="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">
                  {adapter.platform_code}
                </span>
              </div>

              <p className="text-sm text-gray-600 mb-4">{adapter.description}</p>

              <div className="space-y-2 text-sm text-gray-600 mb-4">
                <div className="flex justify-between">
                  <span>{t('adapters.version')}:</span>
                  <span>v{adapter.version}</span>
                </div>
                <div className="flex justify-between">
                  <span>{t('adapters.runtime')}:</span>
                  <span>{adapter.runtime_type}</span>
                </div>
                {adapter.capabilities && adapter.capabilities.length > 0 && (
                  <div className="flex flex-wrap gap-1 pt-1">
                    {adapter.capabilities.map((cap) => (
                      <span
                        key={cap}
                        className="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 rounded"
                      >
                        {cap}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex items-center justify-between pt-4 border-t border-gray-200">
                <span className="text-xs text-gray-400">
                  {Object.keys(adapter.config_schema || {}).length} 个配置项
                </span>
                <button
                  onClick={() => navigate('/instances')}
                  className="text-sm text-blue-600 hover:text-blue-800"
                >
                  {t('instances.create')} →
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
