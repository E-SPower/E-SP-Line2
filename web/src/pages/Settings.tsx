import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bell, Database, Globe, Save, Shield, Trash2, UserPlus, Users } from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { Card, CardBody, CardHeader, CardTitle } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { FormField, controlClassName } from '../components/ui/form-field'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'
import { Badge } from '../components/ui/badge'
import { ErrorState } from '../components/ui/error-state'
import { ConfirmDialog } from '../components/ui/confirm-dialog'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../components/ui/dialog'
import { TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TableShell } from '../components/ui/table-shell'
import { cn } from '@/lib/cn'

interface Settings {
  general: { siteName: string; language: string; timezone: string }
  database: { driver: string; host: string; port: number; name: string }
  security: { tokenExpiry: number; maxLoginAttempts: number; requireEmailVerification: boolean }
  notifications: { emailNotifications: boolean; webhookUrl: string; alertOnError: boolean }
}

const DEFAULTS: Settings = {
  general: { siteName: 'E-SP-Line2', language: 'zh-CN', timezone: 'Asia/Shanghai' },
  database: { driver: 'sqlite', host: 'localhost', port: 5432, name: 'e-sp-line2' },
  security: { tokenExpiry: 3600, maxLoginAttempts: 5, requireEmailVerification: false },
  notifications: { emailNotifications: true, webhookUrl: '', alertOnError: true },
}

type SectionId = 'general' | 'database' | 'security' | 'notifications' | 'users'

interface Account {
  id: string
  username: string
  email: string
  role: string
  status: string
}

const SECTIONS: { id: SectionId; icon: typeof Globe; labelKey: string; hintKey: string }[] = [
  { id: 'general', icon: Globe, labelKey: 'settings.general', hintKey: 'settings.generalHint' },
  { id: 'database', icon: Database, labelKey: 'settings.database', hintKey: 'settings.databaseHint' },
  { id: 'security', icon: Shield, labelKey: 'settings.security', hintKey: 'settings.securityHint' },
  { id: 'notifications', icon: Bell, labelKey: 'settings.notifications', hintKey: 'settings.notificationsHint' },
  { id: 'users', icon: Users, labelKey: 'settings.userManagement', hintKey: 'settings.userManagementHint' },
]

/** A labelled checkbox row, since these are switches rather than text fields. */
function SwitchField({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string
  hint?: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <div className="flex items-start justify-between gap-4 rounded-lg border border-border px-4 py-3">
      <div className="min-w-0">
        <p className="text-sm font-medium text-text-primary">{label}</p>
        {hint && <p className="mt-0.5 text-sm text-text-tertiary">{hint}</p>}
      </div>
      {/* shadcn Switch geometry: 36x20 track, 16px thumb, ring on focus. */}
      <label className="relative mt-0.5 inline-flex shrink-0 cursor-pointer items-center rounded-full">
        <input
          type="checkbox"
          role="switch"
          className="peer sr-only"
          checked={checked}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span className="h-5 w-9 rounded-full bg-input transition-colors peer-checked:bg-primary peer-focus-visible:ring-2 peer-focus-visible:ring-ring/50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50" />
        <span className="pointer-events-none absolute left-0.5 top-0.5 size-4 rounded-full bg-surface-raised shadow-xs transition-transform peer-checked:translate-x-4" />
      </label>
    </div>
  )
}

