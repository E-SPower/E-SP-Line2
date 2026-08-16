import { Outlet, Link, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  LayoutDashboard,
  Package,
  Server,
  MessageSquare,
  GitBranch,
  BookOpen,
  LogOut
} from 'lucide-react'

export default function Layout() {
  const { t, i18n } = useTranslation()
  const location = useLocation()

  const navItems = [
    { path: '/dashboard', icon: LayoutDashboard, label: t('nav.dashboard') },
    { path: '/adapters', icon: Package, label: t('nav.adapters') },
    { path: '/instances', icon: Server, label: t('nav.instances') },
    { path: '/messages', icon: MessageSquare, label: t('nav.messages') },
    { path: '/routes', icon: GitBranch, label: t('nav.routes') },
    { path: '/docs', icon: BookOpen, label: t('nav.docs') },
  ]

  const toggleLanguage = () => {
    const newLang = i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN'
    i18n.changeLanguage(newLang)
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 h-full w-64 bg-white border-r border-gray-200">
        <div className="p-6">
          <h1 className="text-xl font-bold text-gray-900">{t('app.title')}</h1>
          <p className="text-sm text-gray-500 mt-1">{t('app.subtitle')}</p>
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
                    ? 'bg-blue-50 text-blue-700 border-r-2 border-blue-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`}
              >
                <Icon className="w-5 h-5 mr-3" />
                {item.label}
              </Link>
            )
          })}
        </nav>

        <div className="absolute bottom-0 left-0 right-0 p-6 border-t border-gray-200">
          <div className="flex items-center justify-between">
            <button
              onClick={toggleLanguage}
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              {i18n.language === 'zh-CN' ? 'English' : '中文'}
            </button>
            <button className="text-gray-600 hover:text-gray-900">
              <LogOut className="w-5 h-5" />
            </button>
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
