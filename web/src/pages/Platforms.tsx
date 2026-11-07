import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe, Pencil, Plus, Trash2 } from 'lucide-react'
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

interface Platform {
  id: string
  name: string
  code: string
  status: string
  description: string
  created_at: string
}

interface Notice {
  variant: 'success' | 'danger'
  text: string
}

const emptyForm = { name: '', code: '', description: '', status: 'active' }

export default function Platforms() {
  const { t } = useTranslation()
  const [platforms, setPlatforms] = useState<Platform[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Platform | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [deleting, setDeleting] = useState<Platform | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deletePending, setDeletePending] = useState(false)

  const loadPlatforms = useCallback(async () => {
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
  }, [])

  useEffect(() => {
    void loadPlatforms()
  }, [loadPlatforms])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const openEdit = (platform: Platform) => {
    setEditing(platform)
    setForm({
      name: platform.name,
      code: platform.code,
      description: platform.description || '',
      status: platform.status || 'active',
    })
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const submit = async () => {
    const errors: Record<string, string> = {}
    if (!form.name.trim()) errors.name = t('common.required', { field: t('common.name') })
    if (!editing && !form.code.trim()) errors.code = t('common.required', { field: t('platforms.code') })
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) return

    setSaving(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updatePlatform(editing.id, {
          name: form.name.trim(),
          description: form.description.trim(),
          status: form.status,
        })
      : await apiClient.createPlatform({
          name: form.name.trim(),
          code: form.code.trim(),
          description: form.description.trim(),
        })
    setSaving(false)

    if (result.error) {
      setFormError(result.error)
      return
    }
    setDialogOpen(false)
    setNotice({ variant: 'success', text: t(editing ? 'notice.updated' : 'notice.created', { name: form.name.trim() }) })
    await loadPlatforms()
  }

  const confirmDelete = async () => {
    if (!deleting) return
    setDeletePending(true)
    setDeleteError(null)
    const result = await apiClient.deletePlatform(deleting.id)
    setDeletePending(false)
    if (result.error) {
      setDeleteError(result.error)
      return
    }
    setNotice({ variant: 'success', text: t('notice.deleted', { name: deleting.name }) })
    setDeleting(null)
    await loadPlatforms()
  }

  const columns = [
    { key: 'name', label: t('common.name'), className: 'min-w-[11rem]' },
    { key: 'code', label: t('platforms.code'), className: 'min-w-[9rem]' },
    { key: 'description', label: t('common.description'), className: 'min-w-[14rem]' },
    { key: 'status', label: t('common.status'), className: 'min-w-[6rem]' },
    { key: 'actions', label: t('common.actions'), className: 'min-w-[6rem] text-right' },
  ]

  return (
    <>
      <PageHeader
        title={t('platforms.title')}
        description={t('platforms.subtitle')}
        actions={
          <Button variant="primary" size="sm" onClick={openCreate}>
            <Plus />
            {t('platforms.create')}
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
          onRetry={loadPlatforms}
          retryLabel={t('common.retry')}
        />
      ) : (
        <TableShell
          busy={loading}
          toolbar={
            <span className="tabular text-xs text-text-tertiary">
              {loading ? t('state.loadingData') : t('platforms.count', { count: platforms.length })}
            </span>
          }
          empty={
            loading ? (
              <SkeletonRows rows={5} columns={4} />
            ) : platforms.length === 0 ? (
              <EmptyState
                icon={Globe}
                title={t('platforms.empty')}
                description={t('platforms.emptyHint')}
                action={
                  <Button variant="primary" size="sm" onClick={openCreate}>
                    <Plus />
                    {t('platforms.create')}
                  </Button>
                }
              />
            ) : undefined
          }
        >
          <TableHead>
            <tr>
              {columns.map((column) => (
                <TableHeaderCell key={column.key} className={column.className}>
                  {column.label}
                </TableHeaderCell>
              ))}
            </tr>
          </TableHead>
          <TableBody>
            {platforms.map((platform) => (
              <TableRow key={platform.id}>
                <TableCell className="min-w-[11rem] font-medium text-text-primary">{platform.name}</TableCell>
                <TableCell className="min-w-[9rem]">
                  <code className="rounded-xs bg-surface-inset px-1.5 py-0.5 font-mono text-xs text-text-secondary">
                    {platform.code}
                  </code>
                </TableCell>
                <TableCell className="min-w-[14rem]">
                  <span className="truncate-cell text-text-tertiary" title={platform.description}>
                    {platform.description || '—'}
                  </span>
                </TableCell>
                <TableCell className="min-w-[6rem]">
                  <Badge variant={platform.status === 'active' ? 'success' : 'neutral'} dot>
                    {platform.status === 'active' ? t('platforms.active') : t('platforms.inactive')}
                  </Badge>
                </TableCell>
                <TableCell className="min-w-[6rem] text-right">
                  <div className="inline-flex items-center gap-0.5">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => openEdit(platform)}
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
                        setDeleting(platform)
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
            <DialogTitle>{editing ? t('platforms.editTitle') : t('platforms.createTitle')}</DialogTitle>
            <DialogDescription>{t('platforms.createHint')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <FormField label={t('common.name')} hint={t('platforms.nameHint')} required error={fieldErrors.name}>
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

            <FormField label={t('platforms.code')} hint={t('platforms.codeHint')} required={!editing} error={fieldErrors.code}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.code}
                  disabled={Boolean(editing)}
                  onChange={(event) => setForm({ ...form, code: event.target.value })}
                />
              )}
            </FormField>

            <FormField label={t('common.description')} hint={t('platforms.descHint')}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.description}
                  onChange={(event) => setForm({ ...form, description: event.target.value })}
                />
              )}
            </FormField>

            {editing && (
              <FormField label={t('common.status')}>
                {(props) => (
                  <div {...props}>
                    <Select value={form.status} onValueChange={(value) => setForm({ ...form, status: value })}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="active">{t('platforms.active')}</SelectItem>
                        <SelectItem value="inactive">{t('platforms.inactive')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </FormField>
            )}

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
        title={t('platforms.deleteTitle')}
        description={
          <>
            {deleting && <span className="font-medium text-text-secondary">{deleting.name}</span>}
            {deleting && ' — '}
            {t('platforms.deleteHint')}
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
