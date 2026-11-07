import { forwardRef, type HTMLAttributes, type ReactNode, type ThHTMLAttributes, type TdHTMLAttributes } from 'react'
import { cn } from '@/lib/cn'

export interface TableShellProps extends HTMLAttributes<HTMLDivElement> {
  /** Optional empty state rendered inside the scrolling region. */
  empty?: ReactNode
  /** Rendered above the table, inside the card chrome. */
  toolbar?: ReactNode
  /** Zebra rows — use on long, scan-heavy lists only. */
  striped?: boolean
  /** Announce a busy state to assistive tech. */
  busy?: boolean
}

/**
 * The only table container in the console. Owns the shadcn card chrome
 * (`rounded-xl border shadow-sm`), muted header, divider rows, optional zebra
 * striping, horizontal scrolling on narrow screens, and the empty slot.
 * Cells keep their own min-widths and truncation.
 */
export const TableShell = forwardRef<HTMLDivElement, TableShellProps>(
  ({ className, children, empty, toolbar, striped, busy, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('overflow-hidden rounded-xl border border-border bg-surface-raised shadow-sm', className)}
      {...props}
    >
      {toolbar && (
        <div className="border-b border-border bg-surface-raised px-4 py-3">{toolbar}</div>
      )}
      <div className="w-full overflow-x-auto">
        {empty ? (
          empty
        ) : (
          <table
            aria-busy={busy || undefined}
            className={cn('w-full min-w-[44rem] caption-bottom border-collapse text-left text-sm', striped && 'zebra')}
          >
            {children}
          </table>
        )}
      </div>
    </div>
  ),
)

TableShell.displayName = 'TableShell'

export const TableHead = forwardRef<HTMLTableSectionElement, HTMLAttributes<HTMLTableSectionElement>>(
  ({ className, ...props }, ref) => (
    <thead
      ref={ref}
      className={cn('border-b border-border text-2xs font-medium uppercase tracking-wide text-text-tertiary', className)}
      {...props}
    />
  ),
)
TableHead.displayName = 'TableHead'

export const TableBody = forwardRef<HTMLTableSectionElement, HTMLAttributes<HTMLTableSectionElement>>(
  ({ className, ...props }, ref) => <tbody ref={ref} className={cn('divide-y divide-border', className)} {...props} />,
)
TableBody.displayName = 'TableBody'

export const TableRow = forwardRef<HTMLTableRowElement, HTMLAttributes<HTMLTableRowElement>>(
  ({ className, ...props }, ref) => (
    <tr
      ref={ref}
      className={cn('transition-colors hover:bg-surface-hover/60 data-[state=selected]:bg-surface-hover', className)}
      {...props}
    />
  ),
)
TableRow.displayName = 'TableRow'

export const TableHeaderCell = forwardRef<HTMLTableCellElement, ThHTMLAttributes<HTMLTableCellElement>>(
  ({ className, ...props }, ref) => (
    <th
      ref={ref}
      scope="col"
      className={cn('h-10 whitespace-nowrap px-4 py-3 text-left align-middle font-medium', className)}
      {...props}
    />
  ),
)
TableHeaderCell.displayName = 'TableHeaderCell'

export const TableCell = forwardRef<HTMLTableCellElement, TdHTMLAttributes<HTMLTableCellElement>>(
  ({ className, ...props }, ref) => (
    <td ref={ref} className={cn('px-4 py-3 align-middle text-text-secondary', className)} {...props} />
  ),
)
TableCell.displayName = 'TableCell'