export default function Settings() {
  const { t, i18n } = useTranslation()
  const [settings, setSettings] = useState<Settings>(DEFAULTS)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [active, setActive] = useState<SectionId>('general')

  // ---- user management: server state, unlike the local settings above ----
  const [accounts, setAccounts] = useState<Account[]>([])
  const [accountsLoading, setAccountsLoading] = useState(false)
  const [accountsError, setAccountsError] = useState<string | null>(null)
  const [accountsNotice, setAccountsNotice] = useState<string | null>(null)
  const [allowRegistration, setAllowRegistration] = useState(true)
  const [registrationSaving, setRegistrationSaving] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState({ username: '', password: '' })
  const [createErrors, setCreateErrors] = useState<Record<string, string>>({})
  const [createError, setCreateError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [deletingUser, setDeletingUser] = useState<Account | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deletePending, setDeletePending] = useState(false)
  const [statusPendingId, setStatusPendingId] = useState<string | null>(null)

  const loadAccounts = useCallback(async () => {
    setAccountsLoading(true)
    setAccountsError(null)
    const result = await apiClient.listUsers()
    setAccountsLoading(false)
    if (result.error) {
      setAccountsError(result.error === 'admin access required' ? t('settings.adminOnly') : result.error)
      setAccounts([])
      return
    }
    setAccounts(result.data || [])
  }, [t])

  useEffect(() => {
    if (active !== 'users') return
    void loadAccounts()
    void apiClient.getRegistrationStatus().then((result) => {
      if (!result.error && typeof result.data === 'boolean') setAllowRegistration(result.data)
    })
  }, [active, loadAccounts])

  const toggleRegistration = async (enabled: boolean) => {
    setRegistrationSaving(true)
    setAccountsError(null)
    setAllowRegistration(enabled)
    const result = await apiClient.setRegistrationStatus(enabled)
    setRegistrationSaving(false)
    if (result.error) {
      setAllowRegistration(!enabled)
      setAccountsError(result.error)
      return
    }
    setAccountsNotice(t(enabled ? 'settings.registrationOpened' : 'settings.registrationClosed'))
  }

  const submitCreateUser = async () => {
    const errors: Record<string, string> = {}
    if (!createForm.username.trim()) errors.username = t('common.required', { field: t('settings.username') })
    if (createForm.password.length < 6) errors.password = t('settings.passwordTooShort')
    setCreateErrors(errors)
    if (Object.keys(errors).length > 0) return

    setCreating(true)
    setCreateError(null)
    const result = await apiClient.createUser(createForm.username.trim(), createForm.password, 'user')
    setCreating(false)
    if (result.error) {
      setCreateError(result.error)
      return
    }
    setCreateOpen(false)
    setAccountsNotice(t('settings.userCreated', { name: createForm.username.trim() }))
    await loadAccounts()
  }

  const toggleAccountStatus = async (account: Account) => {
    const next = account.status === 'active' ? 'disabled' : 'active'
    setStatusPendingId(account.id)
    setAccountsError(null)
    const result = await apiClient.updateUserStatus(account.id, next)
    setStatusPendingId(null)
    if (result.error) {
      setAccountsError(result.error)
      return
    }
    setAccountsNotice(t(next === 'active' ? 'settings.userEnabled' : 'settings.userDisabledNotice', { name: account.username }))
    await loadAccounts()
  }

  const confirmDeleteUser = async () => {
    if (!deletingUser) return
    setDeletePending(true)
    setDeleteError(null)
    const result = await apiClient.deleteUser(deletingUser.id)
    setDeletePending(false)
    if (result.error) {
      setDeleteError(result.error)
      return
    }
    setAccountsNotice(t('settings.userDeleted', { name: deletingUser.username }))
    setDeletingUser(null)
    await loadAccounts()
  }

  useEffect(() => {
    try {
      const stored = localStorage.getItem('app_settings')
      if (stored) setSettings({ ...DEFAULTS, ...JSON.parse(stored) })
    } catch {
      setError(t('state.loadFailed'))
    }
  }, [t])

  const update = <K extends keyof Settings>(category: K, key: keyof Settings[K], value: Settings[K][keyof Settings[K]]) => {
    setSettings((prev) => ({ ...prev, [category]: { ...prev[category], [key]: value } }))
    setSaved(false)
  }

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      localStorage.setItem('app_settings', JSON.stringify(settings))
      if (settings.general.language !== i18n.language) {
        await i18n.changeLanguage(settings.general.language)
      }
      setSaved(true)
      window.setTimeout(() => setSaved(false), 3000)
    } catch {
      setError(t('state.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const sectionMeta = useMemo(() => SECTIONS.find((section) => section.id === active)!, [active])

  return (
    <>
      <PageHeader
        title={t('settings.title')}
        description={t('settings.subtitle')}
        actions={
          <>
            {saved && (
              <Badge variant="success" dot>
                {t('settings.saved')}
              </Badge>
            )}
            <Button variant="primary" size="sm" onClick={() => void handleSave()} disabled={saving}>
              <Save />
              {saving ? t('common.saving') : t('common.save')}
            </Button>
          </>
        }
      />

      {error && <ErrorState title={t('state.saveFailed')} description={error} onRetry={handleSave} retryLabel={t('common.retry')} />}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[13rem_minmax(0,1fr)]">
        {/* Section navigation — the "左分区导航" for this page. */}
        <nav aria-label={t('settings.title')} className="lg:sticky lg:top-20 lg:self-start">
          <ul className="flex gap-1 overflow-x-auto pb-1 lg:flex-col lg:overflow-visible lg:pb-0">
            {SECTIONS.map((section) => {
              const Icon = section.icon
              const selected = active === section.id
              return (
                <li key={section.id} className="shrink-0 lg:shrink">
                  <button
                    type="button"
                    onClick={() => setActive(section.id)}
                    aria-current={selected ? 'true' : undefined}
                    className={cn(
                      'flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-sm transition-colors',
                      selected
                        ? 'bg-surface-hover font-medium text-text-primary'
                        : 'text-text-secondary hover:bg-surface-hover/60 hover:text-text-primary',
                    )}
                  >
                    <Icon className="size-4 shrink-0" />
                    <span className="truncate">{t(section.labelKey)}</span>
                  </button>
                </li>
              )
            })}
          </ul>
          <p className="mt-3 hidden px-1 text-xs text-text-tertiary lg:block">{t('settings.saveHint')}</p>
        </nav>

        <Card padding="none">
          <CardHeader>
            <div>
              <CardTitle>{t(sectionMeta.labelKey)}</CardTitle>
              <p className="mt-1 text-xs text-text-tertiary">{t(sectionMeta.hintKey)}</p>
            </div>
          </CardHeader>

          <CardBody className="space-y-4">
            {active === 'general' && (
              <>
                <FormField label={t('settings.siteName')} hint={t('settings.siteNameHint')} required>
                  {(props) => (
                    <input
                      {...props}
                      type="text"
                      className={controlClassName}
                      value={settings.general.siteName}
                      onChange={(event) => update('general', 'siteName', event.target.value)}
                    />
                  )}
                </FormField>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <FormField label={t('settings.language')}>
                    {(props) => (
                      <Select
                        value={settings.general.language}
                        onValueChange={(value) => update('general', 'language', value)}
                      >
                        <SelectTrigger {...props}>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="zh-CN">简体中文</SelectItem>
                          <SelectItem value="en-US">English</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  </FormField>

                  <FormField label={t('settings.timezone')}>
                    {(props) => (
                      <Select
                        value={settings.general.timezone}
                        onValueChange={(value) => update('general', 'timezone', value)}
                      >
                        <SelectTrigger {...props}>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Asia/Shanghai">Asia/Shanghai (UTC+8)</SelectItem>
                          <SelectItem value="UTC">UTC</SelectItem>
                          <SelectItem value="America/New_York">America/New_York (UTC-5)</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  </FormField>
                </div>
              </>
            )}

            {active === 'database' && (
              <>
                <FormField label={t('settings.databaseDriver')}>
                  {(props) => (
                    <Select
                      value={settings.database.driver}
                      onValueChange={(value) => update('database', 'driver', value)}
                    >
                      <SelectTrigger {...props}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="sqlite">SQLite</SelectItem>
                        <SelectItem value="postgres">PostgreSQL</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                </FormField>

                {settings.database.driver === 'postgres' && (
                  <>
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-[minmax(0,1fr)_7rem]">
                      <FormField label={t('settings.host')}>
                        {(props) => (
                          <input
                            {...props}
                            type="text"
                            className={controlClassName}
                            value={settings.database.host}
                            onChange={(event) => update('database', 'host', event.target.value)}
                          />
                        )}
                      </FormField>
                      <FormField label={t('settings.port')}>
                        {(props) => (
                          <input
                            {...props}
                            type="number"
                            className={cn(controlClassName, 'tabular')}
                            value={settings.database.port}
                            onChange={(event) => update('database', 'port', Number(event.target.value))}
                          />
                        )}
                      </FormField>
                    </div>

                    <FormField label={t('settings.databaseName')}>
                      {(props) => (
                        <input
                          {...props}
                          type="text"
                          className={controlClassName}
                          value={settings.database.name}
                          onChange={(event) => update('database', 'name', event.target.value)}
                        />
                      )}
                    </FormField>
                  </>
                )}
              </>
            )}

            {active === 'security' && (
              <>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <FormField label={`${t('settings.tokenExpiry')} (${t('common.seconds')})`} hint={t('settings.tokenExpiryHint')}>
                    {(props) => (
                      <input
                        {...props}
                        type="number"
                        className={cn(controlClassName, 'tabular')}
                        value={settings.security.tokenExpiry}
                        onChange={(event) => update('security', 'tokenExpiry', Number(event.target.value))}
                      />
                    )}
                  </FormField>

                  <FormField label={t('settings.maxLoginAttempts')} hint={t('settings.maxLoginAttemptsHint')}>
                    {(props) => (
                      <input
                        {...props}
                        type="number"
                        className={cn(controlClassName, 'tabular')}
                        value={settings.security.maxLoginAttempts}
                        onChange={(event) => update('security', 'maxLoginAttempts', Number(event.target.value))}
                      />
                    )}
                  </FormField>
                </div>

                <SwitchField
                  label={t('settings.requireEmailVerification')}
                  checked={settings.security.requireEmailVerification}
                  onChange={(checked) => update('security', 'requireEmailVerification', checked)}
                />
              </>
            )}

            {active === 'users' && (
              <>
                <SwitchField
                  label={t('settings.allowRegistration')}
                  hint={t('settings.allowRegistrationHint')}
                  checked={allowRegistration}
                  onChange={(checked) => {
                    if (!registrationSaving) void toggleRegistration(checked)
                  }}
                />

                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h3 className="text-sm font-medium text-text-primary">{t('settings.addUser')}</h3>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setCreateForm({ username: '', password: '' })
                      setCreateErrors({})
                      setCreateError(null)
                      setCreateOpen(true)
                    }}
                  >
                    <UserPlus />
                    {t('settings.addUser')}
                  </Button>
                </div>

                {accountsError && (
                  <p role="alert" className="rounded-md border border-danger-border bg-danger-bg px-2.5 py-2 text-xs text-danger-fg">
                    {accountsError}
                  </p>
                )}
                {accountsNotice && (
                  <Badge variant="success" role="status" aria-live="polite">
                    {accountsNotice}
                  </Badge>
                )}

                <TableShell
                  busy={accountsLoading}
                  empty={
                    accountsLoading || accounts.length > 0 ? undefined : (
                      <p className="px-2 py-6 text-center text-sm text-text-tertiary">{t('settings.noUsers')}</p>
                    )
                  }
                >
                  <TableHead>
                    <tr>
                      <TableHeaderCell className="min-w-[10rem]">{t('settings.username')}</TableHeaderCell>
                      <TableHeaderCell className="min-w-[7rem]">{t('common.status')}</TableHeaderCell>
                      <TableHeaderCell className="min-w-[9rem] text-right">{t('common.actions')}</TableHeaderCell>
                    </tr>
                  </TableHead>
                  <TableBody>
                    {accounts.map((account) => (
                      <TableRow key={account.id}>
                        <TableCell className="min-w-[10rem] font-medium text-text-primary">{account.username}</TableCell>
                        <TableCell className="min-w-[7rem]">
                          <Badge variant={account.status === 'active' ? 'success' : 'neutral'} dot>
                            {account.status === 'active' ? t('settings.userActive') : t('settings.userDisabled')}
                          </Badge>
                        </TableCell>
                        <TableCell className="min-w-[9rem] text-right">
                          <div className="inline-flex items-center gap-0.5">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => void toggleAccountStatus(account)}
                              disabled={statusPendingId === account.id}
                            >
                              {account.status === 'active' ? t('settings.disableUser') : t('settings.enableUser')}
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              className="text-danger-fg hover:bg-danger-bg"
                              onClick={() => {
                                setDeleteError(null)
                                setDeletingUser(account)
                              }}
                              aria-label={t('common.delete')}
                              title={t('common.delete')}
                            >
                              <Trash2 />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </TableShell>
              </>
            )}

            {active === 'notifications' && (
              <>
                <SwitchField
                  label={t('settings.emailNotifications')}
                  checked={settings.notifications.emailNotifications}
                  onChange={(checked) => update('notifications', 'emailNotifications', checked)}
                />

                <FormField label={t('settings.webhookUrl')} hint={t('settings.webhookUrlHint')}>
                  {(props) => (
                    <input
                      {...props}
                      type="url"
                      placeholder="https://example.com/webhook"
                      className={controlClassName}
                      value={settings.notifications.webhookUrl}
                      onChange={(event) => update('notifications', 'webhookUrl', event.target.value)}
                    />
                  )}
                </FormField>

                <SwitchField
                  label={t('settings.alertOnError')}
                  checked={settings.notifications.alertOnError}
                  onChange={(checked) => update('notifications', 'alertOnError', checked)}
                />
              </>
            )}
          </CardBody>
        </Card>
      </div>

      <Dialog open={createOpen} onOpenChange={(open) => !creating && setCreateOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('settings.addUser')}</DialogTitle>
            <DialogDescription>{t('settings.addUserHint')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <FormField label={t('settings.username')} required error={createErrors.username}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={createForm.username}
                  onChange={(event) => setCreateForm({ ...createForm, username: event.target.value })}
                  autoFocus
                />
              )}
            </FormField>

            <FormField label={t('auth.password')} hint={t('settings.passwordHint')} required error={createErrors.password}>
              {(props) => (
                <input
                  {...props}
                  type="password"
                  autoComplete="new-password"
                  className={controlClassName}
                  value={createForm.password}
                  onChange={(event) => setCreateForm({ ...createForm, password: event.target.value })}
                />
              )}
            </FormField>

            {createError && (
              <p role="alert" className="rounded-md border border-danger-border bg-danger-bg px-2.5 py-2 text-xs text-danger-fg">
                {createError}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setCreateOpen(false)} disabled={creating}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" size="sm" onClick={() => void submitCreateUser()} disabled={creating}>
              {creating ? t('common.saving') : t('common.save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(deletingUser)}
        onOpenChange={(open) => !open && setDeletingUser(null)}
        title={t('settings.deleteUser')}
        description={
          <>
            {deletingUser && <span className="font-medium text-text-secondary">{deletingUser.username}</span>}
            {deletingUser && ' — '}
            {t('settings.deleteUserHint')}
          </>
        }
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        pendingLabel={t('common.deleting')}
        pending={deletePending}
        error={deleteError}
        onConfirm={() => void confirmDeleteUser()}
      />
    </>
  )
}
