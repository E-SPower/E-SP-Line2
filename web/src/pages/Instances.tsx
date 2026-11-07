import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Play, Pencil, Plus, Server, Square, Trash2 } from 'lucide-react'
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

interface Instance {
  id: string
  name: string
  adapter_id: string
  platform_id: string
  status: string
  config: string
  created_at: string
}

interface AdapterOption {
  id: string
  name: string
  platform_id: string
}

interface Notice {
  variant: 'success' | 'danger'
  text: string
}

const emptyForm = { name: '', adapter_id: '', platform_id: '', config: '' }

// The stored config is a JSON blob; a malformed or absent one must not break
// the page, so it degrades to an empty object.
function parseConfig(config: string | null | undefined): Record<string, any> {
  if (!config) return {}
  try {
    const parsed = JSON.parse(config)
    return typeof parsed === 'object' && parsed !== null ? parsed : {}
  } catch {
    return {}
  }
}

// The proxy is stored either as a URL string or as the structured
// {scheme,host,port,username,password} object. The form renders one text input,
// so the object form is folded back into a URL before prefill — otherwise the
// input would show "[object Object]" and round-trip it back into the config.
function proxyForForm(proxy: unknown): string {
  if (typeof proxy === 'string') return proxy
  if (!proxy || typeof proxy !== 'object') return ''
  const o = proxy as Record<string, any>
  if (o.url) return String(o.url)
  const host = String(o.host || '').trim()
  if (!host) return ''
  const scheme = String(o.scheme || 'http').trim() || 'http'
  const port = o.port ? `:${o.port}` : ''
  const auth = o.username
    ? `${encodeURIComponent(String(o.username))}${o.password ? `:${encodeURIComponent(String(o.password))}` : ''}@`
    : ''
  return `${scheme}://${auth}${host}${port}`
}

// Form values to prefill when editing: the stored config, with the proxy folded
// into its single-input form.
function configForForm(config: string | null | undefined): Record<string, any> {
  const cfg = parseConfig(config)
  if ('proxy' in cfg) cfg.proxy = proxyForForm(cfg.proxy)
  return cfg
}

