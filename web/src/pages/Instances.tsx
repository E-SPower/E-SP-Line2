import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, Play, Pencil, Plus, ScrollText, Server, Square, Trash2 } from 'lucide-react'
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
import { cn } from '@/lib/cn'
import { InstanceLogDrawer, type LogInstance } from '../components/instance-log-drawer'

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

interface PlatformOption {
  id: string
  code: string
}

/**
 * One `config_fields` entry of a platform spec (GET /platforms/spec/:code).
 *
 * The registry carries these for every platform, and the form is rendered from
 * them: `string` becomes a text input, `integer` a number input, and secret-ish
 * names get a masked input with a reveal toggle. That is what replaced the bare
 * JSON textarea the rewrite left behind.
 */
interface ConfigField {
  name: string
  type: 'string' | 'integer' | string
  required?: boolean
  description?: string
  description_en?: string
  default?: unknown
}

/** True for names whose value should not be readable by default. */
const SECRET_FIELD = /(cookie|token|secret|password|key)/i

interface Notice {
  variant: 'success' | 'danger'
  text: string
}

const emptyForm = { name: '', adapter_id: '', platform_id: '' }

/** Render a stored value for an input; null/undefined become "". */
function fieldValue(value: unknown): string {
  return value === null || value === undefined ? '' : String(value)
}

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

/**
 * Serialize the field map back to the stored JSON. Empty fields are dropped,
 * `integer` fields become numbers, and an all-empty form serializes to "" so
 * the reader can clear a stored config (the API treats a sent "" as a real
 * clear).
 */
function serializeConfig(values: Record<string, string>, fields: ConfigField[]): string {
  const out: Record<string, unknown> = {}
  for (const [name, raw] of Object.entries(values)) {
    if (raw === '') continue
    if (fields.find((field) => field.name === name)?.type === 'integer') {
      const parsed = Number(raw)
      if (!Number.isNaN(parsed)) out[name] = parsed
      continue
    }
    out[name] = raw
  }
  return Object.keys(out).length === 0 ? '' : JSON.stringify(out)
}

