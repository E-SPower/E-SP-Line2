import { useTranslation } from 'react-i18next'
import {
  Trash2,
  RefreshCw,
  Loader2,
  Wifi,
  WifiOff,
  X,
  Pencil,
  Server,
  Plug,
  Eye,
  EyeOff,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'
import EntityModal, { FieldConfig } from '../components/EntityModal'

// Access key input with a show/hide toggle and a regenerate button.
function KeyField({
  value,
  onChange,
  placeholder,
  onRegenerate,
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  onRegenerate: () => void
}) {
  const { t } = useTranslation()
  const [show, setShow] = useState(false)
  return (
    <div className="flex items-center space-x-2">
      <input
        type={show ? 'text' : 'password'}
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        required
        className="w-full px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 placeholder-gray-400 dark:placeholder-gray-500 font-mono"
      />
      <button
        type="button"
        onClick={() => setShow((s) => !s)}
        title={show ? t('adapterGateway.hideKey') : t('adapterGateway.showKey')}
        className="shrink-0 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
      >
        {show ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
      </button>
      <button
        type="button"
        onClick={onRegenerate}
        title={t('adapterGateway.regenerateKey')}
        className="shrink-0 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
      >
        <RefreshCw className="w-4 h-4" />
      </button>
    </div>
  )
}

interface GatewayAdapter {
  id: string
  name: string
  mode: string // server / client
  listen_path: string
  ws_url: string
  key: string
  platform: string
  scope: string
  status: string
  enabled: boolean
  last_connected_at: string | null
  created_at: string
  created_by: string
}

interface AdapterConnection {
  id: string
  adapter_id: string
  adapter_name: string
  mode: string
  platform: string
  remote_addr: string
  status: string
  connected_at: string
  disconnected_at: string | null
  last_heartbeat: string
  message_count: number
}

export default function AdapterGateway() {
  const { t } = useTranslation()
  const [adapters, setAdapters] = useState<GatewayAdapter[]>([])
  const [connections, setConnections] = useState<AdapterConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Create/edit modal
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<GatewayAdapter | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [formMode, setFormMode] = useState<string>('server')

  // Connection detail drawer
  const [detailAdapter, setDetailAdapter] = useState<GatewayAdapter | null>(null)
  const [detailConns, setDetailConns] = useState<AdapterConnection[]>([])

  const loadData = async () => {
    setLoading(true)
    setError(null)
    const [adapterRes, connRes] = await Promise.all([
      apiClient.getGatewayAdapters(100, 0),
      apiClient.getAdapterConnections(100, 0),
    ])
    if (adapterRes.error) {
      setError(adapterRes.error)
      setAdapters([])
    } else {
      setAdapters(adapterRes.data || [])
    }
    if (connRes.error) {
      setConnections([])
    } else {
      setConnections(connRes.data || [])
    }
    setLoading(false)
  }

  useEffect(() => {
    loadData()
  }, [])

  const openCreate = (mode: string) => {
    setEditing(null)
    setFormError(null)
    setFormMode(mode)
    setModalOpen(true)
  }

  const openEdit = (adapter: GatewayAdapter) => {
    setEditing(adapter)
    setFormError(null)
    setFormMode(adapter.mode)
    setModalOpen(true)
  }

  const handleValuesChange = (values: Record<string, any>) => {
    if (values.mode && values.mode !== formMode) {
      setFormMode(values.mode)
    }
  }

  // Generate a random 16-character access key (digits + upper/lowercase).
  const generateKey = () => {
    const charset = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'
    let key = ''
    const arr = new Uint32Array(16)
    crypto.getRandomValues(arr)
    for (let i = 0; i < 16; i++) {
      key += charset[arr[i] % charset.length]
    }
    return key
  }

  const handleSubmit = async (values: Record<string, any>) => {
    setSubmitting(true)
    setFormError(null)
    const payload = {
      name: values.name,
      mode: values.mode || 'server',
      listen_path: values.listen_path || '',
      ws_url: values.ws_url || '',
      key: values.key || '',
      platform: values.platform || '',
      scope: values.scope || 'read+write',
      enabled: values.enabled !== undefined ? values.enabled : true,
    }
    const result = editing
      ? await apiClient.updateGatewayAdapter(editing.id, payload)
      : await apiClient.createGatewayAdapter(payload)
    setSubmitting(false)
    if (result.error) {
      setFormError(result.error)
      return
    }
    setModalOpen(false)
    loadData()
  }

  const handleDelete = async (adapter: GatewayAdapter) => {
    if (!window.confirm(t('common.confirmDelete') || '确定要删除吗？')) return
    const result = await apiClient.deleteGatewayAdapter(adapter.id)
    if (result.error) {
      window.alert(result.error)
      return
    }
    loadData()
  }

  const openDetail = async (adapter: GatewayAdapter) => {
    setDetailAdapter(adapter)
    setDetailConns([])
    const result = await apiClient.getAdapterConnectionsByAdapter(adapter.id, 50, 0)
    if (!result.error) {
      setDetailConns(result.data || [])
    }
  }

  // Access key field with show/hide toggle and a regenerate button. A random
  // key is pre-filled for new adapters (no insecure default).
  const keyField: FieldConfig = {
    key: 'key',
    label: t('adapterGateway.key'),
    type: 'password',
    required: true,
    placeholder: t('adapterGateway.keyPlaceholder'),
    helpText: t('adapterGateway.keyHelp'),
    defaultValue: generateKey(),
    render: (value: any, setValue: (v: any) => void) => (
      <KeyField
        value={value ?? ''}
        onChange={setValue}
        placeholder={t('adapterGateway.keyPlaceholder')}
        onRegenerate={() => setValue(generateKey())}
      />
    ),
  }

  // Fields are mode-dependent: server mode shows the listen path, client mode
  // shows the target WebSocket URL. The access key is always shown.
  const fields: FieldConfig[] = [
    { key: 'name', label: t('adapterGateway.name'), required: true, placeholder: t('adapterGateway.namePlaceholder') },
    {
      key: 'mode',
      label: t('adapterGateway.mode'),
      type: 'select',
      options: [
        { value: 'server', label: t('adapterGateway.modeServer') },
        { value: 'client', label: t('adapterGateway.modeClient') },
      ],
      defaultValue: 'server',
      helpText: t('adapterGateway.modeHelp'),
    },
    ...(formMode === 'server'
      ? ([
          {
            key: 'listen_path',
            label: t('adapterGateway.listenPath'),
            type: 'text',
            placeholder: '/ws',
            helpText: t('adapterGateway.listenPathHelp'),
          },
        ] as FieldConfig[])
      : ([
          {
            key: 'ws_url',
            label: t('adapterGateway.wsUrl'),
            type: 'text',
            placeholder: 'ws://host:port/path',
            helpText: t('adapterGateway.wsUrlHelp'),
          },
        ] as FieldConfig[])),
    keyField,
    {
      key: 'platform',
      label: t('adapterGateway.platform'),
      type: 'select',
      options: [
        { value: '', label: t('adapterGateway.allPlatforms') },
        { value: 'taobao', label: '淘宝 (taobao)' },
        { value: 'xianyu', label: '闲鱼 (xianyu)' },
      ],
      defaultValue: '',
      helpText: t('adapterGateway.platformHelp'),
    },
    {
      key: 'scope',
      label: t('adapterGateway.scope'),
      type: 'select',
      options: [
        { value: 'read+write', label: 'read+write' },
        { value: 'read', label: 'read' },
        { value: 'write', label: 'write' },
      ],
      defaultValue: 'read+write',
      helpText: t('adapterGateway.scopeHelp'),
    },
    {
      key: 'enabled',
      label: t('adapterGateway.enabled'),
      type: 'switch',
      defaultValue: true,
    },
  ]

  const connectedCount = connections.filter((c) => c.status === 'connected').length

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
          onClick={loadData}
          className="mt-2 text-sm text-red-600 hover:text-red-800 underline"
        >
          {t('common.refresh')}
        </button>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('adapterGateway.title')}</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('adapterGateway.subtitle')}</p>
        </div>
        <div className="flex items-center space-x-3">
          <div className="flex items-center px-3 py-2 bg-white dark:bg-gray-800 rounded-lg shadow text-sm">
            <Wifi className="w-4 h-4 mr-2 text-green-500" />
            <span className="text-gray-700 dark:text-gray-200">
              {t('adapterGateway.connected')}: {connectedCount}
            </span>
          </div>
          <button
            onClick={loadData}
            className="flex items-center px-3 py-2 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-200 rounded-lg shadow hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            {t('common.refresh')}
          </button>
          <button
            onClick={() => openCreate('server')}
            className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <Server className="w-4 h-4 mr-2" />
            {t('adapterGateway.createServer')}
          </button>
          <button
            onClick={() => openCreate('client')}
            className="flex items-center px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700"
          >
            <Plug className="w-4 h-4 mr-2" />
            {t('adapterGateway.createClient')}
          </button>
        </div>
      </div>

      {/* Adapter cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {adapters.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500">
            {t('adapterGateway.noAdapters')}
          </div>
        ) : (
          adapters.map((adapter) => {
            const isActive = adapter.enabled && adapter.status === 'active'
            const adapterConns = connections.filter((c) => c.adapter_id === adapter.id)
            const online = adapterConns.some((c) => c.status === 'connected')
            return (
              <div
                key={adapter.id}
                className={`bg-white dark:bg-gray-800 rounded-lg shadow p-6 ${
                  !isActive ? 'opacity-60' : ''
                }`}
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center">
                    <span className="mr-3 p-2 rounded-lg bg-blue-50 dark:bg-blue-900/30">
                      {adapter.mode === 'server' ? (
                        <Server className="w-6 h-6 text-blue-500" />
                      ) : (
                        <Plug className="w-6 h-6 text-purple-500" />
                      )}
                    </span>
                    <div>
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{adapter.name}</h3>
                      <p className="text-xs text-gray-400 font-mono">{adapter.id.slice(0, 8)}...</p>
                    </div>
                  </div>
                  <span
                    className={`px-2 py-1 text-xs font-semibold rounded-full ${
                      isActive
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                        : 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'
                    }`}
                  >
                    {isActive ? t('adapterGateway.active') : t('adapterGateway.disabled')}
                  </span>
                </div>

                <div className="space-y-2 text-sm text-gray-600 dark:text-gray-300 mb-4">
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.mode')}:</span>
                    <span className="flex items-center">
                      {adapter.mode === 'server' ? (
                        <>
                          <Server className="w-3.5 h-3.5 mr-1 text-blue-500" />
                          {t('adapterGateway.modeServer')}
                        </>
                      ) : (
                        <>
                          <Plug className="w-3.5 h-3.5 mr-1 text-purple-500" />
                          {t('adapterGateway.modeClient')}
                        </>
                      )}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.endpoint')}:</span>
                    <span className="font-mono text-xs break-all max-w-[60%] text-right">
                      {adapter.mode === 'server'
                        ? adapter.listen_path || '/ws'
                        : adapter.ws_url || '-'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.platform')}:</span>
                    <span>{adapter.platform || t('adapterGateway.allPlatforms')}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.scope')}:</span>
                    <span className="font-mono">{adapter.scope}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.status')}:</span>
                    <span className="flex items-center">
                      {online ? (
                        <>
                          <Wifi className="w-3.5 h-3.5 mr-1 text-green-500" />
                          {t('adapterGateway.online')}
                        </>
                      ) : (
                        <>
                          <WifiOff className="w-3.5 h-3.5 mr-1 text-gray-400" />
                          {t('adapterGateway.offline')}
                        </>
                      )}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('adapterGateway.createdAt')}:</span>
                    <span>{new Date(adapter.created_at).toLocaleString()}</span>
                  </div>
                </div>

                <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
                  <button
                    onClick={() => openDetail(adapter)}
                    className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400"
                  >
                    {t('adapterGateway.viewConnections')} ({adapterConns.length})
                  </button>
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => openEdit(adapter)}
                      title={t('common.edit')}
                      className="p-1.5 text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                      <Pencil className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(adapter)}
                      title={t('common.delete')}
                      className="p-1.5 text-red-600 hover:text-red-800 dark:text-red-400 rounded hover:bg-red-50 dark:hover:bg-red-900/30"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Create/edit modal */}
      <EntityModal
        title={editing ? t('adapterGateway.edit') : t('adapterGateway.create')}
        open={modalOpen}
        fields={fields}
        initialValues={editing ? { ...editing } : undefined}
        submitting={submitting}
        error={formError}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
        onValuesChange={handleValuesChange}
      />

      {/* Connection detail drawer */}
      {detailAdapter && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen px-4">
            <div className="fixed inset-0 bg-black bg-opacity-50" onClick={() => setDetailAdapter(null)} />
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {t('adapterGateway.connectionsFor')}: {detailAdapter.name}
                </h2>
                <button
                  onClick={() => setDetailAdapter(null)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {detailConns.length === 0 ? (
                <div className="text-center py-8 text-gray-500">{t('adapterGateway.noConnections')}</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                      <tr>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                          {t('common.status')}
                        </th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                          {t('adapterGateway.remoteAddr')}
                        </th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                          {t('adapterGateway.connectedAt')}
                        </th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                          {t('adapterGateway.messages')}
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                      {detailConns.map((conn) => (
                        <tr key={conn.id}>
                          <td className="px-4 py-3 whitespace-nowrap">
                            <span
                              className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                conn.status === 'connected'
                                  ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                                  : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
                              }`}
                            >
                              {conn.status === 'connected'
                                ? t('adapterGateway.online')
                                : t('adapterGateway.offline')}
                            </span>
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700 dark:text-gray-300 font-mono">
                            {conn.remote_addr}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                            {new Date(conn.connected_at).toLocaleString()}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                            {conn.message_count}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
