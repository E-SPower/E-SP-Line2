import { useEffect, useMemo, useState } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  BookOpen,
  ChevronLeft,
  ChevronRight,
  GitBranch,
  Globe,
  Languages,
  LayoutDashboard,
  LogOut,
  Menu,
  MessageSquare,
  Monitor,
  Moon,
  Server,
  Settings as SettingsIcon,
  Sun,
  X,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTheme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import { ConnectionBadge } from '@/components/connection-badge'

type NavGroup = 'business' | 'ops' | 'settings'

interface NavItem {
  path: string
  icon: typeof LayoutDashboard
  labelKey: string
  group: NavGroup
}

const NAV_ITEMS: NavItem[] = [
  { path: '/dashboard', icon: LayoutDashboard, labelKey: 'nav.dashboard', group: 'business' },
  { path: '/platforms', icon: Globe, labelKey: 'nav.platforms', group: 'business' },
  { path: '/instances', icon: Server, labelKey: 'nav.instances', group: 'business' },
  { path: '/messages', icon: MessageSquare, labelKey: 'nav.messages', group: 'business' },
  { path: '/routes', icon: GitBranch, labelKey: 'nav.routes', group: 'business' },
  // The `ops` group holds exactly this one entry; the dashboard rail's ops
  // summary folded into it, so there is no second ops destination.
  { path: '/ops', icon: Monitor, labelKey: 'nav.ops', group: 'ops' },
  { path: '/settings', icon: SettingsIcon, labelKey: 'nav.settings', group: 'settings' },
  { path: '/docs', icon: BookOpen, labelKey: 'nav.docs', group: 'settings' },
]

/** Nav groups, in sidebar order. Every existing route is reachable here. */
const NAV_GROUPS: { id: NavGroup; labelKey: string }[] = [
  { id: 'business', labelKey: 'nav.group.business' },
  { id: 'ops', labelKey: 'nav.group.ops' },
  { id: 'settings', labelKey: 'nav.group.settings' },
]

const COLLAPSED_KEY = 'espl.sidebar-collapsed'

