import { useTranslation } from 'react-i18next'
import { Plus, Play, Square, Edit, Trash2, FileText, X, RefreshCw } from 'lucide-react'
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

// init status returned by GET /instances/:id/init
interface InitStatus {
  status: string // installing, done, failed
  progress: number
  message?: string
  error?: string
  instance_status?: string
}

interface AdapterOption {
  id: string
  name: string
  platform_code: string
  config_schema: Record<string, ConfigFieldDef>
}

// config_schema field definition from adapter.yaml
interface ConfigFieldDef {
  label: string
  type: 'text' | 'password' | 'number'
  required?: boolean
  placeholder?: string
  help?: string
  default?: any
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
  // Dependency installation (init) status per instance
  const [initStatuses, setInitStatuses] = useState<Record<string, InitStatus>>({})
  // Log drawer state
  const [logInstance, setLogInstance] = useState<Instance | null>(null)
  const [logs, setLogs] = useState('')
  const [logsLoading, setLogsLoading] = useState(false)
  const [logLevel, setLogLevel] = useState<string>('')
  const [logKeyword, setLogKeyword] = useState<string>('')

  useEffect(() => {
    loadInstances()
    loadAdapters()
  }, [])

  // Poll dependency installation status for instances that are initializing.
  useEffect(() => {
    const initializing = instances.filter((i) => i.status === 'initializing')
    if (initializing.length === 0) return

    const poll = async () => {
      const results = await Promise.all(
        initializing.map(async (inst) => {
          const r = await apiClient.getInstanceInitStatus(inst.id)
          return { id: inst.id, status: r.data }
        })
      )
      const next: Record<string, InitStatus> = {}
      let stillInitializing = false
      results.forEach(({ id, status }) => {
        next[id] = status
        const st = status?.status || ''
        if (st === 'installing' || st === 'initializing') {
          stillInitializing = true
        }
      })
      setInitStatuses((prev) => ({ ...prev, ...next }))
      // If all instances finished, refresh the instance list once.
      if (!stillInitializing) {
        loadInstances()
      }
    }

    poll()
    const timer = setInterval(poll, 3000)
    return () => clearInterval(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instances])

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
    const result = await apiClient.getAdapterCatalog()
    if (!result.error) {
      setAdapters((result.data || []).map((a: any) => ({
        id: a.id,
        name: a.name,
        platform_code: a.platform_code,
        config_schema: a.config_schema || {},
      })))
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

    // Determine the adapter's config schema from the selected catalog adapter.
    const adapterId = values.adapter_id || formAdapterId
    const adapter = adapters.find((a) => a.id === adapterId)
    const configKeys = new Set(Object.keys(adapter?.config_schema || {}))
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

    // Resolve platform_id from the selected adapter's platform code.
    if (adapter && !payload.platform_id) {
      payload.platform_id = adapter.platform_code
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

  const openLogs = async (instance: Instance) => {
    setLogInstance(instance)
    setLogLevel('')
    setLogKeyword('')
    setLogs('')
    setLogsLoading(true)
    const result = await apiClient.getInstanceLogs(instance.id, { lines: 'all' })
    setLogsLoading(false)
    setLogs(result.error ? `无法读取日志: ${result.error}` : (result.data || ''))
  }

  const refreshLogs = async () => {
    if (!logInstance) return
    setLogsLoading(true)
    const result = await apiClient.getInstanceLogs(logInstance.id, {
      lines: 'all',
      level: logLevel || undefined,
    })
    setLogsLoading(false)
    setLogs(result.error ? `无法读取日志: ${result.error}` : (result.data || ''))
  }

  const changeLevel = async (level: string) => {
    setLogLevel(level)
    if (!logInstance) return
    setLogsLoading(true)
    const result = await apiClient.getInstanceLogs(logInstance.id, {
      lines: 'all',
      level: level || undefined,
    })
    setLogsLoading(false)
    setLogs(result.error ? `无法读取日志: ${result.error}` : (result.data || ''))
  }

  const closeLogs = () => {
    setLogInstance(null)
    setLogs('')
  }

  const adapterOptions = adapters.map((a) => ({ value: a.id, label: a.name }))

  // Build config fields from the selected adapter's config_schema (from its
  // adapter.yaml). No fields are shown until an adapter is chosen.
  const selectedAdapter = adapters.find((a) => a.id === formAdapterId)
  const configFields: FieldConfig[] = Object.entries(
    selectedAdapter?.config_schema || {}
  ).map(([key, f]) => ({
    key,
    label: f.label || key,
    type: f.type === 'number' ? 'number' : f.type === 'password' ? 'password' : 'text',
    required: f.required,
    placeholder: f.placeholder,
    helpText: f.help,
    defaultValue: f.default,
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
                {t('instances.config')}
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
                <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                  {t('instances.noInstances') || 'No instances found. Create your first instance to get started.'}
                </td>
              </tr>
            ) : (
              instances.map((instance) => {
                const instAdapter = adapters.find((a) => a.id === instance.adapter_id)
                const cfg = parseConfig(instance.config)
                const cfgKeys = Object.keys(cfg)
                return (
                <tr key={instance.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {instance.name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {instAdapter?.name || instance.adapter_id}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {cfgKeys.length === 0 ? (
                      <span className="text-gray-400">未配置</span>
                    ) : (
                      <div className="flex flex-wrap gap-1">
                        {cfgKeys.map((k) => (
                          <span
                            key={k}
                            className="px-1.5 py-0.5 text-xs bg-green-50 text-green-700 rounded"
                            title={k === 'cookie' ? '已设置' : undefined}
                          >
                            {k === 'cookie' ? 'Cookie ✓' : `${k} ✓`}
                          </span>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {instance.status === 'initializing' ? (
                      <div>
                        <span className="px-2 inline-flex items-center text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                          <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                          {t('instances.initializing') || '初始化中'}
                        </span>
                        {initStatuses[instance.id] && (
                          <div className="mt-1 flex items-center space-x-2">
                            <div className="w-20 h-1.5 bg-gray-200 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-yellow-500 rounded-full transition-all"
                                style={{ width: `${Math.min(initStatuses[instance.id].progress ?? 5, 100)}%` }}
                              />
                            </div>
                            <span className="text-xs text-gray-500">
                              {initStatuses[instance.id].progress ?? 0}%
                            </span>
                          </div>
                        )}
                        {initStatuses[instance.id]?.message && (
                          <div className="mt-1 text-xs text-gray-400 max-w-[180px] truncate">
                            {initStatuses[instance.id].message}
                          </div>
                        )}
                      </div>
                    ) : instance.status === 'error' ? (
                      <span className="px-2 inline-flex items-center text-xs leading-5 font-semibold rounded-full bg-red-100 text-red-800">
                        {t('instances.statusError') || '异常'}
                      </span>
                    ) : (
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        instance.status === 'running'
                          ? 'bg-green-100 text-green-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}>
                        {instance.status === 'running' ? t('adapters.running') : t('adapters.stopped')}
                      </span>
                    )}
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
                    ) : instance.status === 'initializing' ? (
                      <button
                        disabled
                        className="text-gray-300 rounded p-1 mr-3 cursor-not-allowed"
                        title={t('instances.initializing') || '初始化中，请稍候'}
                      >
                        <Play className="w-4 h-4" />
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
                      onClick={() => openLogs(instance)}
                      className="text-gray-500 hover:text-gray-700 mr-3"
                      title={t('instances.logs') || '查看日志'}
                    >
                      <FileText className="w-4 h-4" />
                    </button>
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
                )
              })
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

      {/* 实例日志窗口预览(居中模态) */}
      {logInstance && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen px-4 py-8">
            <div
              className="fixed inset-0 bg-black bg-opacity-50"
              onClick={closeLogs}
            />
            <div className="relative bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[85vh] flex flex-col">
              <div className="px-5 py-3 border-b border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <FileText className="w-5 h-5 text-gray-500 mr-2" />
                    <span className="font-semibold text-gray-800">
                      {logInstance.name} — 日志
                    </span>
                    <span className="ml-3 text-xs text-gray-400">
                      {logInstance.adapter_id}
                    </span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={refreshLogs}
                      className="inline-flex items-center px-2.5 py-1.5 text-sm text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded"
                      title="刷新"
                    >
                      <RefreshCw className={`w-4 h-4 mr-1 ${logsLoading ? 'animate-spin' : ''}`} />
                      刷新
                    </button>
                    <button
                      onClick={closeLogs}
                      className="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded"
                      title="关闭"
                    >
                      <X className="w-5 h-5" />
                    </button>
                  </div>
                </div>
                {/* 级别过滤 + 关键词搜索 */}
                <div className="mt-3 flex items-center space-x-2">
                  <div className="flex items-center space-x-1 bg-gray-100 rounded-lg p-1">
                    {[
                      { v: '', label: '全部' },
                      { v: 'debug', label: 'DEBUG' },
                      { v: 'info', label: 'INFO' },
                      { v: 'warning', label: 'WARNING' },
                      { v: 'error', label: 'ERROR' },
                    ].map((lv) => (
                      <button
                        key={lv.v || 'all'}
                        onClick={() => changeLevel(lv.v)}
                        className={`px-2.5 py-1 text-xs rounded ${
                          logLevel === lv.v
                            ? 'bg-blue-600 text-white'
                            : 'text-gray-600 hover:bg-gray-200'
                        }`}
                      >
                        {lv.label}
                      </button>
                    ))}
                  </div>
                  <input
                    value={logKeyword}
                    onChange={(e) => setLogKeyword(e.target.value)}
                    placeholder="关键词过滤..."
                    className="flex-1 px-3 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
              <div className="flex-1 overflow-y-auto p-4 bg-gray-900 rounded-b-lg">
                {logsLoading ? (
                  <p className="text-gray-400 text-sm">加载中...</p>
                ) : (
                  <pre className="text-xs text-green-400 font-mono whitespace-pre-wrap break-words">
                    {(() => {
                      if (!logKeyword) return logs || '(暂无日志)'
                      const kw = logKeyword.toLowerCase()
                      return (logs || '')
                        .split('\n')
                        .filter((l) => l.toLowerCase().includes(kw))
                        .join('\n') || '(无匹配)'
                    })()}
                  </pre>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