export default function Instances() {
  const { t } = useTranslation()
  const [instances, setInstances] = useState<Instance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<Notice | null>(null)
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [adapters, setAdapters] = useState<AdapterOption[]>([])

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Instance | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [deleting, setDeleting] = useState<Instance | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deletePending, setDeletePending] = useState(false)

  const loadInstances = useCallback(async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getInstances()
    if (result.error) {
      setError(result.error)
      setInstances([])
    } else {
      setInstances(result.data || [])
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    void loadInstances()
  }, [loadInstances])

  // Creating an instance starts from a real adapter, so its platform follows.
  useEffect(() => {
    apiClient.getAdapters(100, 0).then((result) => {
      if (!result.error) setAdapters(result.data || [])
    })
  }, [])

  const pickAdapter = (adapterId: string) => {
    const adapter = adapters.find((item) => item.id === adapterId)
    setForm({ ...form, adapter_id: adapterId, platform_id: adapter?.platform_id || form.platform_id })
  }

  const toggle = async (instance: Instance) => {
    setPendingId(instance.id)
    setNotice(null)
    try {
      const running = instance.status === 'running'
      const result = await apiClient.updateInstance(instance.id, { status: running ? 'stopped' : 'running' })
      if (result.error) {
        setNotice({ variant: 'danger', text: t('notice.failed', { error: result.error }) })
      } else {
        setNotice({
          variant: 'success',
          text: t(running ? 'notice.stopped' : 'notice.started', { name: instance.name }),
        })
        await loadInstances()
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

  const openEdit = (instance: Instance) => {
    setEditing(instance)
    setForm({
      name: instance.name,
      adapter_id: instance.adapter_id,
      platform_id: instance.platform_id || '',
      config: JSON.stringify(configForForm(instance.config), null, 2),
    })
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const submit = async () => {
    const errors: Record<string, string> = {}
    if (!form.name.trim()) errors.name = t('common.required', { field: t('common.name') })
    if (!editing && !form.adapter_id.trim()) errors.adapter_id = t('instances.adapterRequired')
    if (!editing && !form.platform_id.trim()) errors.platform_id = t('instances.platformRequired')
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) return

    setSaving(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updateInstance(editing.id, { name: form.name.trim(), config: form.config.trim() })
      : await apiClient.createInstance({
          name: form.name.trim(),
          adapter_id: form.adapter_id.trim(),
          platform_id: form.platform_id.trim(),
          config: form.config.trim(),
        })
    setSaving(false)

    if (result.error) {
      setFormError(result.error)
      return
    }
    setDialogOpen(false)
    setNotice({ variant: 'success', text: t(editing ? 'notice.updated' : 'notice.created', { name: form.name.trim() }) })
    await loadInstances()
  }

  const confirmDelete = async () => {
    if (!deleting) return
    setDeletePending(true)
    setDeleteError(null)
    const result = await apiClient.deleteInstance(deleting.id)
    setDeletePending(false)
    if (result.error) {
      setDeleteError(result.error)
      return
    }
    setNotice({ variant: 'success', text: t('notice.deleted', { name: deleting.name }) })
    setDeleting(null)
    await loadInstances()
  }

  return (
    <>
      <PageHeader
        title={t('instances.title')}
        description={t('instances.subtitle')}
        actions={
          <Button variant="primary" size="sm" onClick={openCreate}>
            <Plus />
            {t('instances.create')}
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
          onRetry={loadInstances}
          retryLabel={t('common.retry')}
        />
      ) : (
        <TableShell
          busy={loading}
          toolbar={
            <span className="tabular text-xs text-text-tertiary">
              {loading ? t('state.loadingData') : t('instances.count', { count: instances.length })}
            </span>
          }
          empty={
            loading ? (
              <SkeletonRows rows={5} columns={4} />
            ) : instances.length === 0 ? (
              <EmptyState
                icon={Server}
                title={t('instances.empty')}
                description={t('instances.emptyHint')}
                action={
                  <Button variant="primary" size="sm" onClick={openCreate}>
                    <Plus />
                    {t('instances.create')}
                  </Button>
                }
              />
            ) : undefined
          }
        >
          <TableHead>
            <tr>
              <TableHeaderCell className="min-w-[13rem]">{t('common.name')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[11rem]">{t('instances.adapter')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[10rem]">{t('messages.platform')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[7rem]">{t('common.status')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[9rem] text-right">{t('common.actions')}</TableHeaderCell>
            </tr>
          </TableHead>
          <TableBody>
            {instances.map((instance) => {
              const running = instance.status === 'running'
              return (
                <TableRow key={instance.id}>
                  <TableCell className="min-w-[13rem] font-medium text-text-primary">{instance.name}</TableCell>
                  <TableCell className="min-w-[11rem]">
                    <span className="truncate-cell font-mono text-xs" title={instance.adapter_id}>
                      {instance.adapter_id}
                    </span>
                  </TableCell>
                  <TableCell className="min-w-[10rem] font-mono text-xs text-text-tertiary">
                    {instance.platform_id || '—'}
                  </TableCell>
                  <TableCell className="min-w-[7rem]">
                    <Badge variant={running ? 'success' : 'neutral'} dot>
                      {running ? t('state.status.running') : t('state.status.stopped')}
                    </Badge>
                  </TableCell>
                  <TableCell className="min-w-[9rem] text-right">
                    <div className="inline-flex items-center gap-0.5">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => void toggle(instance)}
                        disabled={pendingId === instance.id}
                        aria-label={running ? t('adapters.stop') : t('adapters.start')}
                        title={running ? t('adapters.stop') : t('adapters.start')}
                      >
                        {running ? <Square /> : <Play />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => openEdit(instance)}
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
                          setDeleting(instance)
                        }}
                        aria-label={t('common.delete')}
                        title={t('common.delete')}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </TableShell>
      )}

      <Dialog open={dialogOpen} onOpenChange={(open) => !saving && setDialogOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? t('instances.editTitle') : t('instances.createTitle')}</DialogTitle>
            <DialogDescription>{t('instances.createHint')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <FormField label={t('common.name')} hint={t('instances.nameHint')} required error={fieldErrors.name}>
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
              <FormField label={t('instances.adapter')} hint={t('instances.adapterHint')} required error={fieldErrors.adapter_id}>
                {(props) => (
                  <div {...props}>
                    <Select value={form.adapter_id} onValueChange={pickAdapter}>
                      <SelectTrigger aria-invalid={fieldErrors.adapter_id ? true : undefined}>
                        <SelectValue placeholder={t('instances.adapter')} />
                      </SelectTrigger>
                      <SelectContent>
                        {adapters.map((adapter) => (
                          <SelectItem key={adapter.id} value={adapter.id}>
                            {adapter.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </FormField>
            )}

            <FormField label={t('messages.platform')} hint={t('instances.platformHint')} required error={fieldErrors.platform_id}>
              {(props) => (
                <input
                  {...props}
                  className={controlClassName}
                  value={form.platform_id}
                  disabled={true}
                  onChange={(event) => setForm({ ...form, platform_id: event.target.value })}
                />
              )}
            </FormField>

            <FormField label={t('instances.config')} hint={t('instances.configHint')}>
              {(props) => (
                <textarea
                  {...props}
                  className={`${controlClassName} h-20 py-2 font-mono text-xs`}
                  value={form.config}
                  onChange={(event) => setForm({ ...form, config: event.target.value })}
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
        title={t('instances.deleteTitle')}
        description={
          <>
            {deleting && <span className="font-medium text-text-secondary">{deleting.name}</span>}
            {deleting && ' — '}
            {t('instances.deleteHint')}
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