export default function Layout() {
  const { t, i18n } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const { resolvedTheme, toggleTheme } = useTheme()

  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(COLLAPSED_KEY) === '1')
  const [mobileOpen, setMobileOpen] = useState(false)

  useEffect(() => {
    setMobileOpen(false)
  }, [location.pathname])

  useEffect(() => {
    localStorage.setItem(COLLAPSED_KEY, collapsed ? '1' : '0')
  }, [collapsed])

  useEffect(() => {
    if (!mobileOpen) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMobileOpen(false)
    }
    window.addEventListener('keydown', onKey)
    // Stop the page behind the drawer from scrolling.
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      window.removeEventListener('keydown', onKey)
      document.body.style.overflow = previousOverflow
    }
  }, [mobileOpen])

  const current = useMemo(
    () => NAV_ITEMS.find((item) => location.pathname.startsWith(item.path)),
    [location.pathname],
  )

  const currentGroup = useMemo(
    () => NAV_GROUPS.find((group) => group.id === current?.group),
    [current],
  )

  const toggleLanguage = () => {
    void i18n.changeLanguage(i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN')
  }

  const handleLogout = () => {
    localStorage.removeItem('auth_token')
    navigate('/login')
  }

  const renderSidebar = (mobile: boolean) => {
    const compact = collapsed && !mobile
    return (
      <div className="flex h-full flex-col bg-surface-sunken">
        {/* Brand */}
        <div
          className={cn(
            'flex h-topbar shrink-0 items-center gap-2.5 border-b border-border px-3',
            compact && 'justify-center px-2',
          )}
        >
          <Link to="/dashboard" className="flex min-w-0 items-center gap-2.5 rounded-md" title={t('app.title')}>
            <span
              aria-hidden
              className="grid size-8 shrink-0 place-items-center rounded-md border border-border bg-primary-foreground dark:bg-white"
            >
              <img src="/espl-logo.png" alt="" className="size-5 object-contain" />
            </span>
            {!compact && (
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold tracking-tight text-text-primary">
                  {t('app.title')}
                </span>
                <span className="block truncate text-xs text-text-tertiary">{t('app.subtitle')}</span>
              </span>
            )}
          </Link>
          {mobile && (
            <Button
              variant="ghost"
              size="icon-sm"
              className="ml-auto"
              onClick={() => setMobileOpen(false)}
              aria-label={t('common.close')}
            >
              <X />
            </Button>
          )}
        </div>

        {/* Grouped navigation */}
        <nav aria-label={t('nav.primary')} className="flex-1 overflow-y-auto px-2 py-3">
          {NAV_GROUPS.map((group) => {
            const items = NAV_ITEMS.filter((item) => item.group === group.id)
            if (items.length === 0) return null
            return (
              <div key={group.id} className="mb-4 last:mb-0">
                {!compact && (
                  <p className="px-2 pb-1 text-2xs font-semibold uppercase tracking-wider text-text-tertiary">
                    {t(group.labelKey)}
                  </p>
                )}
                <ul className="space-y-0.5">
                  {items.map((item) => {
                    const Icon = item.icon
                    return (
                      <li key={item.path}>
                        <NavLink
                          to={item.path}
                          title={compact ? t(item.labelKey) : undefined}
                          className={({ isActive }) =>
                            cn(
                              'relative flex h-9 items-center gap-2.5 rounded-md px-2 text-sm transition-colors',
                              isActive
                                ? 'bg-surface-hover font-medium text-text-primary'
                                : 'text-text-secondary hover:bg-surface-hover/60 hover:text-text-primary',
                              compact && 'justify-center px-0',
                            )
                          }
                        >
                          <Icon className="size-4 shrink-0" />
                          {!compact && <span className="truncate">{t(item.labelKey)}</span>}
                        </NavLink>
                      </li>
                    )
                  })}
                </ul>
              </div>
            )
          })}
        </nav>

        {/* Utilities */}
        <div className={cn('shrink-0 border-t border-border p-2', compact && 'flex flex-col items-center gap-1')}>
          <div className={cn('flex items-center gap-1', compact && 'flex-col')}>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={toggleLanguage}
              aria-label={t('common.switchLanguage')}
              title={i18n.language === 'zh-CN' ? 'English' : '中文'}
            >
              <Languages />
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={toggleTheme}
              aria-label={t('common.toggleTheme')}
              title={resolvedTheme === 'dark' ? t('common.themeLight') : t('common.themeDark')}
            >
              {resolvedTheme === 'dark' ? <Sun /> : <Moon />}
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleLogout}
              aria-label={t('auth.logout')}
              title={t('auth.logout')}
            >
              <LogOut />
            </Button>
          </div>
          {!compact && <p className="mt-1 px-1 text-xs text-text-tertiary">{t('app.version')}</p>}
          {!mobile && (
            <Button
              variant="ghost"
              size="sm"
              className="mt-1 w-full min-w-0 justify-start gap-1.5 px-1 py-1 text-text-tertiary"
              onClick={() => setCollapsed((value) => !value)}
              aria-label={collapsed ? t('nav.expand') : t('nav.collapse')}
              title={collapsed ? t('nav.expand') : t('nav.collapse')}
            >
              {collapsed ? <ChevronRight /> : <ChevronLeft />}
              {/* `truncate`: the label is `nowrap`, so a long one (English at a
                  large root size) used to paint straight over the main column
                  instead of ellipsising inside the sidebar. */}
              {!collapsed && <span className="min-w-0 break-words">{t('nav.collapse')}</span>}
            </Button>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-3 focus:top-3 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-sm focus:text-primary-foreground"
      >
        {t('common.skipToContent')}
      </a>

      {/* Desktop sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-30 hidden border-r border-border bg-surface-sunken lg:block',
          collapsed ? 'w-sidebar-collapsed' : 'w-sidebar',
        )}
      >
        {renderSidebar(false)}
      </aside>

      {/* Mobile drawer */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div
            className="absolute inset-0 animate-fade-in bg-text-primary/40"
            onClick={() => setMobileOpen(false)}
            aria-hidden
          />
          <aside
            role="dialog"
            aria-modal="true"
            aria-label={t('nav.primary')}
            className="absolute inset-y-0 left-0 w-72 animate-slide-in-left shadow-overlay"
          >
            {renderSidebar(true)}
          </aside>
        </div>
      )}

      <div
        className={cn(
          'flex min-h-screen flex-col transition-[padding] duration-200',
          collapsed ? 'lg:pl-sidebar-collapsed' : 'lg:pl-sidebar',
        )}
      >
        {/* Fixed top bar. The bar itself is full-bleed so its rule reaches both
            edges, but its contents sit on `.shell-rail` — the exact rail `main`
            below uses — so the breadcrumb and the page body share one left edge
            at every width instead of agreeing only below the container cap.
            `min-w-0` on the bar and its left side lets the breadcrumb truncate
            instead of pushing the trailing actions out. */}
        <header className="sticky top-0 z-20 min-w-0 border-b border-border bg-background">
          <div className="shell-rail flex h-topbar min-w-0 items-center gap-3">
            <Button
              variant="ghost"
              size="icon-sm"
              className="shrink-0 lg:hidden"
              onClick={() => setMobileOpen(true)}
              aria-label={t('nav.openMenu')}
            >
              <Menu />
            </Button>

            <nav aria-label={t('nav.breadcrumb')} className="min-w-0 flex-1">
              <ol className="flex min-w-0 items-center gap-1.5 text-sm">
                <li className="hidden shrink-0 text-text-tertiary sm:block">{t('app.title')}</li>
                {currentGroup && (
                  <>
                    <li aria-hidden className="hidden shrink-0 text-text-disabled sm:block">
                      /
                    </li>
                    <li className="hidden shrink-0 text-text-tertiary sm:block">{t(currentGroup.labelKey)}</li>
                  </>
                )}
                {current && (
                  <li aria-hidden className="hidden shrink-0 text-text-disabled sm:block">
                    /
                  </li>
                )}
                <li className="min-w-0 truncate font-medium text-text-primary">
                  {current ? t(current.labelKey) : t('app.title')}
                </li>
              </ol>
            </nav>

            <div className="flex shrink-0 items-center gap-1.5">
              <ConnectionBadge />
              <Button variant="ghost" size="icon-sm" onClick={toggleTheme} aria-label={t('common.toggleTheme')}>
                {resolvedTheme === 'dark' ? <Sun /> : <Moon />}
              </Button>
              <Link
                to="/docs"
                className="grid size-8 place-items-center rounded-md text-text-secondary transition-colors hover:bg-surface-hover hover:text-text-primary"
                aria-label={t('nav.docs')}
                title={t('nav.docs')}
              >
                <BookOpen className="size-4" />
              </Link>
            </div>
          </div>
        </header>

        <main id="main" className="min-w-0 flex-1">
          {/* Same rail as the bar above, so the page body, the page `h1` and
              the cards inside it line up with the breadcrumb. The top inset
              steps 24/32/48: at phone width the horizontal gutter is also 24,
              but from a desktop bar down the page needs more air than the
              sides or the `h1` reads as glued to the top bar. */}
          <div className="shell-rail stack py-6 sm:py-8 lg:py-[clamp(56px,5vw,96px)]">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
