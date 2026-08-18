import { useTranslation } from 'react-i18next'
import { Users as UsersIcon, RefreshCw, Plus, Trash2, X } from 'lucide-react'
import { useState, useEffect } from 'react'
import apiClient from '../api/client'
import Select from '../components/Select'

export default function Users() {
  const { t } = useTranslation()

  // User management
  const [users, setUsers] = useState<any[]>([])
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [usersLoading, setUsersLoading] = useState(false)
  const [addUserOpen, setAddUserOpen] = useState(false)
  const [newUser, setNewUser] = useState({ username: '', password: '', role: 'user' })
  const [addUserError, setAddUserError] = useState<string | null>(null)
  const [addUserSubmitting, setAddUserSubmitting] = useState(false)

  const loadUsers = async () => {
    setUsersLoading(true)
    const [u, reg] = await Promise.all([
      apiClient.listUsers(),
      apiClient.getRegistrationStatus(),
    ])
    setUsersLoading(false)
    if (!u.error) setUsers(u.data || [])
    if (!reg.error) setRegistrationEnabled(reg.data !== false)
  }

  useEffect(() => {
    loadUsers()
  }, [])

  const handleToggleRegistration = async (enabled: boolean) => {
    const res = await apiClient.setRegistrationStatus(enabled)
    if (res.error) {
      alert(`操作失败: ${res.error}`)
      return
    }
    setRegistrationEnabled(enabled)
  }

  const handleUserRole = async (id: string, role: string) => {
    const res = await apiClient.updateUserRole(id, role)
    if (res.error) {
      alert(`操作失败: ${res.error}`)
      return
    }
    loadUsers()
  }

  const handleUserStatus = async (id: string, status: string) => {
    const res = await apiClient.updateUserStatus(id, status)
    if (res.error) {
      alert(`操作失败: ${res.error}`)
      return
    }
    loadUsers()
  }

  const handleDeleteUser = async (id: string, username: string) => {
    if (!window.confirm(`确定删除用户 "${username}"？`)) return
    const res = await apiClient.deleteUser(id)
    if (res.error) {
      alert(`删除失败: ${res.error}`)
      return
    }
    loadUsers()
  }

  const handleCreateUser = async () => {
    setAddUserError(null)
    if (!newUser.username.trim() || !newUser.password) {
      setAddUserError('用户名和密码不能为空')
      return
    }
    setAddUserSubmitting(true)
    const res = await apiClient.createUser(newUser.username.trim(), newUser.password, newUser.role)
    setAddUserSubmitting(false)
    if (res.error) {
      setAddUserError(res.error)
      return
    }
    setAddUserOpen(false)
    setNewUser({ username: '', password: '', role: 'user' })
    loadUsers()
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('settings.userManagement')}</h1>
        <div className="flex items-center space-x-2">
          <button
            onClick={() => { setAddUserOpen(true); setAddUserError(null) }}
            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <Plus className="w-4 h-4 mr-1" />
            {t('settings.addUser')}
          </button>
          <button
            onClick={loadUsers}
            className="inline-flex items-center px-3 py-2 text-sm text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <RefreshCw className={`w-4 h-4 mr-1 ${usersLoading ? 'animate-spin' : ''}`} />
            {t('common.refresh')}
          </button>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center space-x-2">
          <UsersIcon className="w-5 h-5 text-indigo-500" />
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {t('settings.userManagement')}
          </h2>
        </div>
        <div className="p-6 space-y-4">
          {/* Registration toggle */}
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('settings.allowRegistration')}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                {t('settings.allowRegistrationHint')}
              </p>
            </div>
            <button
              type="button"
              onClick={() => handleToggleRegistration(!registrationEnabled)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                registrationEnabled ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
              }`}
              role="switch"
              aria-checked={registrationEnabled}
            >
              <span
                className={`inline-block h-5 w-5 transform rounded-full bg-white transition-transform ${
                  registrationEnabled ? 'translate-x-5' : 'translate-x-0.5'
                }`}
              />
            </button>
          </div>

          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            {usersLoading ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">{t('common.loading')}</p>
            ) : users.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">{t('settings.noUsers')}</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead className="bg-gray-50 dark:bg-gray-700">
                    <tr>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        {t('common.name')}
                      </th>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        {t('settings.role')}
                      </th>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        {t('common.status')}
                      </th>
                      <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        {t('common.actions')}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                    {users.map((u) => (
                      <tr key={u.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                        <td className="px-4 py-2 text-sm text-gray-900 dark:text-gray-100">
                          {u.username}
                          {u.role === 'admin' && (
                            <span className="ml-2 px-1.5 py-0.5 text-xs rounded bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300">
                              admin
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
                          <button
                            onClick={() => handleUserRole(u.id, u.role === 'admin' ? 'user' : 'admin')}
                            className="text-blue-600 dark:text-blue-400 hover:underline"
                            title={u.role === 'admin' ? '降为普通用户' : '提升为管理员'}
                          >
                            {u.role === 'admin' ? '管理员' : '普通用户'}
                          </button>
                        </td>
                        <td className="px-4 py-2 text-sm">
                          <button
                            onClick={() => handleUserStatus(u.id, u.status === 'active' ? 'disabled' : 'active')}
                            className={`px-2 py-0.5 text-xs rounded-full ${
                              u.status === 'active'
                                ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300'
                                : 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300'
                            }`}
                          >
                            {u.status === 'active' ? '启用' : '禁用'}
                          </button>
                        </td>
                        <td className="px-4 py-2 text-right">
                          <button
                            onClick={() => handleDeleteUser(u.id, u.username)}
                            className="text-red-600 dark:text-red-400 hover:text-red-900 dark:hover:text-red-300"
                            title="删除用户"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
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

      {/* Add User Modal */}
      {addUserOpen && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen px-4">
            <div className="fixed inset-0 bg-black bg-opacity-50" onClick={() => setAddUserOpen(false)} />
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md">
              <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {t('settings.addUser')}
                </h2>
                <button
                  type="button"
                  onClick={() => setAddUserOpen(false)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="px-6 py-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('auth.username')}
                  </label>
                  <input
                    type="text"
                    value={newUser.username}
                    onChange={(e) => setNewUser({ ...newUser, username: e.target.value })}
                    className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('auth.password')}
                  </label>
                  <input
                    type="password"
                    value={newUser.password}
                    onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
                    className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('settings.role')}
                  </label>
                  <Select
                    value={newUser.role}
                    onChange={(v) => setNewUser({ ...newUser, role: v })}
                    options={[
                      { value: 'user', label: '普通用户' },
                      { value: 'admin', label: '管理员' },
                    ]}
                  />
                </div>
                {addUserError && (
                  <div className="rounded-md bg-red-50 dark:bg-red-900/30 p-3">
                    <p className="text-sm text-red-800 dark:text-red-300">{addUserError}</p>
                  </div>
                )}
                <div className="flex justify-end space-x-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setAddUserOpen(false)}
                    className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    {t('common.cancel')}
                  </button>
                  <button
                    type="button"
                    onClick={handleCreateUser}
                    disabled={addUserSubmitting}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                  >
                    {addUserSubmitting ? t('common.saving') : t('common.save')}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
