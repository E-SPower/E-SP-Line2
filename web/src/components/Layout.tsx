import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  LayoutDashboard,
  Package,
  Server,
  GitBranch,
  BookOpen,
  LogOut,
  Settings,
  Moon,
  Sun,
  Languages,
  Users,
  PlugZap,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTheme } from '../hooks/useTheme'
import apiClient from '../api/client'

export default function Layout() {
  const { t, i18n } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const [theme, toggleTheme] = useTheme()

  const navItems = [
    { path: '/dashboard', icon: LayoutDashboard, label: t('nav.dashboard') },
    { path: '/adapters', icon: Package, label: t('nav.adapters') },
    { path: '/instances', icon: Server, label: t('nav.instances') },
    { path: '/adapter-gateway', icon: PlugZap, label: t('nav.adapterGateway') },
    { path: '/routes', icon: GitBranch, label: t('nav.routes') },
  ]

  const toggleLanguage = () => {
    const newLang = i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN'
    localStorage.setItem('esp-lang', newLang)
    i18n.changeLanguage(newLang)
  }

  const handleLogout = () => {
    apiClient.clearToken()
    navigate('/login')
  }

  // Settings dropdown state
  const [settingsOpen, setSettingsOpen] = useState(false)
  const settingsRef = useRef<HTMLDivElement>(null)

  // Close dropdown on outside click
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (settingsRef.current && !settingsRef.current.contains(e.target as Node)) {
        setSettingsOpen(false)
      }
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const darkMode = theme === 'dark'

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 transition-colors">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 h-full w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 transition-colors">
        <div className="p-6">
          <div className="flex items-center space-x-3">
            <img
              src="/espl-logo.png"
              alt="E-SP-Line2"
              className="h-10 w-10 object-contain rounded"
            />
            <div>
              <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">{t('app.title')}</h1>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{t('app.subtitle')}</p>
            </div>
          </div>
        </div>
        
        <nav className="mt-6">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive = location.pathname === item.path
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center px-6 py-3 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border-r-2 border-blue-700'
                    : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-100'
                }`}
              >
                <Icon className="w-5 h-5 mr-3" />
                {item.label}
              </Link>
            )
          })}
        </nav>

        <div className="absolute bottom-0 left-0 right-0 p-6 border-t border-gray-200 dark:border-gray-700">
          {/* Docs center - sits with system settings at the bottom */}
          <Link
            to="/docs"
            className={`w-full flex items-center px-3 py-2 rounded-lg text-sm transition-colors mb-1 ${
              location.pathname.startsWith('/docs')
                ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-gray-100'
            }`}
          >
            <BookOpen className="w-4 h-4 mr-2" />
            {t('nav.docs')}
          </Link>

          {/* Settings dropdown */}
          <div className="relative" ref={settingsRef}>
            <button
              onClick={() => setSettingsOpen((o) => !o)}
              className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              <span className="flex items-center">
                <Settings className="w-4 h-4 mr-2" />
                {t('settings.title')}
              </span>
              <span className="text-gray-400">{settingsOpen ? '▲' : '▼'}</span>
            </button>

            {settingsOpen && (
              <div className="absolute bottom-full left-0 right-0 mb-2 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-xl shadow-2xl p-1.5 overflow-hidden">
                {/* Dark mode */}
                <button
                  onClick={toggleTheme}
                  className="w-full flex items-center px-3 py-2.5 text-sm rounded-lg text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  {darkMode ? <Sun className="w-4 h-4 mr-2" /> : <Moon className="w-4 h-4 mr-2" />}
                  {darkMode ? t('settings.lightMode') : t('settings.darkMode')}
                </button>
                {/* Language */}
                <button
                  onClick={toggleLanguage}
                  className="w-full flex items-center px-3 py-2.5 text-sm rounded-lg text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  <Languages className="w-4 h-4 mr-2" />
                  {i18n.language === 'zh-CN' ? 'English' : '中文'}
                </button>
                {/* Settings page */}
                <Link
                  to="/settings"
                  onClick={() => setSettingsOpen(false)}
                  className="w-full flex items-center px-3 py-2.5 text-sm rounded-lg text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  <Settings className="w-4 h-4 mr-2" />
                  {t('settings.title')}
                </Link>
                {/* User management */}
                <Link
                  to="/settings?tab=users"
                  onClick={() => setSettingsOpen(false)}
                  className="w-full flex items-center px-3 py-2.5 text-sm rounded-lg text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  <Users className="w-4 h-4 mr-2" />
                  {t('settings.userManagement')}
                </Link>
                <div className="my-1 border-t border-gray-100 dark:border-gray-600" />
                {/* Logout */}
                <button
                  onClick={handleLogout}
                  className="w-full flex items-center px-3 py-2.5 text-sm rounded-lg text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 transition-colors"
                >
                  <LogOut className="w-4 h-4 mr-2" />
                  {t('auth.logout')}
                </button>
              </div>
            )}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="ml-64 p-8">
        <Outlet />
      </main>
    </div>
  )
}
