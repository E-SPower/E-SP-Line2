import { useTranslation } from 'react-i18next'
import { Plus, Play, Square, Edit, Trash2, FileText, X, RefreshCw, Loader2, Wifi, WifiOff, AlertCircle, Clock, Plug, ShoppingBag, Fish } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
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

// Per-hour log level counts (heatmap entry)
interface HeatmapEntry {
  hour: string
  debug: number
  info: number
  warning: number
  error: number
  unmarked: number
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
  const [logFrom, setLogFrom] = useState<string>('')
  const [logTo, setLogTo] = useState<string>('')
  // Heatmap: per-hour counts
  const [heatmap, setHeatmap] = useState<HeatmapEntry[]>([])

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

  // instanceOverride lets callers fetch logs before the `logInstance` state
  // has been committed (e.g. right after openLogs calls setLogInstance), so
  // the first click always loads logs instead of returning early.
  const fetchLogs = async (level: string, keyword: string, from: string, to: string, instanceOverride?: Instance) => {
    const instance = instanceOverride || logInstance
    if (!instance) return
    setLogsLoading(true)
    // Read a bounded tail (default 2000 lines) so a huge log never stalls the request.
    const result = await apiClient.getInstanceLogs(instance.id, {
      level: level || undefined,
      keyword: keyword || undefined,
      from: from || undefined,
      to: to || undefined,
    })
    setLogsLoading(false)
    setLogs(result.error ? `无法读取日志: ${result.error}` : (result.data || ''))

    // Load heatmap (respect current level filter)
    const hm = await apiClient.getInstanceLogHeatmap(instance.id, level || undefined)
    if (!hm.error) setHeatmap(hm.data || [])
  }

  // 关键词输入防抖：停止输入 300ms 后才向后端请求过滤
  const keywordTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const handleKeywordChange = (value: string) => {
    setLogKeyword(value)
    if (keywordTimer.current) clearTimeout(keywordTimer.current)
    keywordTimer.current = setTimeout(() => {
      fetchLogs(logLevel, value, logFrom, logTo)
    }, 300)
  }

  const openLogs = async (instance: Instance) => {
    setLogInstance(instance)
    setLogLevel('')
    setLogKeyword('')
    setLogFrom('')
    setLogTo('')
    setLogs('')
    setHeatmap([])
    // Pass the instance explicitly: setLogInstance is async, so relying on
    // the `logInstance` state here would skip loading on the first click.
    await fetchLogs('', '', '', '', instance)
  }

  const refreshLogs = async () => {
    await fetchLogs(logLevel, logKeyword, logFrom, logTo)
  }

  const handleClearLogs = async () => {
    if (!logInstance) return
    if (!window.confirm('确定清空该实例的全部日志？')) return
    const result = await apiClient.clearInstanceLogs(logInstance.id)
    if (result.error) {
      alert(`清空日志失败: ${result.error}`)
      return
    }
    setLogs('')
    setHeatmap([])
    await fetchLogs(logLevel, logKeyword, logFrom, logTo)
  }

  const changeLevel = async (level: string) => {
    setLogLevel(level)
    await fetchLogs(level, logKeyword, logFrom, logTo)
  }

