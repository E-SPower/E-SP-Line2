import type { ComponentType, ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { Card } from './card'
import { Skeleton } from './skeleton'

export type StatTone = 'neutral' | 'primary' | 'success' | 'warning' | 'danger' | 'info'

const toneValue: Record<StatTone, string> = {
  neutral: 'text-text-primary',
  primary: 'text-text-primary',
  success: 'text-success-fg',
  warning: 'text-warning-fg',
  danger: 'text-danger-fg',
  info: 'text-info-fg',
}

export interface StatCardProps {
  label: ReactNode
  value: ReactNode
  icon?: ComponentType<{ className?: string }>
  tone?: StatTone
  /** Short delta/caption under the number. */
  caption?: ReactNode
  loading?: boolean
  className?: string
}

/**
 * A single metric in shadcn's "stat" arrangement: muted label, large tabular
 * number, optional muted icon top-right. Colour only when the metric is a
 * status (success/danger).
 */
export function StatCard({ label, value, icon: Icon, tone = 'neutral', caption, loading, className }: StatCardProps) {
  return (
    <Card padding="none" className={cn('px-5 py-4', className)}>
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm font-medium text-text-tertiary">{label}</p>
        {Icon && <Icon className="size-4 shrink-0 text-text-tertiary" />}
      </div>
      {loading ? (
        <Skeleton shape="line" className="mt-3 h-7 w-16" />
      ) : (
        <p className={cn('tabular mt-2 text-2xl font-semibold tracking-tight', toneValue[tone])}>{value}</p>
      )}
      {caption && !loading && <p className="mt-1 text-xs text-text-tertiary">{caption}</p>}
    </Card>
  )
}

/** Row of stat cards in a responsive grid. */
export function StatGrid({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('grid grid-cols-2 gap-4 lg:grid-cols-4', className)}>{children}</div>
}

export default StatCard
