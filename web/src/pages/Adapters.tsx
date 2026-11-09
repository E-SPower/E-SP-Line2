import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { Boxes, Pencil, Play, Plus, Square, Trash2 } from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { Card, CardBody, CardFooter, CardHeader, CardTitle } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { EmptyState } from '../components/ui/empty-state'
import { ErrorState } from '../components/ui/error-state'
import { Skeleton } from '../components/ui/skeleton'
import { ConfirmDialog } from '../components/ui/confirm-dialog'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog'
import { FormField, controlClassName } from '../components/ui/form-field'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'

interface Adapter {
  id: string
  name: string
  platform_id: string
  version: string
  runtime_type: string
  status: string
  created_at: string
}

interface Platform {
  id: string
  name: string
  code: string
}

interface Notice {
  variant: 'success' | 'danger'
  text: string
}

const emptyForm = { name: '', platform_id: '', version: '', runtime_type: '' }

function AdapterCardSkeleton() {
  return (
    <Card padding="none">
      <CardHeader>
        <div className="w-full space-y-2">
          <Skeleton shape="line" className="h-4 w-2/3" />
          <Skeleton shape="line" className="h-3 w-1/3" />
        </div>
      </CardHeader>
      <CardBody className="space-y-2">
        <Skeleton shape="line" className="h-3 w-full" />
        <Skeleton shape="line" className="h-3 w-4/5" />
      </CardBody>
    </Card>
  )
}

