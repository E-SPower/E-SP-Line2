import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { GitBranch, Pencil, Plus, Trash2 } from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TableShell } from '../components/ui/table-shell'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { EmptyState } from '../components/ui/empty-state'
import { ErrorState } from '../components/ui/error-state'
import { SkeletonRows } from '../components/ui/skeleton'
import { ConfirmDialog } from '../components/ui/confirm-dialog'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog'
import { FormField, controlClassName } from '../components/ui/form-field'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'

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

interface Notice {
  variant: 'success' | 'danger'
  text: string
}

const TARGET_TYPES = ['app', 'webhook']

const emptyForm = { name: '', target_type: 'app', target_id: '', priority: '10', platform_id: '' }

export default function Routes() {
  const { t } = useTranslation()
  const [routes, setRoutes] = useState<Route[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Route | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [deleting, setDeleting] = useState<Route | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deletePending, setDeletePending] = useState(false)

  const loadRoutes = useCallback(async () => {
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
  }, [])

  useEffect(() => {
    void loadRoutes()
  }, [loadRoutes])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const openEdit = (route: Route) => {
    setEditing(route)
    setForm({
      name: route.name,
      target_type: route.target_type || 'app',
      target_id: route.target_id || '',
      priority: String(route.priority ?? 10),
      platform_id: route.platform_id || '',
    })
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const submit = async () => {
    const errors: Record<string, string> = {}
    if (!form.name.trim()) errors.name = t('common.required', { field: t('common.name') })
    if (!form.target_type) errors.target_type = t('routes.targetTypeRequired')
    if (!form.target_id.trim()) errors.target_id = t('routes.targetIdRequired')
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) return

    const priority = Number.parseInt(form.priority, 10)
    setSaving(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updateRoute(editing.id, {
          name: form.name.trim(),
          target_type: form.target_type,
          target_id: form.target_id.trim(),
          priority: Number.isNaN(priority) ? 0 : priority,
          platform_id: form.platform_id.trim(),
          enabled: editing.enabled,
        })
      : await apiClient.createRoute({
          name: form.name.trim(),
          target_type: form.target_type,
          target_id: form.target_id.trim(),
          priority: Number.isNaN(priority) ? 0 : priority,
          platform_id: form.platform_id.trim(),
          enabled: true,
        })
    setSaving(false)

    if (result.error) {
      setFormError(result.error)
      return
    }
    setDialogOpen(false)
    setNotice({ variant: 'success', text: t(editing ? 'notice.updated' : 'notice.created', { name: form.name.trim() }) })
    await loadRoutes()
  }

  const confirmDelete = async () => {
    if (!deleting) return
    setDeletePending(true)
    setDeleteError(null)
    const result = await apiClient.deleteRoute(deleting.id)
    setDeletePending(false)
    if (result.error) {
      setDeleteError(result.error)
      return
    }
    setNotice({ variant: 'success', text: t('notice.deleted', { name: deleting.name }) })
    setDeleting(null)
    await loadRoutes()
  }

  return (
    <>
      <PageHeader
        title={t('routes.title')}
        description={t('routes.subtitle')}
        actions={
          <Button variant="primary" size="sm" onClick={openCreate}>
            <Plus />
            {t('routes.create')}
          </Button>
        }
      />

      {notice && (
        <Badge variant={notice.variant} role="status" aria-live="polite">
          {notice.text}
        </Badge>
      )}

      {error ? (
        <ErrorState
          title={t('state.loadFailed')}
          description={error || t('state.loadFailedHint')}
          onRetry={loadRoutes}
          retryLabel={t('common.retry')}
        />
      ) : (
        <TableShell
          busy={loading}
          toolbar={
            <span className="tabular text-xs text-text-tertiary">
              {loading ? t('state.loadingData') : t('routes.count', { count: routes.length })}
            </span>
          }
          empty={
            loading ? (
              <SkeletonRows rows={4} columns={5} />
            ) : routes.length === 0 ? (
              <EmptyState
                icon={GitBranch}
                title={t('routes.empty')}
                description={t('routes.emptyHint')}
                action={
                  <Button variant="primary" size="sm" onClick={openCreate}>
                    <Plus />
                    {t('routes.create')}
                  </Button>
                }
              />
            ) : undefined
          }
        >
          <TableHead>
            <tr>
              <TableHeaderCell className="min-w-[12rem]">{t('common.name')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[10rem]">{t('messages.platform')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[5rem]">{t('routes.priority')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[15rem]">{t('routes.target')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[6rem]">{t('routes.enabled')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[6rem] text-right">{t('common.actions')}</TableHeaderCell>
            </tr>
          </TableHead>
          <TableBody>
            {routes.map((route) => (
              <TableRow key={route.id}>
                <TableCell className="min-w-[12rem] font-medium text-text-primary">{route.name}</TableCell>
                <TableCell className="min-w-[10rem] font-mono text-xs text-text-tertiary">
                  {route.platform_id}
                </TableCell>
                <TableCell className="min-w-[5rem]">
                  <span className="tabular inline-flex min-w-6 items-center justify-center rounded-xs bg-surface-inset px-1.5 py-0.5 text-xs font-semibold text-text-secondary">
                    {route.priority}
                  </span>
                </TableCell>
                <TableCell className="min-w-[15rem]">
                  <span className="truncate-cell text-xs" title={`${route.target_type}: ${route.target_id}`}>
                    <span className="text-text-tertiary">{route.target_type}:</span>{' '}
                    <span className="font-mono text-text-secondary">{route.target_id}</span>
                  </span>
                </TableCell>
                <TableCell className="min-w-[6rem]">
                  <Badge variant={route.enabled ? 'success' : 'neutral'} dot>
                    {route.enabled ? t('state.status.enabled') : t('state.status.disabled')}
                  </Badge>
                </TableCell>
                <TableCell className="min-w-[6rem] text-right">
                  <div className="inline-flex items-center gap-0.5">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => openEdit(route)}
                      aria-label={t('common.edit')}
                      title={t('common.edit')}
                    >
                      <Pencil />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-danger-fg hover:bg-danger-bg"
                      onClick={() => {
                        setDeleteError(null)
                        setDeleting(route)
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
      )}

      <Dialog open={dialogOpen} onOpenChange={(open) => !saving && setDialogOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? t('routes.editTitle') : t('routes.createTitle')}</DialogTitle>
            <DialogDescription>{t('routes.createHint')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <FormField label={t('common.name')} hint={t('routes.nameHint')} required error={fieldErrors.name}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.name}
                  onChange={(event) => setForm({ ...form, name: event.target.value })}
                  autoFocus
                />
              )}
            </FormField>

            <FormField label={t('routes.targetType')} hint={t('routes.targetTypeHint')} required error={fieldErrors.target_type}>
              {(props) => (
                <div {...props}>
                  <Select value={form.target_type} onValueChange={(value) => setForm({ ...form, target_type: value })}>
                    <SelectTrigger aria-invalid={fieldErrors.target_type ? true : undefined}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TARGET_TYPES.map((type) => (
                        <SelectItem key={type} value={type}>
                          {type}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
            </FormField>

            <FormField label={t('routes.targetId')} hint={t('routes.targetIdHint')} required error={fieldErrors.target_id}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.target_id}
                  onChange={(event) => setForm({ ...form, target_id: event.target.value })}
                />
              )}
            </FormField>

            <FormField label={t('routes.priority')} hint={t('routes.priorityHint')}>
              {(props) => (
                <input
                  {...props}
                  type="number"
                  className={`${controlClassName} tabular`}
                  value={form.priority}
                  onChange={(event) => setForm({ ...form, priority: event.target.value })}
                />
              )}
            </FormField>

            {formError && (
              <p role="alert" className="rounded-md border border-danger-border bg-danger-bg px-2.5 py-2 text-xs text-danger-fg">
                {formError}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setDialogOpen(false)} disabled={saving}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" size="sm" onClick={() => void submit()} disabled={saving}>
              {saving ? t('common.saving') : t('common.save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={t('routes.deleteTitle')}
        description={
          <>
            {deleting && <span className="font-medium text-text-secondary">{deleting.name}</span>}
            {deleting && ' — '}
            {t('routes.deleteHint')}
          </>
        }
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        pendingLabel={t('common.deleting')}
        pending={deletePending}
        error={deleteError}
        onConfirm={() => void confirmDelete()}
      />
    </>
  )
}
