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
  /**
   * Hang the faint aurora behind the copy instead of leaving a flat void.
   *
   * Only for page-sized blanks. Geometry, timing and the reduced-motion still
   * all come from `index.css` (`.aurora`, `.aurora-blob-*`); the only local
   * decision is the wash colour — the login hero's `--primary-foreground`
   * wash disappears on an ordinary surface, so re-tint it to `--ring` and
   * drop it a stop below the hero's already-low alpha.
   */
  aurora?: boolean
  className?: string
}

/**
 * The one empty state. Never write a bare "No records found" div.
 * Always say what is missing and what the reader can do about it.
 */
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  size = 'inline',
  aurora = false,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        size === 'page' ? 'min-h-[22rem] gap-4 px-6 py-16' : 'gap-3 px-6 py-14',
        // `isolate` + `-z-10` keeps the aurora behind the copy but above the
        // card it sits in; `overflow-hidden` crops the bodies at the edge.
        aurora && 'relative isolate overflow-hidden',
        className,
      )}
    >
      {aurora && (
        /*
         * Same primitive as the login hero, re-tinted: `--primary-foreground`
         * only shapes on a solid `primary` plane, so on a surface it becomes
         * `--ring` and the peaks drop one stop (spec §7.3: 0.14 cap here,
         * 0.2 is reserved for the primary plane). Geometry, timing and the
         * reduced-motion still are inherited untouched.
         */
        <div
          aria-hidden
          className="aurora -z-10 [--primary-foreground:var(--ring)]"
        >
          <span className="aurora-blob aurora-blob-a ![--blob-alpha:0.14] dark:![--blob-alpha:0.13]" />
          <span className="aurora-blob aurora-blob-b ![--blob-alpha:0.11] dark:![--blob-alpha:0.1]" />
          <span className="aurora-blob aurora-blob-c ![--blob-alpha:0.09] dark:![--blob-alpha:0.08]" />
        </div>
      )}
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
