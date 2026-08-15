import { useTranslation } from 'react-i18next'
import { Plus, Play, Square, Edit, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'
import EntityModal, { FieldConfig } from '../components/EntityModal'

interface Instance {
  id: string
  name: string
  adapter_id: string
  platform_id: string
  status: string
  config: string
  created_at: string
}

interface AdapterOption {
  id: string
  name: string
  platform_id: string
}

/**
 * 内置适配器 ConfigSchema 映射(按平台 platform_id 匹配，与后端
 * internal/adapter/adapters.go 保持一致)。当所选接入器的平台匹配时，
 * 实例表单动态渲染这些配置字段。
 */
const ADAPTER_CONFIG_SCHEMAS: Record<string, { key: string; label: string; type: 'text' | 'password' | 'number'; placeholder?: string }[]> = {
  xianyu: [
    { key: 'cookie', label: '闲鱼 Cookie', type: 'password', placeholder: '登录闲鱼后从浏览器复制完整 Cookie' },
    { key: 'device_id', label: 'Device ID', type: 'text', placeholder: '可选，留空自动生成' },
    { key: 'heartbeat_interval', label: '心跳间隔(秒)', type: 'number', placeholder: '默认 15' },
    { key: 'reconnect_delay', label: '重连延迟(秒)', type: 'number', placeholder: '默认 5' },
  ],
  taobao: [
    { key: 'cookie', label: '淘宝 Cookie', type: 'password', placeholder: '登录淘宝后从浏览器复制完整 Cookie' },
    { key: 'device_id', label: 'Device ID', type: 'text', placeholder: '可选，留空自动生成' },
  ],
}

// Parse instance config JSON string -> object
function parseConfig(config: string | null | undefined): Record<string, any> {
  if (!config) return {}
  try {
    const parsed = JSON.parse(config)
    return typeof parsed === 'object' && parsed !== null ? parsed : {}
  } catch {
    return {}
  }
}

// Serialize config object -> JSON string
function serializeConfig(config: Record<string, any>): string {
  return JSON.stringify(config)
}

export default function Instances() {
  const { t } = useTranslation()
  const [instances, setInstances] = useState<Instance[]>([])
  const [adapters, setAdapters] = useState<AdapterOption[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Instance | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [formAdapterId, setFormAdapterId] = useState<string>('')

  useEffect(() => {
    loadInstances()
    loadAdapters()
  }, [])

  const loadInstances = async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getInstances()
    if (result.error) {
      setError(result.error)
      setInstances([])
    } else {
      setInstances(result.data || [])
    }
    setLoading(false)
  }

  const loadAdapters = async () => {
    const result = await apiClient.getAdapters()
    if (!result.error) {
      setAdapters((result.data || []).map((a: any) => ({ id: a.id, name: a.name, platform_id: a.platform_id })))
    }
  }

  const openCreate = () => {
    setEditing(null)
    setFormError(null)
    setFormAdapterId('')
    setModalOpen(true)
  }

  const openEdit = (instance: Instance) => {
    setEditing(instance)
    setFormError(null)
    setFormAdapterId(instance.adapter_id)
    setModalOpen(true)
  }

  const handleValuesChange = (values: Record<string, any>) => {
    if (values.adapter_id !== formAdapterId) {
      setFormAdapterId(values.adapter_id || '')
    }
  }

  const handleSubmit = async (values: Record<string, any>) => {
    setSubmitting(true)
    setFormError(null)

    // Determine the adapter's config schema from the submitted adapter's platform.
    const adapterId = values.adapter_id || formAdapterId
    const adapter = adapters.find((a) => a.id === adapterId)
    const configKeys = new Set(
      (ADAPTER_CONFIG_SCHEMAS[adapter?.platform_id || ''] || []).map((f) => f.key)
    )
    const config: Record<string, any> = {}
    const top: Record<string, any> = {}
    Object.entries(values).forEach(([key, val]) => {
      if (configKeys.has(key)) {
        config[key] = val
      } else {
        top[key] = val
      }
    })

    const payload: Record<string, any> = { ...top }

    // Resolve platform_id from the selected adapter.
    if (adapter && !payload.platform_id) {
      payload.platform_id = adapter.platform_id
    }

    if (Object.keys(config).length > 0) {
      payload.config = serializeConfig(config)
    }

    const result = editing
      ? await apiClient.updateInstance(editing.id, payload)
      : await apiClient.createInstance(payload)
    setSubmitting(false)
    if (result.error) {
      setFormError(result.error)
      return
    }
    setModalOpen(false)
    loadInstances()
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('common.confirmDelete') || 'Delete this item?')) return
    const result = await apiClient.deleteInstance(id)
    if (!result.error) {
      loadInstances()
    }
  }

  const handleStart = async (id: string) => {
    const result = await apiClient.startInstance(id)
    if (result.error) {
      window.alert(result.error)
      return
    }
    loadInstances()
  }

  const handleStop = async (id: string) => {
    const result = await apiClient.stopInstance(id)
    if (result.error) {
      window.alert(result.error)
      return
    }
    loadInstances()
  }

  const adapterOptions = adapters.map((a) => ({ value: a.id, label: a.name }))

  // Build config fields from all known adapter schemas (union of fields),
  // so the form is stable regardless of adapter selection. On submit, only
  // fields belonging to the selected adapter's schema are stored in config.
  const configFields: FieldConfig[] = Object.values(ADAPTER_CONFIG_SCHEMAS)
    .flat()
    .filter(
      (f, index, self) => self.findIndex((x) => x.key === f.key) === index
    )
    .map((f) => ({
      key: f.key,
      label: f.label,
      type: f.type === 'number' ? 'number' : 'text',
      placeholder: f.placeholder,
      helpText:
        f.key === 'cookie'
          ? t('instances.cookieHelp')
          : f.key === 'device_id'
          ? t('instances.deviceIdHelp')
          : undefined,
    }))

  const fields: FieldConfig[] = [
    { key: 'name', label: t('common.name'), required: true },
    {
      key: 'adapter_id',
      label: t('instances.adapter'),
      type: 'select',
      required: true,
      options: adapterOptions,
      placeholder: t('instances.selectAdapter'),
      helpText: t('instances.adapterHelp'),
    },
    ...configFields,
  ]

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">{t('common.loading')}</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="text-red-800">Error: {error}</div>
        <button
          onClick={loadInstances}
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
        <h1 className="text-2xl font-bold text-gray-900">{t('instances.title')}</h1>
        <button
          onClick={openCreate}
          className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4 mr-2" />
          {t('instances.create')}
        </button>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('common.name')}
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('instances.adapter')}
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('common.status')}
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('common.actions')}
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {instances.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-6 py-8 text-center text-gray-500">
                  {t('instances.noInstances') || 'No instances found. Create your first instance to get started.'}
                </td>
              </tr>
            ) : (
              instances.map((instance) => (
                <tr key={instance.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {instance.name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {instance.adapter_id}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      instance.status === 'running'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {instance.status === 'running' ? t('adapters.running') : t('adapters.stopped')}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    {instance.status === 'running' ? (
                      <button
                        onClick={() => handleStop(instance.id)}
                        className="text-red-600 hover:bg-red-50 rounded p-1 mr-3"
                        title={t('adapters.stop')}
                      >
                        <Square className="w-4 h-4" />
                      </button>
                    ) : (
                      <button
                        onClick={() => handleStart(instance.id)}
                        className="text-green-600 hover:bg-green-50 rounded p-1 mr-3"
                        title={t('adapters.start')}
                      >
                        <Play className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      onClick={() => openEdit(instance)}
                      className="text-blue-600 hover:text-blue-900 mr-3"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(instance.id)}
                      className="text-red-600 hover:text-red-900"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <EntityModal
        title={editing ? t('instances.edit') || 'Edit Instance' : t('instances.create')}
        open={modalOpen}
        fields={fields}
        initialValues={
          editing
            ? {
                name: editing.name,
                adapter_id: editing.adapter_id,
                ...parseConfig(editing.config),
              }
            : undefined
        }
        submitting={submitting}
        error={formError}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
        onValuesChange={handleValuesChange}
      />
    </div>
  )
}
