import { useTranslation } from 'react-i18next'
import { Plus, Play, Square, Edit, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'
import EntityModal, { FieldConfig } from '../components/EntityModal'
import { useFormOptions } from '../hooks/useFormOptions'

interface Adapter {
  id: string
  name: string
  platform_id: string
  version: string
  runtime_type: string
  status: string
  created_at: string
}

export default function Adapters() {
  const { t } = useTranslation()
  const [adapters, setAdapters] = useState<Adapter[]>([])
  const [platforms, setPlatforms] = useState<{ id: string; name: string }[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Adapter | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const { getOptions } = useFormOptions()
  const runtimeOptions = getOptions('runtime_types')

  useEffect(() => {
    loadAdapters()
    loadPlatforms()
  }, [])

  const loadAdapters = async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getAdapters()
    if (result.error) {
      setError(result.error)
      setAdapters([])
    } else {
      setAdapters(result.data || [])
    }
    setLoading(false)
  }

  const loadPlatforms = async () => {
    const result = await apiClient.getPlatforms()
    if (!result.error) {
      setPlatforms((result.data || []).map((p: any) => ({ id: p.id, name: p.name })))
    }
  }

  const handleStart = async (id: string) => {
    await apiClient.startAdapter(id)
    loadAdapters()
  }

  const handleStop = async (id: string) => {
    await apiClient.stopAdapter(id)
    loadAdapters()
  }

  const openCreate = () => {
    setEditing(null)
    setFormError(null)
    setModalOpen(true)
  }

  const openEdit = (adapter: Adapter) => {
    setEditing(adapter)
    setFormError(null)
    setModalOpen(true)
  }

  const handleSubmit = async (values: Record<string, any>) => {
    setSubmitting(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updateAdapter(editing.id, values)
      : await apiClient.createAdapter(values)
    setSubmitting(false)
    if (result.error) {
      setFormError(result.error)
      return
    }
    setModalOpen(false)
    loadAdapters()
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('common.confirmDelete') || 'Delete this item?')) return
    const result = await apiClient.deleteAdapter(id)
    if (!result.error) {
      loadAdapters()
    }
  }

  const platformOptions = platforms.map((p) => ({ value: p.id, label: p.name }))

  const fields: FieldConfig[] = [
    { key: 'name', label: t('common.name'), required: true },
    {
      key: 'platform_id',
      label: t('adapters.platform'),
      type: 'select',
      required: true,
      options: platformOptions,
      placeholder: t('adapters.selectPlatform'),
      helpText: t('adapters.platformHelp'),
    },
    { key: 'version', label: t('adapters.version'), required: true },
    {
      key: 'runtime_type',
      label: t('adapters.runtime'),
      type: 'select',
      options: runtimeOptions,
      placeholder: t('adapters.selectRuntime'),
      helpText: t('adapters.runtimeHelp'),
    },
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
          onClick={loadAdapters}
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
          onClick={openCreate}
          className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4 mr-2" />
          {t('adapters.create')}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {adapters.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500">
            {t('adapters.noAdapters') || 'No adapters found. Create your first adapter to get started.'}
          </div>
        ) : (
          adapters.map((adapter) => (
            <div key={adapter.id} className="bg-white rounded-lg shadow p-6">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">{adapter.name}</h3>
                  <p className="text-sm text-gray-500 mt-1">{t('adapters.platform')}: {adapter.platform_id}</p>
                </div>
                <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                  adapter.status === 'active' 
                    ? 'bg-green-100 text-green-800' 
                    : 'bg-gray-100 text-gray-800'
                }`}>
                  {adapter.status === 'active' ? t('adapters.running') : t('adapters.stopped')}
                </span>
              </div>
              
              <div className="space-y-2 text-sm text-gray-600 mb-4">
                <div className="flex justify-between">
                  <span>{t('adapters.version')}:</span>
                  <span>{adapter.version}</span>
                </div>
                <div className="flex justify-between">
                  <span>{t('adapters.runtime')}:</span>
                  <span>{adapter.runtime_type || 'N/A'}</span>
                </div>
              </div>

              <div className="flex items-center justify-between pt-4 border-t border-gray-200">
                <div className="flex space-x-2">
                  {adapter.status === 'active' ? (
                    <button 
                      onClick={() => handleStop(adapter.id)}
                      className="p-2 text-red-600 hover:bg-red-50 rounded"
                    >
                      <Square className="w-4 h-4" />
                    </button>
                  ) : (
                    <button 
                      onClick={() => handleStart(adapter.id)}
                      className="p-2 text-green-600 hover:bg-green-50 rounded"
                    >
                      <Play className="w-4 h-4" />
                    </button>
                  )}
                  <button
                    onClick={() => openEdit(adapter)}
                    className="p-2 text-blue-600 hover:bg-blue-50 rounded"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                </div>
                <button
                  onClick={() => handleDelete(adapter.id)}
                  className="p-2 text-red-600 hover:bg-red-50 rounded"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      <EntityModal
        title={editing ? t('adapters.edit') || 'Edit Adapter' : t('adapters.create')}
        open={modalOpen}
        fields={fields}
        initialValues={editing ? { name: editing.name, platform_id: editing.platform_id, version: editing.version, runtime_type: editing.runtime_type } : undefined}
        submitting={submitting}
        error={formError}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
      />
    </div>
  )
}
