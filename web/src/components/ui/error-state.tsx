import type { ReactNode } from 'react'
import { AlertTriangle, RotateCw } from 'lucide-react'
import { Button } from './button'
import { Card } from './card'
import { cn } from '@/lib/cn'

export interface ErrorStateProps {
  /** What failed, in the reader's words. */
  title: ReactNode
  /** The raw reason (API message, status). */
  description?: ReactNode
  /** Retry handler; the button is hidden when omitted. */
  onRetry?: () => void
  retryLabel?: ReactNode
  className?: string
}

/**
 * The single error presentation. Always a danger-tinted card, always with a
 * retry affordance when the caller can retry. Never a bare red div.
 */
export function ErrorState({ title, description, onRetry, retryLabel = 'Retry', className }: ErrorStateProps) {
  return (
    <Card role="alert" variant="danger" padding="tight" className={cn('flex items-start gap-3', className)}>
      <span aria-hidden className="mt-0.5 shrink-0">
        <AlertTriangle className="size-4" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{title}</p>
        {description && <p className="mt-1 break-words text-sm opacity-90">{description}</p>}
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry} className="shrink-0 border-danger-border bg-transparent">
          <RotateCw />
          {retryLabel}
        </Button>
      )}
    </Card>
  )
}

export default ErrorState
