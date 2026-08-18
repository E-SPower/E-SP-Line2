import { useTranslation } from 'react-i18next'
import { Save, Database, Globe, Shield, Bell, HardDrive, RefreshCw, Users, Trash2, Plus, X } from 'lucide-react'
import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import apiClient from '../api/client'
import Select from '../components/Select'

// Format a byte count into a human-readable string.
function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

interface Settings {
  general: {
    siteName: string
    language: string
    timezone: string
  }
  database: {
    driver: string
    host: string
    port: number
    name: string
  }
  security: {
    tokenExpiry: number
    maxLoginAttempts: number
    requireEmailVerification: boolean
  }
  notifications: {
    emailNotifications: boolean
    webhookUrl: string
    alertOnError: boolean
  }
}

// Tab keys rendered as the settings column tabs.
const TABS = ['general', 'database', 'security', 'notifications', 'storage', 'users'] as const
type TabKey = (typeof TABS)[number]

const TAB_ICONS: Record<TabKey, any> = {
  general: Globe,
  database: Database,
  security: Shield,
  notifications: Bell,
  storage: HardDrive,
  users: Users,
}

export default function Settings() {
  const { t, i18n } = useTranslation()
  const [searchParams] = useSearchParams()
  const urlTab = searchParams.get('tab')
  const [activeTab, setActiveTab] = useState<TabKey>(
    urlTab && (TABS as readonly string[]).includes(urlTab) ? (urlTab as TabKey) : 'general'
  )
  const [settings, setSettings] = useState<Settings>({
    general: {
      siteName: 'E-SP-Line2',
      language: 'zh-CN',
      timezone: 'Asia/Shanghai'
    },
    database: {
      driver: 'sqlite',
      host: 'localhost',
      port: 5432,
      name: 'e-sp-line2'
    },
    security: {
      tokenExpiry: 3600,
      maxLoginAttempts: 5,
      requireEmailVerification: false
    },
    notifications: {
      emailNotifications: true,
      webhookUrl: '',
      alertOnError: true
    }
  })
  const [loading, setLoading] = useState(false)
  const [saved, setSaved] = useState(false)

  // Storage stats
  const [storage, setStorage] = useState<any>(null)
  const [storageLoading, setStorageLoading] = useState(false)

  // User management
  const [users, setUsers] = useState<any[]>([])
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [usersLoading, setUsersLoading] = useState(false)
  const [addUserOpen, setAddUserOpen] = useState(false)
  const [newUser, setNewUser] = useState({ username: '', password: '', role: 'user' })
  const [addUserError, setAddUserError] = useState<string | null>(null)
  const [addUserSubmitting, setAddUserSubmitting] = useState(false)

  const loadStorage = async () => {
    setStorageLoading(true)
    const res = await apiClient.getStorageStats()
    setStorageLoading(false)
    if (!res.error) setStorage(res.data)
  }

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
    loadSettings()
    loadStorage()
    loadUsers()
  }, [])

  const loadSettings = async () => {
    try {
      // Load settings from local storage
      const savedSettings = localStorage.getItem('app_settings')
      if (savedSettings) {
        setSettings(JSON.parse(savedSettings))
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }

  const handleSave = async () => {
    setLoading(true)
    setSaved(false)
    
    try {
      // Save settings to local storage
      localStorage.setItem('app_settings', JSON.stringify(settings))
      
      // Update i18n language
      if (settings.general.language !== i18n.language) {
        await i18n.changeLanguage(settings.general.language)
      }
      
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (error) {
      console.error('Failed to save settings:', error)
    } finally {
      setLoading(false)
    }
  }

  const updateSetting = (category: keyof Settings, key: string, value: any) => {
    setSettings(prev => ({
      ...prev,
      [category]: {
        ...prev[category],
        [key]: value
      }
    }))
  }

  // ---- User management actions ----
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

  const tabLabel = (key: TabKey): string => {
    switch (key) {
      case 'general': return t('settings.general')
      case 'database': return t('settings.database')
      case 'security': return t('settings.security')
      case 'notifications': return t('settings.notifications')
      case 'storage': return t('settings.storageUsage')
      case 'users': return t('settings.userManagement')
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('settings.title')}</h1>
        <button
          onClick={handleSave}
          disabled={loading}
          className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center space-x-2"
        >
          <Save className="w-4 h-4" />
          <span>{loading ? t('common.saving') : t('common.save')}</span>
        </button>
      </div>

      {saved && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <p className="text-green-800">{t('settings.saved')}</p>
        </div>
      )}

      {/* Settings column tabs */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="flex overflow-x-auto border-b border-gray-200 dark:border-gray-700">
          {TABS.map((key) => {
            const Icon = TAB_ICONS[key]
            const isActive = activeTab === key
            return (
              <button
                key={key}
                onClick={() => setActiveTab(key)}
                className={`flex items-center px-5 py-3 text-sm font-medium whitespace-nowrap border-b-2 transition-colors ${
                  isActive
                    ? 'border-blue-600 text-blue-600 dark:text-blue-400'
                    : 'border-transparent text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-700'
                }`}
              >
                <Icon className="w-4 h-4 mr-2" />
                {tabLabel(key)}
              </button>
            )
          })}
        </div>
      </div>

      {/* General Settings */}
      {activeTab === 'general' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="p-6 space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.siteName')}
              </label>
              <input
                type="text"
                value={settings.general.siteName}
                onChange={(e) => updateSetting('general', 'siteName', e.target.value)}
                className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.language')}
              </label>
              <Select
                value={settings.general.language}
                onChange={(v) => updateSetting('general', 'language', v)}
                options={[
                  { value: 'zh-CN', label: '简体中文' },
                  { value: 'en-US', label: 'English' },
                ]}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.timezone')}
              </label>
              <Select
                value={settings.general.timezone}
                onChange={(v) => updateSetting('general', 'timezone', v)}
                options={[
                  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
                  { value: 'UTC', label: 'UTC' },
                  { value: 'America/New_York', label: 'America/New_York (UTC-5)' },
                ]}
              />
            </div>
          </div>
        </div>
      )}

      {/* Database Settings */}
      {activeTab === 'database' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="p-6 space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.databaseDriver')}
              </label>
              <Select
                value={settings.database.driver}
                onChange={(v) => updateSetting('database', 'driver', v)}
                options={[
                  { value: 'sqlite', label: 'SQLite' },
                  { value: 'postgres', label: 'PostgreSQL' },
                ]}
              />
            </div>

            {settings.database.driver === 'postgres' && (
              <>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('settings.host')}
                    </label>
                    <input
                      type="text"
                      value={settings.database.host}
                      onChange={(e) => updateSetting('database', 'host', e.target.value)}
                      className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('settings.port')}
                    </label>
                    <input
                      type="number"
                      value={settings.database.port}
                      onChange={(e) => updateSetting('database', 'port', parseInt(e.target.value))}
                      className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('settings.databaseName')}
                  </label>
                  <input
                    type="text"
                    value={settings.database.name}
                    onChange={(e) => updateSetting('database', 'name', e.target.value)}
                    className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Security Settings */}
      {activeTab === 'security' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="p-6 space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.tokenExpiry')} ({t('common.seconds')})
              </label>
              <input
                type="number"
                value={settings.security.tokenExpiry}
                onChange={(e) => updateSetting('security', 'tokenExpiry', parseInt(e.target.value))}
                className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.maxLoginAttempts')}
              </label>
              <input
                type="number"
                value={settings.security.maxLoginAttempts}
                onChange={(e) => updateSetting('security', 'maxLoginAttempts', parseInt(e.target.value))}
                className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('settings.requireEmailVerification')}
              </label>
              <input
                type="checkbox"
                checked={settings.security.requireEmailVerification}
                onChange={(e) => updateSetting('security', 'requireEmailVerification', e.target.checked)}
                className="w-4 h-4 text-blue-600 bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600 rounded focus:ring-blue-500"
              />
            </div>
          </div>
        </div>
      )}

      {/* Notification Settings */}
      {activeTab === 'notifications' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="p-6 space-y-4">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('settings.emailNotifications')}
              </label>
              <input
                type="checkbox"
                checked={settings.notifications.emailNotifications}
                onChange={(e) => updateSetting('notifications', 'emailNotifications', e.target.checked)}
                className="w-4 h-4 text-blue-600 bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600 rounded focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('settings.webhookUrl')}
              </label>
              <input
                type="url"
                value={settings.notifications.webhookUrl}
                onChange={(e) => updateSetting('notifications', 'webhookUrl', e.target.value)}
                placeholder="https://example.com/webhook"
                className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('settings.alertOnError')}
              </label>
              <input
                type="checkbox"
                checked={settings.notifications.alertOnError}
                onChange={(e) => updateSetting('notifications', 'alertOnError', e.target.checked)}
                className="w-4 h-4 text-blue-600 bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600 rounded focus:ring-blue-500"
              />
            </div>
          </div>
        </div>
      )}

      {/* Storage Usage */}
      {activeTab === 'storage' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {t('settings.storageUsage')}
            </h2>
            <button
              onClick={loadStorage}
              className="inline-flex items-center px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              <RefreshCw className={`w-4 h-4 mr-1 ${storageLoading ? 'animate-spin' : ''}`} />
              {t('settings.refreshStorage')}
            </button>
          </div>
          <div className="p-6">
            {storageLoading ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">{t('settings.storageLoading')}</p>
            ) : !storage ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">{t('common.loading')}</p>
            ) : (
              <>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
                  <p className="text-xs text-gray-500 dark:text-gray-400">{t('settings.dbSize')}</p>
                  <p className="text-xl font-bold text-gray-900 dark:text-gray-100">
                    {formatBytes(storage.db_size_bytes || 0)}
                  </p>
                </div>
                <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
                  <p className="text-xs text-gray-500 dark:text-gray-400">{t('settings.dataDirSize')}</p>
                  <p className="text-xl font-bold text-gray-900 dark:text-gray-100">
                    {formatBytes(storage.data_dir_bytes || 0)}
                  </p>
                </div>
                <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
                  <p className="text-xs text-gray-500 dark:text-gray-400">{t('settings.totalRows')}</p>
                  <p className="text-xl font-bold text-gray-900 dark:text-gray-100">
                    {Object.values(storage.tables || {}).reduce((a: number, b: any) => a + (Number(b) || 0), 0)}
                  </p>
                </div>
              </div>

              <div>
                <p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('settings.tableRows')}
                </p>
                <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
                  {Object.entries(storage.tables || {}).map(([name, count]: [string, any]) => (
                    <div key={name} className="bg-gray-50 dark:bg-gray-700 rounded px-3 py-2">
                      <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{name}</p>
                      <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">{Number(count) || 0}</p>
                    </div>
                  ))}
                </div>
              </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* User Management */}
      {activeTab === 'users' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {t('settings.userManagement')}
            </h2>
            <div className="flex items-center space-x-2">
              <button
                onClick={() => { setAddUserOpen(true); setAddUserError(null) }}
                className="inline-flex items-center px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                <Plus className="w-4 h-4 mr-1" />
                {t('settings.addUser')}
              </button>
              <button
                onClick={loadUsers}
                className="inline-flex items-center px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                <RefreshCw className={`w-4 h-4 mr-1 ${usersLoading ? 'animate-spin' : ''}`} />
                {t('common.refresh')}
              </button>
            </div>
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
      )}

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
