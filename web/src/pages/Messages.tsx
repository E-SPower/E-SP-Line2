import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCheck, Inbox, Plus, Send } from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TableShell } from '../components/ui/table-shell'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { EmptyState } from '../components/ui/empty-state'
import { ErrorState } from '../components/ui/error-state'
import { SkeletonRows } from '../components/ui/skeleton'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../components/ui/dialog'
import { FormField, controlClassName } from '../components/ui/form-field'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'
import { cn } from '@/lib/cn'

interface Message {
  id: string
  platform_id: string
  instance_id: string
  sender_id: string
  sender_name: string
  message_type: string
  message_content: string
  status: string
  created_at: string
}

/** Status → badge variant, so the table never re-derives colours. */
const statusVariant = (status: string) => {
  if (status === 'acked') return 'success' as const
  if (status === 'delivered') return 'info' as const
  return 'warning' as const
}

const statusLabelKey = (status: string) => {
  if (status === 'acked') return 'state.status.acked'
  if (status === 'delivered') return 'state.status.delivered'
  return 'state.status.pending'
}

type Filter = 'all' | 'pending'

interface InstanceOption {
  id: string
  name: string
  platform_id: string
}

/** The six outbound command types from the protocol (§ Outbound command). */
const COMMAND_TYPES = ['send_text', 'send_image', 'upload_media', 'create_conversation', 'get_history', 'refresh_token']

const emptyCommandForm = { instance_id: '', command_type: 'send_text', payload: '{\n  "text": ""\n}' }