export default function Adapters() {
  const { t } = useTranslation()
  const [adapters, setAdapters] = useState<Adapter[]>([])
  const [platforms, setPlatforms] = useState<Platform[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [notice, setNotice] = useState<Notice | null>(null)

  // create / edit dialog
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Adapter | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  // delete confirm
  const [deleting, setDeleting] = useState<Adapter | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deletePending, setDeletePending] = useState(false)

  const loadAdapters = useCallback(async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getAdapters()
    if (result.error) {
      setError(result.error)
      setAdapters([])
    } else {
      setAdapters(result.data || [])
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    void loadAdapters()
  }, [loadAdapters])

  useEffect(() => {
    apiClient.getPlatforms(100, 0).then((result) => {
      if (!result.error) setPlatforms(result.data || [])
    })
  }, [])

  // The dashboard's "create" shortcut lands here with ?create=1.
  const [searchParams, setSearchParams] = useSearchParams()
  useEffect(() => {
    if (searchParams.get('create') !== '1') return
    setEditing(null)
    setForm(emptyForm)
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
    setSearchParams({}, { replace: true })
  }, [searchParams, setSearchParams])

  const toggle = async (adapter: Adapter) => {
    setPendingId(adapter.id)
    setNotice(null)
    try {
      const running = adapter.status === 'active'
      const result = running ? await apiClient.stopAdapter(adapter.id) : await apiClient.startAdapter(adapter.id)
      if (result.error) {
        setNotice({ variant: 'danger', text: t('notice.failed', { error: result.error }) })
      } else {
        setNotice({ variant: 'success', text: t(running ? 'notice.stopped' : 'notice.started', { name: adapter.name }) })
        await loadAdapters()
      }
    } finally {
      setPendingId(null)
    }
  }

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const openEdit = (adapter: Adapter) => {
    setEditing(adapter)
    setForm({ name: adapter.name, platform_id: adapter.platform_id, version: adapter.version || '', runtime_type: adapter.runtime_type || '' })
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const submit = async () => {
    const errors: Record<string, string> = {}
    if (!form.name.trim()) errors.name = t('common.required', { field: t('common.name') })
    if (!editing && !form.platform_id) errors.platform_id = t('adapters.platformRequired')
    if (!editing && !form.version.trim()) errors.version = t('common.required', { field: t('adapters.version') })
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) return

    setSaving(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updateAdapter(editing.id, { name: form.name.trim(), version: form.version.trim(), runtime_type: form.runtime_type.trim() })
      : await apiClient.createAdapter({
          name: form.name.trim(),
          platform_id: form.platform_id,
          version: form.version.trim(),
          runtime_type: form.runtime_type.trim(),
        })
    setSaving(false)

    if (result.error) {
      setFormError(result.error)
      return
    }
    setDialogOpen(false)
    setNotice({
      variant: 'success',
      text: t(editing ? 'notice.updated' : 'notice.created', { name: form.name.trim() }),
    })
    await loadAdapters()
  }

  const confirmDelete = async () => {
    if (!deleting) return
    setDeletePending(true)
    setDeleteError(null)
    const result = await apiClient.deleteAdapter(deleting.id)
    setDeletePending(false)
    if (result.error) {
      setDeleteError(result.error)
      return
    }
    setNotice({ variant: 'success', text: t('notice.deleted', { name: deleting.name }) })
    setDeleting(null)
    await loadAdapters()
  }

  const noticeBadge = notice && (
    <Badge variant={notice.variant} role="status" aria-live="polite">
      {notice.text}
    </Badge>
  )

  return (
    <>
      <PageHeader
        title={t('adapters.title')}
        description={t('adapters.subtitle')}
        actions={
          <Button variant="primary" size="sm" onClick={openCreate}>
            <Plus />
            {t('adapters.create')}
          </Button>
        }
      />

      {noticeBadge}

      {error ? (
        <ErrorState
          title={t('state.loadFailed')}
          description={error || t('state.loadFailedHint')}
          onRetry={loadAdapters}
          retryLabel={t('common.retry')}
        />
      ) : loading ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <AdapterCardSkeleton key={index} />
          ))}
        </div>
      ) : adapters.length === 0 ? (
        <Card padding="none">
          <EmptyState
            size="page"
            icon={Boxes}
            title={t('adapters.empty')}
            description={t('adapters.emptyHint')}
            action={
              <Button variant="primary" size="sm" onClick={openCreate}>
                <Plus />
                {t('adapters.create')}
              </Button>
            }
          />
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {adapters.map((adapter) => {
            const running = adapter.status === 'active'
            return (
              <Card key={adapter.id} padding="none" className="flex flex-col">
                <CardHeader className="flex-row items-start justify-between gap-3">
                  <div className="min-w-0">
                    <CardTitle className="truncate text-base">{adapter.name}</CardTitle>
                    <p className="mt-0.5 truncate font-mono text-xs text-text-tertiary">{adapter.platform_id}</p>
                  </div>
                  <Badge variant={running ? 'success' : 'neutral'} dot className="shrink-0">
                    {running ? t('adapters.running') : t('adapters.stopped')}
                  </Badge>
                </CardHeader>

                <CardBody className="flex-1 space-y-1.5 text-sm">
                  <div className="flex items-baseline justify-between gap-3">
                    <span className="text-text-tertiary">{t('adapters.version')}</span>
                    <span className="tabular font-mono text-xs text-text-secondary">{adapter.version || '—'}</span>
                  </div>
                  <div className="flex items-baseline justify-between gap-3">
                    <span className="text-text-tertiary">{t('adapters.runtime')}</span>
                    <span className="truncate font-mono text-xs text-text-secondary">
                      {adapter.runtime_type || t('adapters.runtimeMissing')}
                    </span>
                  </div>
                </CardBody>

                <CardFooter className="justify-between">
                  <Button
                    variant={running ? 'secondary' : 'outline'}
                    size="sm"
                    onClick={() => void toggle(adapter)}
                    disabled={pendingId === adapter.id}
                  >
                    {running ? <Square /> : <Play />}
                    {running ? t('adapters.stop') : t('adapters.start')}
                  </Button>
                  <div className="flex items-center gap-0.5">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => openEdit(adapter)}
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
                        setDeleting(adapter)
                      }}
                      aria-label={t('common.delete')}
                      title={t('common.delete')}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                </CardFooter>
              </Card>
            )
          })}
        </div>
      )}

      {!loading && !error && adapters.length > 0 && (
        <p className="tabular text-xs text-text-tertiary">{t('adapters.count', { count: adapters.length })}</p>
      )}

      <Dialog open={dialogOpen} onOpenChange={(open) => !saving && setDialogOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? t('adapters.editTitle') : t('adapters.createTitle')}</DialogTitle>
            <DialogDescription>{t('adapters.createHint')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <FormField label={t('common.name')} hint={t('adapters.nameHint')} required error={fieldErrors.name}>
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

            {!editing && (
              <FormField label={t('adapters.platform')} required error={fieldErrors.platform_id}>
                {(props) => (
                  <div {...props}>
                    <Select value={form.platform_id} onValueChange={(value) => setForm({ ...form, platform_id: value })}>
                      <SelectTrigger aria-invalid={fieldErrors.platform_id ? true : undefined}>
                        <SelectValue placeholder={t('adapters.platform')} />
                      </SelectTrigger>
                      <SelectContent>
                        {platforms.map((platform) => (
                          <SelectItem key={platform.id} value={platform.id}>
                            {platform.name} · {platform.code}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </FormField>
            )}

            <FormField label={t('adapters.version')} hint={t('adapters.versionHint')} required error={fieldErrors.version}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.version}
                  onChange={(event) => setForm({ ...form, version: event.target.value })}
                />
              )}
            </FormField>

            <FormField label={t('adapters.runtime')} hint={t('adapters.runtimeHint')}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.runtime_type}
                  onChange={(event) => setForm({ ...form, runtime_type: event.target.value })}
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
        title={t('adapters.deleteTitle')}
        description={
          <>
            {deleting && <span className="font-medium text-text-secondary">{deleting.name}</span>}
            {deleting && ' — '}
            {t('adapters.deleteHint')}
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
