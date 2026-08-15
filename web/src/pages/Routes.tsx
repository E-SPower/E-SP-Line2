import { useTranslation } from 'react-i18next'
import { Plus, Edit, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import apiClient from '../api/client'
import EntityModal, { FieldConfig } from '../components/EntityModal'

interface Route {
  id: string
  name: string
  platform_id: string
  priority: number
  target_type: string
  target_id: string
  enabled: boolean
  created_at: string
}

export default function RoutesPage() {
  const { t } = useTranslation()
  const [routes, setRoutes] = useState<Route[]>([])
  const [platforms, setPlatforms] = useState<{ id: string; name: string }[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Route | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  useEffect(() => {
    loadRoutes()
    loadPlatforms()
  }, [])

  const loadRoutes = async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getRoutes()
    if (result.error) {
      setError(result.error)
      setRoutes([])
    } else {
      setRoutes(result.data || [])
    }
    setLoading(false)
  }

  const loadPlatforms = async () => {
    const result = await apiClient.getPlatforms()
    if (!result.error) {
      setPlatforms((result.data || []).map((p: any) => ({ id: p.id, name: p.name })))
    }
  }

  const openCreate = () => {
    setEditing(null)
    setFormError(null)
    setModalOpen(true)
  }

  const openEdit = (route: Route) => {
    setEditing(route)
    setFormError(null)
    setModalOpen(true)
  }

  const handleSubmit = async (values: Record<string, any>) => {
    setSubmitting(true)
    setFormError(null)
    // Convert the enabled select value ("true"/"false") back to a boolean.
    const payload = { ...values }
    if (typeof payload.enabled === 'string') {
      payload.enabled = payload.enabled === 'true'
    }
    if (payload.priority === '' || payload.priority === undefined) {
      delete payload.priority
    }
    const result = editing
      ? await apiClient.updateRoute(editing.id, payload)
      : await apiClient.createRoute(payload)
    setSubmitting(false)
    if (result.error) {
      setFormError(result.error)
      return
    }
    setModalOpen(false)
    loadRoutes()
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('common.confirmDelete') || 'Delete this item?')) return
    const result = await apiClient.deleteRoute(id)
    if (!result.error) {
      loadRoutes()
    }
  }

  const platformOptions = platforms.map((p) => ({ value: p.id, label: p.name }))

  const fields: FieldConfig[] = [
    { key: 'name', label: t('common.name'), required: true },
    {
      key: 'platform_id',
      label: t('messages.platform'),
      type: 'select',
      options: platformOptions,
    },
    { key: 'priority', label: t('routes.priority'), type: 'number' },
    {
      key: 'target_type',
      label: t('routes.targetType') || 'Target Type',
      required: true,
    },
    { key: 'target_id', label: t('routes.targetId') || 'Target ID', required: true },
    { key: 'conditions', label: t('routes.conditions') },
    {
      key: 'enabled',
      label: t('routes.enabled'),
      type: 'select',
      options: [
        { value: 'true', label: 'true' },
        { value: 'false', label: 'false' },
      ],
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
          onClick={loadRoutes}
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
        <h1 className="text-2xl font-bold text-gray-900">{t('routes.title')}</h1>
        <button
          onClick={openCreate}
          className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4 mr-2" />
          {t('routes.create')}
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
                {t('messages.platform')}
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('routes.priority')}
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('routes.target')}
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('routes.enabled')}
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                {t('common.actions')}
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {routes.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                  {t('routes.noRoutes') || 'No routes found. Create your first route to get started.'}
                </td>
              </tr>
            ) : (
              routes.map((route) => (
                <tr key={route.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {route.name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {route.platform_id}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {route.priority}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate">
                    {route.target_type}: {route.target_id}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      route.enabled 
                        ? 'bg-green-100 text-green-800' 
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {route.enabled ? t('routes.enabled') : t('routes.disabled') || 'Disabled'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => openEdit(route)}
                      className="text-blue-600 hover:text-blue-900 mr-3"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(route.id)}
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
        title={editing ? t('routes.edit') || 'Edit Route' : t('routes.create')}
        open={modalOpen}
        fields={fields}
        initialValues={editing ? { ...editing, enabled: editing.enabled ? 'true' : 'false' } : { enabled: 'true', priority: 0 }}
        submitting={submitting}
        error={formError}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
      />
    </div>
  )
}
