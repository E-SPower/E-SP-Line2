import { useTranslation } from 'react-i18next'
import { Plus, Edit, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'
import EntityModal, { FieldConfig } from '../components/EntityModal'
import { useFormOptions } from '../hooks/useFormOptions'

interface Platform {
  id: string
  name: string
  code: string
  status: string
  description: string
  created_at: string
}

export default function Platforms() {
  const { t } = useTranslation()
  const [platforms, setPlatforms] = useState<Platform[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Platform | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const { getOptions } = useFormOptions()
  const codeOptions = getOptions('platform_codes')

  useEffect(() => {
    loadPlatforms()
  }, [])

  const loadPlatforms = async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getPlatforms()
    if (result.error) {
      setError(result.error)
      setPlatforms([])
    } else {
      setPlatforms(result.data || [])
    }
    setLoading(false)
  }

  const openCreate = () => {
    setEditing(null)
    setFormError(null)
    setModalOpen(true)
  }

  const openEdit = (platform: Platform) => {
    setEditing(platform)
    setFormError(null)
    setModalOpen(true)
  }

  const handleSubmit = async (values: Record<string, any>) => {
    setSubmitting(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updatePlatform(editing.id, values)
      : await apiClient.createPlatform(values)
    setSubmitting(false)
    if (result.error) {
      setFormError(result.error)
      return
    }
    setModalOpen(false)
    loadPlatforms()
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('common.confirmDelete') || 'Delete this item?')) return
    const result = await apiClient.deletePlatform(id)
    if (!result.error) {
      loadPlatforms()
    }
  }

  const fields: FieldConfig[] = [
    { key: 'name', label: t('common.name'), required: true },
    {
      key: 'code',
      label: t('platforms.code'),
      type: 'select',
      required: true,
      options: codeOptions,
      placeholder: t('platforms.selectCode'),
      helpText: t('platforms.codeHelp'),
    },
    { key: 'description', label: t('common.description') },
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
          onClick={loadPlatforms}
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
        <h1 className="text-2xl font-bold text-gray-900">{t('platforms.title')}</h1>
        <button
          onClick={openCreate}
          className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4 mr-2" />
          {t('platforms.create')}
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
                {t('platforms.code')}
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('common.description')}
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
            {platforms.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                  {t('platforms.noPlatforms') || 'No platforms found. Create your first platform to get started.'}
                </td>
              </tr>
            ) : (
              platforms.map((platform) => (
                <tr key={platform.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {platform.name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {platform.code}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {platform.description}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      platform.status === 'active' 
                        ? 'bg-green-100 text-green-800' 
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {platform.status === 'active' ? t('platforms.active') : t('platforms.inactive')}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => openEdit(platform)}
                      className="text-blue-600 hover:text-blue-900 mr-3"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(platform.id)}
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
        title={editing ? t('platforms.edit') || 'Edit Platform' : t('platforms.create')}
        open={modalOpen}
        fields={fields}
        initialValues={editing ? { name: editing.name, code: editing.code, description: editing.description } : undefined}
        submitting={submitting}
        error={formError}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
      />
    </div>
  )
}
