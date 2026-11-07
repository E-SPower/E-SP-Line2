import { forwardRef, type HTMLAttributes } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/cn'

/**
 * shadcn/ui badge: pill-shaped (`rounded-full`), transparent border,
 * `text-xs font-medium`. Colour is carried by the tinted ground only.
 * Call sites pass a semantic variant — never a colour triple.
 *
 * Mapping used by the console:
 *   success  → running / active / acked
 *   warning  → pending / degraded
 *   danger   → error / disabled-by-failure
 *   info     → in-flight / delivered / informational
 *   neutral  → stopped / inactive / unknown
 */
const badgeVariants = cva(
  'inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full border border-transparent px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/50 [&>svg]:pointer-events-none [&>svg]:size-3',
  {
    variants: {
      variant: {
        success: 'bg-success-bg text-success-fg',
        warning: 'bg-warning-bg text-warning-fg',
        danger: 'bg-danger-bg text-danger-fg',
        info: 'bg-info-bg text-info-fg',
        neutral: 'bg-surface-inset text-text-secondary',
        outline: 'border-border text-text-primary',
      },
      /** `dot` renders a leading status dot without an icon. */
      dot: {
        true: 'pl-1.5',
        false: '',
      },
    },
    defaultVariants: {
      variant: 'neutral',
      dot: false,
    },
  },
)

const dotColor = {
  success: 'bg-success',
  warning: 'bg-warning',
  danger: 'bg-danger',
  info: 'bg-info',
  neutral: 'bg-text-tertiary',
  outline: 'bg-text-tertiary',
} as const

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export const Badge = forwardRef<HTMLSpanElement, BadgeProps>(
  ({ className, variant = 'neutral', dot, children, ...props }, ref) => (
    <span ref={ref} className={cn(badgeVariants({ variant, dot }), className)} {...props}>
      {dot && <span aria-hidden className={cn('size-1.5 rounded-full', dotColor[variant ?? 'neutral'])} />}
      {children}
    </span>
  ),
)

Badge.displayName = 'Badge'

export { badgeVariants }
