import type { ComponentType, ReactNode } from 'react'
import { cn } from '@/lib/cn'

export interface EmptyStateProps {
  /** A lucide icon component, rendered in a neutral well. */
  icon?: ComponentType<{ className?: string }>
  title: ReactNode
  description?: ReactNode
  /** Optional call-to-action, typically a <Button>. */
  action?: ReactNode
  /** `page` fills a whole view, `inline` sits inside a card or table. */
  size?: 'page' | 'inline'
  className?: string
}

/**
 * The one empty state. Never write a bare "No records found" div.
 * Always say what is missing and what the reader can do about it.
 */
export function EmptyState({ icon: Icon, title, description, action, size = 'inline', className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        size === 'page' ? 'min-h-[22rem] gap-4 px-6 py-16' : 'gap-3 px-6 py-14',
        className,
      )}
    >
      {Icon && (
        <span
          aria-hidden
          className={cn(
            'flex items-center justify-center rounded-full border border-border bg-surface-inset text-text-tertiary',
            size === 'page' ? 'size-12 [&_svg]:size-5' : 'size-10 [&_svg]:size-4',
          )}
        >
          <Icon />
        </span>
      )}
      <div className="max-w-sm space-y-1">
        <p className={cn('font-medium text-text-primary', size === 'page' ? 'text-base' : 'text-sm')}>{title}</p>
        {description && <p className="text-sm text-text-tertiary">{description}</p>}
      </div>
      {action && <div className="mt-1">{action}</div>}
    </div>
  )
}

export default EmptyState
