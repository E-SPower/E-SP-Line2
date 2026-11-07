import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCheck, Inbox, Plus } from 'lucide-react'
import apiClient from '../api/client'
import { PageHeader } from '../components/ui/page-header'
import { TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TableShell } from '../components/ui/table-shell'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { EmptyState } from '../components/ui/empty-state'
import { ErrorState } from '../components/ui/error-state'
import { SkeletonRows } from '../components/ui/skeleton'
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

export default function Messages() {
  const { t } = useTranslation()
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<Filter>('all')
  const [ackingId, setAckingId] = useState<string | null>(null)

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
          <Button variant="primary" size="sm">
            <Plus />
            {t('common.create')}
          </Button>
        }
      />

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
    </>
  )
}
