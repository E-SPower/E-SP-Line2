import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

export interface PageHeaderProps {
  title: ReactNode
  description?: ReactNode
  /** Right-aligned primary actions. Keep to one primary button. */
  actions?: ReactNode
  /** Small text above the title — section or breadcrumb-ish context. */
  eyebrow?: ReactNode
  /** Hairline rule under the masthead. On by default, as shadcn pages do. */
  divider?: boolean
  className?: string
}

/**
 * Page masthead: title, one-line purpose, and the page's actions, closing with
 * a hairline rule. Sits directly under the fixed top bar.
 *
 * Under horizontal pressure the actions wrap onto their own line instead of
 * pushing the masthead past the content edge, and the title column is the one
 * that gives way (`min-w-0 flex-1`) so long titles truncate-free wrap.
 */
export function PageHeader({ title, description, actions, eyebrow, divider = true, className }: PageHeaderProps) {
  return (
    <header
      className={cn(
        'flex min-w-0 flex-col gap-4 pb-5 sm:flex-row sm:items-start sm:justify-between',
        divider && 'border-b border-border',
        className,
      )}
    >
      <div className="min-w-0 flex-1 space-y-1.5">
        {eyebrow && (
          <p className="text-xs font-medium text-text-tertiary">{eyebrow}</p>
        )}
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary [overflow-wrap:anywhere]">{title}</h1>
        {description && <p className="max-w-2xl text-sm text-text-tertiary">{description}</p>}
      </div>
      {actions && (
        <div className="flex min-w-0 flex-wrap items-center gap-2 sm:justify-end">{actions}</div>
      )}
    </header>
  )
}

export default PageHeader
