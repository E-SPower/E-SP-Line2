import { useTranslation } from 'react-i18next'
import { Save, Database, Globe, Shield, Bell } from 'lucide-react'
import { useState, useEffect } from 'react'
import apiClient from '../api/client'

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

export default function Settings() {
  const { t, i18n } = useTranslation()
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

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async () => {
    try {
      // Load settings from API (mock for now)
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
      // Save settings to localStorage (mock for now)
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">{t('settings.title')}</h1>
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

      {/* General Settings */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center space-x-2">
          <Globe className="w-5 h-5 text-blue-500" />
          <h2 className="text-lg font-semibold text-gray-900">{t('settings.general')}</h2>
        </div>
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.siteName')}
            </label>
            <input
              type="text"
              value={settings.general.siteName}
              onChange={(e) => updateSetting('general', 'siteName', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.language')}
            </label>
            <select
              value={settings.general.language}
              onChange={(e) => updateSetting('general', 'language', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="zh-CN">简体中文</option>
              <option value="en-US">English</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.timezone')}
            </label>
            <select
              value={settings.general.timezone}
              onChange={(e) => updateSetting('general', 'timezone', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="Asia/Shanghai">Asia/Shanghai (UTC+8)</option>
              <option value="UTC">UTC</option>
              <option value="America/New_York">America/New_York (UTC-5)</option>
            </select>
          </div>
        </div>
      </div>

      {/* Database Settings */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center space-x-2">
          <Database className="w-5 h-5 text-green-500" />
          <h2 className="text-lg font-semibold text-gray-900">{t('settings.database')}</h2>
        </div>
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.databaseDriver')}
            </label>
            <select
              value={settings.database.driver}
              onChange={(e) => updateSetting('database', 'driver', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="sqlite">SQLite</option>
              <option value="postgres">PostgreSQL</option>
            </select>
          </div>

          {settings.database.driver === 'postgres' && (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    {t('settings.host')}
                  </label>
                  <input
                    type="text"
                    value={settings.database.host}
                    onChange={(e) => updateSetting('database', 'host', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    {t('settings.port')}
                  </label>
                  <input
                    type="number"
                    value={settings.database.port}
                    onChange={(e) => updateSetting('database', 'port', parseInt(e.target.value))}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  {t('settings.databaseName')}
                </label>
                <input
                  type="text"
                  value={settings.database.name}
                  onChange={(e) => updateSetting('database', 'name', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </>
          )}
        </div>
      </div>

      {/* Security Settings */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center space-x-2">
          <Shield className="w-5 h-5 text-purple-500" />
          <h2 className="text-lg font-semibold text-gray-900">{t('settings.security')}</h2>
        </div>
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.tokenExpiry')} ({t('common.seconds')})
            </label>
            <input
              type="number"
              value={settings.security.tokenExpiry}
              onChange={(e) => updateSetting('security', 'tokenExpiry', parseInt(e.target.value))}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.maxLoginAttempts')}
            </label>
            <input
              type="number"
              value={settings.security.maxLoginAttempts}
              onChange={(e) => updateSetting('security', 'maxLoginAttempts', parseInt(e.target.value))}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium text-gray-700">
              {t('settings.requireEmailVerification')}
            </label>
            <input
              type="checkbox"
              checked={settings.security.requireEmailVerification}
              onChange={(e) => updateSetting('security', 'requireEmailVerification', e.target.checked)}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
          </div>
        </div>
      </div>

      {/* Notification Settings */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center space-x-2">
          <Bell className="w-5 h-5 text-orange-500" />
          <h2 className="text-lg font-semibold text-gray-900">{t('settings.notifications')}</h2>
        </div>
        <div className="p-6 space-y-4">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium text-gray-700">
              {t('settings.emailNotifications')}
            </label>
            <input
              type="checkbox"
              checked={settings.notifications.emailNotifications}
              onChange={(e) => updateSetting('notifications', 'emailNotifications', e.target.checked)}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('settings.webhookUrl')}
            </label>
            <input
              type="url"
              value={settings.notifications.webhookUrl}
              onChange={(e) => updateSetting('notifications', 'webhookUrl', e.target.value)}
              placeholder="https://example.com/webhook"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium text-gray-700">
              {t('settings.alertOnError')}
            </label>
            <input
              type="checkbox"
              checked={settings.notifications.alertOnError}
              onChange={(e) => updateSetting('notifications', 'alertOnError', e.target.checked)}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
          </div>
        </div>
      </div>
    </div>
  )
}