export default function Messages() {
  const { t } = useTranslation()
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<Filter>('all')
  const [ackingId, setAckingId] = useState<string | null>(null)
  // Stored as a key + params so the badge re-translates on a language switch.
  const [notice, setNotice] = useState<{ key: string; params?: Record<string, string> } | null>(null)

  // "Create" dispatches an outbound command — messages are inbound, so there is
  // no message to author here.
  const [commandOpen, setCommandOpen] = useState(false)
  const [instances, setInstances] = useState<InstanceOption[]>([])
  const [instancesLoading, setInstancesLoading] = useState(false)
  const [commandForm, setCommandForm] = useState(emptyCommandForm)
  const [commandErrors, setCommandErrors] = useState<Record<string, string>>({})
  const [commandError, setCommandError] = useState<string | null>(null)
  const [sending, setSending] = useState(false)

  const loadMessages = useCallback(async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getMessages()
    if (result.error) {
      setError(result.error)
      setMessages([])
    } else {
      setMessages(result.data || [])
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    void loadMessages()
  }, [loadMessages])

  const handleAck = async (id: string) => {
    setAckingId(id)
    setError(null)
    try {
      const result = await apiClient.ackMessage(id)
      if (result.error) {
        setError(result.error)
        return
      }
      await loadMessages()
    } finally {
      setAckingId(null)
    }
  }

  const pendingCount = useMemo(() => messages.filter((message) => message.status !== 'acked').length, [messages])

  const loadInstances = useCallback(async () => {
    setInstancesLoading(true)
    const result = await apiClient.getInstances(100, 0)
    setInstancesLoading(false)
    if (!result.error) setInstances(result.data || [])
  }, [])

  const openCommand = () => {
    setCommandForm(emptyCommandForm)
    setCommandErrors({})
    setCommandError(null)
    setCommandOpen(true)
    void loadInstances()
  }

  const payloadOf = (commandType: string) =>
    commandType === 'send_text'
      ? '{\n  "text": ""\n}'
      : commandType === 'send_image'
        ? '{\n  "url": "",\n  "width": 0,\n  "height": 0\n}'
        : '{\n  "target_id": ""\n}'

  const changeCommandType = (commandType: string) => {
    setCommandForm({ ...commandForm, command_type: commandType, payload: payloadOf(commandType) })
    setCommandErrors({})
  }

  const submitCommand = async () => {
    const errors: Record<string, string> = {}
    if (!commandForm.instance_id) errors.instance_id = t('messages.instanceRequired')

    let payload: unknown = {}
    try {
      payload = commandForm.payload.trim() ? JSON.parse(commandForm.payload) : {}
    } catch (parseError) {
      errors.payload = t('messages.payloadInvalid', { error: (parseError as Error).message })
    }
    setCommandErrors(errors)
    if (Object.keys(errors).length > 0) return

    setSending(true)
    setCommandError(null)
    const result = await apiClient.createCommand({
      instance_id: commandForm.instance_id,
      command_type: commandForm.command_type,
      payload,
    })
    setSending(false)

    if (result.error) {
      setCommandError(result.error)
      return
    }
    setCommandOpen(false)
    setNotice({ key: 'messages.commandCreated', params: { type: commandForm.command_type } })
    await loadMessages()
  }

  const visible = useMemo(
    () => (filter === 'pending' ? messages.filter((message) => message.status !== 'acked') : messages),
    [messages, filter],
  )

  const filters: { id: Filter; label: string; count: number }[] = [
    { id: 'all', label: t('common.all'), count: messages.length },
    { id: 'pending', label: t('messages.pendingOnly'), count: pendingCount },
  ]

  return (
    <>
      <PageHeader
        title={t('messages.title')}
        description={t('messages.subtitle')}
        actions={
          <Button variant="primary" size="sm" onClick={openCommand}>
            <Plus />
            {t('common.create')}
          </Button>
        }
      />

      {notice && (
        <Badge variant="success" role="status" aria-live="polite">
          {t(notice.key, notice.params)}
        </Badge>
      )}

      {error ? (
        <ErrorState
          title={t('state.loadFailed')}
          description={error || t('state.loadFailedHint')}
          onRetry={loadMessages}
          retryLabel={t('common.retry')}
        />
      ) : (
        <TableShell
          busy={loading}
          striped
          toolbar={
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div role="group" aria-label={t('common.status')} className="flex items-center gap-1">
                {filters.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setFilter(item.id)}
                    aria-pressed={filter === item.id}
                    className={cn(
                      'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors',
                      filter === item.id
                        ? 'border-transparent bg-primary text-primary-foreground'
                        : 'border-border bg-surface-raised text-text-tertiary hover:bg-surface-hover hover:text-text-secondary',
                    )}
                  >
                    {item.label}
                    <span className="tabular opacity-70">{item.count}</span>
                  </button>
                ))}
              </div>
              <Button variant="ghost" size="sm" onClick={() => void loadMessages()}>
                {t('common.refresh')}
              </Button>
            </div>
          }
          empty={
            loading ? (
              <SkeletonRows rows={6} columns={5} />
            ) : visible.length === 0 ? (
              <EmptyState
                icon={Inbox}
                title={t('messages.empty')}
                description={t('messages.emptyHint')}
              />
            ) : undefined
          }
        >
          <TableHead>
            <tr>
              <TableHeaderCell className="min-w-[9rem]">{t('messages.instance')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[9rem]">{t('messages.sender')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[16rem]">{t('messages.content')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[9rem]">{t('messages.timestamp')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[6rem]">{t('common.status')}</TableHeaderCell>
              <TableHeaderCell className="min-w-[5rem] text-right">{t('common.actions')}</TableHeaderCell>
            </tr>
          </TableHead>
          <TableBody>
            {visible.map((message) => (
              <TableRow key={message.id}>
                <TableCell className="min-w-[9rem] font-mono text-xs text-text-tertiary">
                  {message.instance_id}
                </TableCell>
                <TableCell className="min-w-[9rem] font-medium text-text-primary">
                  <span className="truncate-cell" title={message.sender_name || message.sender_id}>
                    {message.sender_name || message.sender_id}
                  </span>
                </TableCell>
                <TableCell className="min-w-[16rem]">
                  <span className="truncate-cell text-text-secondary" title={message.message_content}>
                    {message.message_content}
                  </span>
                </TableCell>
                <TableCell className="tabular min-w-[9rem] whitespace-nowrap text-xs text-text-tertiary">
                  {new Date(message.created_at).toLocaleString()}
                </TableCell>
                <TableCell className="min-w-[6rem]">
                  <Badge variant={statusVariant(message.status)} dot>
                    {t(statusLabelKey(message.status))}
                  </Badge>
                </TableCell>
                <TableCell className="min-w-[5rem] text-right">
                  {message.status !== 'acked' && (
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-success-fg hover:bg-success-bg"
                      onClick={() => void handleAck(message.id)}
                      disabled={ackingId === message.id}
                      aria-label={t('messages.ackAction')}
                      title={t('messages.ackAction')}
                    >
                      <CheckCheck />
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </TableShell>
      )}

      <Dialog open={commandOpen} onOpenChange={(open) => !sending && setCommandOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('messages.commandTitle')}</DialogTitle>
            <DialogDescription>{t('messages.commandHint')}</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <FormField label={t('messages.instance')} required error={commandErrors.instance_id}>
              {(props) => (
                <div {...props}>
                  <Select
                    value={commandForm.instance_id}
                    onValueChange={(value) => setCommandForm({ ...commandForm, instance_id: value })}
                    disabled={instancesLoading || instances.length === 0}
                  >
                    <SelectTrigger aria-invalid={commandErrors.instance_id ? true : undefined}>
                      <SelectValue
                        placeholder={instancesLoading ? t('state.loadingData') : t('messages.instancePlaceholder')}
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {instances.map((instance) => (
                        <SelectItem key={instance.id} value={instance.id}>
                          {instance.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
            </FormField>

            {!instancesLoading && instances.length === 0 && (
              <p className="rounded-md border border-warning-border bg-warning-bg px-2.5 py-2 text-xs text-warning-fg">
                {t('messages.noInstances')}
              </p>
            )}

            <FormField label={t('messages.commandType')} hint={t('messages.commandTypeHint')} required>
              {(props) => (
                <div {...props}>
                  <Select value={commandForm.command_type} onValueChange={changeCommandType}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {COMMAND_TYPES.map((type) => (
                        <SelectItem key={type} value={type}>
                          {type}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
            </FormField>

            <FormField label="payload" hint={t('messages.payloadHint')} error={commandErrors.payload}>
              {(props) => (
                <textarea
                  {...props}
                  className={cn(controlClassName, 'h-28 resize-y py-2 font-mono text-xs')}
                  value={commandForm.payload}
                  onChange={(event) => setCommandForm({ ...commandForm, payload: event.target.value })}
                  spellCheck={false}
                />
              )}
            </FormField>

            {commandError && (
              <p role="alert" className="rounded-md border border-danger-border bg-danger-bg px-2.5 py-2 text-xs text-danger-fg">
                {commandError}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setCommandOpen(false)} disabled={sending}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" size="sm" onClick={() => void submitCommand()} disabled={sending || instances.length === 0}>
              <Send />
              {sending ? t('messages.sending') : t('messages.send')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