  // 组件卸载时清理防抖定时器
  useEffect(() => {
    return () => {
      if (keywordTimer.current) clearTimeout(keywordTimer.current)
    }
  }, [])

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
        <Loader2 className="w-6 h-6 text-blue-500 animate-spin" />
        <span className="ml-2 text-gray-500">{t('common.loading')}</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg p-4">
        <div className="text-red-800 dark:text-red-300">Error: {error}</div>
        <button
          onClick={loadInstances}
          className="mt-2 text-sm text-red-600 hover:text-red-800 underline"
        >
          Retry
        </button>
      </div>
    )
  }

  // Status badge + icon helper for cards
  const statusBadge = (instance: Instance) => {
    if (instance.status === 'initializing') {
      return (
        <span className="px-2 py-1 inline-flex items-center text-xs font-semibold rounded-full bg-yellow-100 dark:bg-yellow-900/40 text-yellow-800 dark:text-yellow-300">
          <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
          {t('instances.initializing') || '初始化中'}
        </span>
      )
    }
    if (instance.status === 'error') {
      return (
        <span className="px-2 py-1 inline-flex items-center text-xs font-semibold rounded-full bg-red-100 dark:bg-red-900/40 text-red-800 dark:text-red-300">
          <AlertCircle className="w-3 h-3 mr-1" />
          {t('instances.statusError') || '异常'}
        </span>
      )
    }
    if (instance.status === 'running') {
      return (
        <span className="px-2 py-1 inline-flex items-center text-xs font-semibold rounded-full bg-green-100 dark:bg-green-900/40 text-green-800 dark:text-green-300">
          <Wifi className="w-3 h-3 mr-1" />
          {t('adapters.running')}
        </span>
      )
    }
    return (
      <span className="px-2 py-1 inline-flex items-center text-xs font-semibold rounded-full bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300">
        <WifiOff className="w-3 h-3 mr-1" />
        {t('adapters.stopped')}
      </span>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('instances.title')}</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('instances.subtitle') || '管理各平台的接入实例'}</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={loadInstances}
            className="flex items-center px-3 py-2 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-200 rounded-lg shadow hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            {t('common.refresh')}
          </button>
          <button
            onClick={openCreate}
            className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <Plus className="w-4 h-4 mr-2" />
            {t('instances.create')}
          </button>
        </div>
      </div>

      {/* Instance cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {instances.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500">
            {t('instances.noInstances') || 'No instances found. Create your first instance to get started.'}
          </div>
        ) : (
          instances.map((instance) => {
            const instAdapter = adapters.find((a) => a.id === instance.adapter_id)
            const cfg = parseConfig(instance.config)
            const cfgKeys = Object.keys(cfg)
            const isRunning = instance.status === 'running'
            const isInitializing = instance.status === 'initializing'
            return (
              <div
                key={instance.id}
                className={`bg-white dark:bg-gray-800 rounded-lg shadow p-6 ${
                  !isRunning && !isInitializing ? 'opacity-90' : ''
                }`}
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center">
                    <span className="mr-3 p-2 rounded-lg bg-blue-50 dark:bg-blue-900/30">
                      {instAdapter?.platform_code === 'taobao' ? (
                        <ShoppingBag className="w-6 h-6 text-blue-500" />
                      ) : instAdapter?.platform_code === 'xianyu' ? (
                        <Fish className="w-6 h-6 text-cyan-500" />
                      ) : (
                        <Plug className="w-6 h-6 text-gray-400" />
                      )}
                    </span>
                    <div>
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{instance.name}</h3>
                      <p className="text-xs text-gray-400 font-mono">{instAdapter?.name || instance.adapter_id}</p>
                    </div>
                  </div>
                  {statusBadge(instance)}
                </div>

                {/* Init progress (initializing) */}
                {isInitializing && initStatuses[instance.id] && (
                  <div className="mb-4">
                    <div className="flex items-center space-x-2">
                      <div className="flex-1 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-yellow-500 rounded-full transition-all"
                          style={{ width: `${Math.min(initStatuses[instance.id].progress ?? 5, 100)}%` }}
                        />
                      </div>
                      <span className="text-xs text-gray-500">
                        {initStatuses[instance.id].progress ?? 0}%
                      </span>
                    </div>
                    {initStatuses[instance.id].message && (
                      <div className="mt-1 text-xs text-gray-400 truncate">
                        {initStatuses[instance.id].message}
                      </div>
                    )}
                  </div>
                )}

                <div className="space-y-2 text-sm text-gray-600 dark:text-gray-300 mb-4">
                  <div className="flex justify-between">
                    <span>{t('instances.adapter')}:</span>
                    <span className="font-mono text-xs break-all max-w-[60%] text-right">
                      {instAdapter?.name || instance.adapter_id}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('instances.platform') || '平台'}:</span>
                    <span>{instAdapter?.platform_code || instance.platform_id || '-'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('instances.config')}:</span>
                    <span className="flex flex-wrap gap-1 justify-end">
                      {cfgKeys.length === 0 ? (
                        <span className="text-gray-400 dark:text-gray-500">未配置</span>
                      ) : (
                        cfgKeys.map((k) => (
                          <span
                            key={k}
                            className="px-1.5 py-0.5 text-xs bg-green-50 dark:bg-green-900/40 text-green-700 dark:text-green-300 rounded"
                            title={k === 'cookie' ? '已设置' : undefined}
                          >
                            {k === 'cookie' ? 'Cookie ✓' : `${k} ✓`}
                          </span>
                        ))
                      )}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.createdAt') || '创建时间'}:</span>
                    <span>{new Date(instance.created_at).toLocaleString()}</span>
                  </div>
                </div>

                <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
                  <div className="flex items-center space-x-1">
                    {isRunning ? (
                      <button
                        onClick={() => handleStop(instance.id)}
                        title={t('adapters.stop')}
                        className="p-1.5 text-red-600 hover:text-red-800 dark:text-red-400 rounded hover:bg-red-50 dark:hover:bg-red-900/30"
                      >
                        <Square className="w-4 h-4" />
                      </button>
                    ) : isInitializing ? (
                      <button
                        disabled
                        title={t('instances.initializing') || '初始化中，请稍候'}
                        className="p-1.5 text-gray-300 rounded cursor-not-allowed"
                      >
                        <Play className="w-4 h-4" />
                      </button>
                    ) : (
                      <button
                        onClick={() => handleStart(instance.id)}
                        title={t('adapters.start')}
                        className="p-1.5 text-green-600 hover:text-green-800 dark:text-green-400 rounded hover:bg-green-50 dark:hover:bg-green-900/30"
                      >
                        <Play className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      onClick={() => openLogs(instance)}
                      title={t('instances.logs') || '查看日志'}
                      className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                      <FileText className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => openEdit(instance)}
                      title={t('common.edit')}
                      className="p-1.5 text-blue-600 hover:text-blue-800 dark:text-blue-400 rounded hover:bg-blue-50 dark:hover:bg-blue-900/30"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(instance.id)}
                      title={t('common.delete')}
                      className="p-1.5 text-red-600 hover:text-red-800 dark:text-red-400 rounded hover:bg-red-50 dark:hover:bg-red-900/30"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                  <span className="flex items-center text-xs text-gray-400">
                    <Clock className="w-3 h-3 mr-1" />
                    {isRunning ? t('adapters.running') : t('adapters.stopped')}
                  </span>
                </div>
              </div>
            )
          })
        )}
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
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-4xl max-h-[85vh] flex flex-col">
              <div className="px-5 py-3 border-b border-gray-200 dark:border-gray-700">
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <FileText className="w-5 h-5 text-gray-500 dark:text-gray-400 mr-2" />
                    <span className="font-semibold text-gray-800 dark:text-gray-100">
                      {logInstance.name} — 日志
                    </span>
                    <span className="ml-3 text-xs text-gray-400 dark:text-gray-500">
                      {logInstance.adapter_id}
                    </span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={refreshLogs}
                      className="inline-flex items-center px-2.5 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                      title="刷新"
                    >
                      <RefreshCw className={`w-4 h-4 mr-1 ${logsLoading ? 'animate-spin' : ''}`} />
                      刷新
                    </button>
                    <button
                      onClick={handleClearLogs}
                      className="inline-flex items-center px-2.5 py-1.5 text-sm text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/30 rounded"
                      title="清空日志"
                    >
                      <Trash2 className="w-4 h-4 mr-1" />
                      清空
                    </button>
                    <button
                      onClick={closeLogs}
                      className="p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                      title="关闭"
                    >
                      <X className="w-5 h-5" />
                    </button>
                  </div>
                </div>
                {/* 级别过滤 + 关键词搜索 */}
                <div className="mt-3 flex items-center space-x-2">
                  <div className="flex items-center space-x-1 bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
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
                    onChange={(e) => handleKeywordChange(e.target.value)}
                    placeholder="关键词过滤（多词以空格分隔，需同时包含）..."
                    className="flex-1 px-3 py-1.5 text-sm bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                {/* 时间范围筛选 */}
                <div className="mt-2 flex items-center space-x-2">
                  <span className="text-xs text-gray-400">时间范围</span>
                  <input
                    type="datetime-local"
                    value={logFrom}
                    onChange={(e) => { setLogFrom(e.target.value); fetchLogs(logLevel, logKeyword, e.target.value, logTo) }}
                    className="px-2 py-1 text-xs bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                  <span className="text-xs text-gray-400">至</span>
                  <input
                    type="datetime-local"
                    value={logTo}
                    onChange={(e) => { setLogTo(e.target.value); fetchLogs(logLevel, logKeyword, logFrom, e.target.value) }}
                    className="px-2 py-1 text-xs bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                  <button
                    onClick={() => { setLogFrom(''); setLogTo(''); fetchLogs(logLevel, logKeyword, '', '') }}
                    className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 border border-gray-300 rounded-lg"
                  >
                    清除
                  </button>
                </div>
                {/* 热力图 */}
                {heatmap.length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs text-gray-400 mb-1">日志热力图（按小时）</p>
                    <div className="flex flex-wrap gap-1">
                      {heatmap.map((h) => {
                        const total = (h.info || 0) + (h.error || 0) + (h.warning || 0) + (h.debug || 0)
                        const errPct = total > 0 ? ((h.error || 0) / total) * 100 : 0
                        return (
                          <div
                            key={h.hour}
                            className="px-2 py-1 text-xs rounded cursor-default"
                            style={{
                              background: errPct >= 50 ? '#ef4444' : errPct >= 20 ? '#f97316' : total > 0 ? '#3b82f6' : '#374151',
                              color: '#fff',
                            }}
                            title={`${new Date(h.hour).toLocaleString()} | E:${h.error} W:${h.warning} I:${h.info} D:${h.debug}`}
                          >
                            {new Date(h.hour).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                            <span className="ml-1 opacity-75">{total}</span>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )}
              </div>
              <div className="flex-1 overflow-y-auto p-4 bg-gray-900 rounded-b-lg">
                {logsLoading ? (
                  <p className="text-gray-400 text-sm">加载中...</p>
                ) : (
                  <pre className="text-xs text-green-400 font-mono whitespace-pre-wrap break-words">
                    {logs || (logKeyword ? '(无匹配)' : '(暂无日志)')}
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