export default function Instances() {
  const { t } = useTranslation()
  const [instances, setInstances] = useState<Instance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<Notice | null>(null)
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [adapters, setAdapters] = useState<AdapterOption[]>([])
  // Platform rows, only used to translate an instance's platform_id into the
  // platform_code the catalog files its schemas under.
  const [platforms, setPlatforms] = useState<PlatformOption[]>([])

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Instance | null>(null)
  const [form, setForm] = useState(emptyForm)
  // Config is schema-driven: the picked adapter supplies the field list.
  const [configFields, setConfigFields] = useState<ConfigField[]>([])
  const [configValues, setConfigValues] = useState<Record<string, string>>({})
  const [revealed, setRevealed] = useState<Record<string, boolean>>({})
  // Raw JSON escape hatch for adapters that declare no schema.
  const [configText, setConfigText] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [logsFor, setLogsFor] = useState<LogInstance | null>(null)
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

  useEffect(() => {
    apiClient.getPlatforms(100, 0).then((result) => {
      if (!result.error) setPlatforms(result.data || [])
    })
  }, [])

  // Creating an instance starts from a real adapter, so its platform follows.
  useEffect(() => {
    apiClient.getAdapters(100, 0).then((result) => {
      if (!result.error) setAdapters(result.data || [])
    })
  }, [])

  /**
   * Load a platform's config fields from the spec registry.
   *
   * Keyed by platform code, because the spec id *is* the code while an instance
   * stores the platform row id — hence the lookup through `platforms`.
   */
  const loadConfigFields = useCallback(
    async (platformId: string) => {
      const code = platforms.find((platform) => platform.id === platformId)?.code
      if (!code) {
        setConfigFields([])
        return
      }
      // The spec registry is public (no token needed) but lives behind the
      // same origin as the API, so read it directly rather than adding a client
      // method for one caller.
      try {
        const response = await fetch(`/api/v1/platforms/spec/${encodeURIComponent(code)}`)
        if (!response.ok) {
          setConfigFields([])
          return
        }
        const body = (await response.json()) as { config_fields?: ConfigField[] }
        setConfigFields(body.config_fields || [])
      } catch {
        setConfigFields([])
      }
    },
    [platforms],
  )

  const pickAdapter = (adapterId: string) => {
    const adapter = adapters.find((item) => item.id === adapterId)
    setForm({ ...form, adapter_id: adapterId, platform_id: adapter?.platform_id || form.platform_id })
    // Field names differ per adapter; drop values that no longer apply.
    setConfigValues({})
    setConfigText('')
    setRevealed({})
    if (adapter?.platform_id) void loadConfigFields(adapter.platform_id)
  }

  const resetConfig = () => {
    setConfigFields([])
    setConfigValues({})
    setConfigText('')
    setRevealed({})
  }

  const toggle = async (instance: Instance) => {
    setPendingId(instance.id)
    setNotice(null)
    try {
      const running = instance.status === 'running'
      // Real process control: these spawn / kill the adapter process, they do
      // not merely flip the status column.
      const result = running ? await apiClient.stopInstance(instance.id) : await apiClient.startInstance(instance.id)
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
    resetConfig()
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const openEdit = (instance: Instance) => {
    setEditing(instance)
    setForm({ name: instance.name, adapter_id: instance.adapter_id, platform_id: instance.platform_id || '' })
    resetConfig()
    // Existing config is re-hydrated into the schema's inputs, so editing never
    // silently drops what is already stored.
    const stored = parseConfig(instance.config)
    if ('proxy' in stored) stored.proxy = proxyForForm(stored.proxy)
    setConfigValues(Object.fromEntries(Object.entries(stored).map(([k, v]) => [k, fieldValue(v)])))
    setConfigText(instance.config || '')
    void loadConfigFields(instance.platform_id)
    setFieldErrors({})
    setFormError(null)
    setDialogOpen(true)
  }

  const submit = async () => {
    const errors: Record<string, string> = {}
    if (!form.name.trim()) errors.name = t('common.required', { field: t('common.name') })
    if (!editing && !form.adapter_id.trim()) errors.adapter_id = t('instances.adapterRequired')
    if (!editing && !form.platform_id.trim()) errors.platform_id = t('instances.platformRequired')
    // Required fields gate creation; editing stays lenient so a stored config
    // can still be emptied (and cleared) field by field.
    if (!editing) {
      for (const field of configFields) {
        const name = field.name
        if (field.required && !configValues[name]?.trim()) {
          errors[`config.${name}`] = t('common.required', { field: name })
        }
      }
    }
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) return

    // `config` is always sent (even as "") so clearing every field really clears it.
    const config = configFields.length > 0
      ? serializeConfig(configValues, configFields)
      : configText.trim()

    setSaving(true)
    setFormError(null)
    const result = editing
      ? await apiClient.updateInstance(editing.id, { name: form.name.trim(), config })
      : await apiClient.createInstance({
          name: form.name.trim(),
          adapter_id: form.adapter_id.trim(),
          platform_id: form.platform_id.trim(),
          config,
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
                        onClick={() => setLogsFor({ id: instance.id, name: instance.name })}
                        aria-label={t('instances.logs')}
                        title={t('instances.logs')}
                      >
                        <ScrollText />
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

            {/* Config comes from the platform's config_fields, not a JSON blob. */}
            {configFields.length > 0 ? (
              <fieldset className="space-y-4">
                <legend className="text-sm font-medium text-text-primary">{t('instances.config')}</legend>
                <p className="-mt-2 text-xs text-text-tertiary">{t('instances.configHint')}</p>
                {configFields.map((field) => {
                  const name = field.name
                  // `cookie` / `token` style names are secrets regardless of the
                  // declared type, which is always `string`.
                  const secret = SECRET_FIELD.test(name)
                  const value = configValues[name] ?? fieldValue(field.default)
                  return (
                    <FormField
                      key={name}
                      label={name}
                      hint={field.description}
                      required={field.required}
                      error={fieldErrors[`config.${name}`]}
                    >
                      {(props) => (
                        <div className="relative">
                          <input
                            {...props}
                            type={secret && !revealed[name] ? 'password' : field.type === 'integer' ? 'number' : 'text'}
                            className={cn(controlClassName, secret && 'pr-10')}
                            value={value}
                            onChange={(event) => setConfigValues({ ...configValues, [name]: event.target.value })}
                          />
                          {secret && (
                            <button
                              type="button"
                              onClick={() => setRevealed({ ...revealed, [name]: !revealed[name] })}
                              aria-label={revealed[name] ? t('auth.hidePassword') : t('auth.showPassword')}
                              aria-pressed={Boolean(revealed[name])}
                              className="absolute inset-y-0 right-0 grid w-9 place-items-center rounded-r-md text-text-tertiary outline-none transition-colors hover:text-text-primary focus-visible:ring-2 focus-visible:ring-ring/50"
                            >
                              {revealed[name] ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                            </button>
                          )}
                        </div>
                      )}
                    </FormField>
                  )
                })}
              </fieldset>
            ) : (
              <FormField label={t('instances.config')} hint={t('instances.configHintRaw')}>
                {(props) => (
                  <textarea
                    {...props}
                    className={cn(controlClassName, 'h-20 resize-y py-2 font-mono text-xs')}
                    value={configText}
                    onChange={(event) => setConfigText(event.target.value)}
                  />
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

      <InstanceLogDrawer instance={logsFor} onClose={() => setLogsFor(null)} />

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
