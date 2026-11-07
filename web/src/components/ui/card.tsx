import { forwardRef, type HTMLAttributes } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/cn'

/**
 * shadcn/ui card: a flex column with `gap-6`, `rounded-xl`, one hairline
 * border and `shadow-sm`. Padding is vertical only (`py-6`) — sub-components
 * own their horizontal `px-6`. Pass `padding="none"` when a caller supplies
 * its own geometry (tables, stat tiles).
 */
const cardVariants = cva('flex flex-col gap-6 rounded-xl border bg-surface-raised py-6 text-text-primary', {
  variants: {
    variant: {
      default: 'border-border shadow-sm',
      sunken: 'border-border-subtle bg-surface-sunken shadow-none',
      elevated: 'border-border shadow-md',
      success: 'border-transparent bg-success-bg text-success-fg shadow-none',
      warning: 'border-transparent bg-warning-bg text-warning-fg shadow-none',
      danger: 'border-transparent bg-danger-bg text-danger-fg shadow-none',
      info: 'border-transparent bg-info-bg text-info-fg shadow-none',
      ghost: 'border-transparent bg-transparent shadow-none',
    },
    padding: {
      none: 'py-0 gap-0',
      tight: 'py-4 gap-4',
      default: 'py-6',
      roomy: 'py-8 gap-8',
    },
  },
  defaultVariants: {
    variant: 'default',
    padding: 'default',
  },
})

export interface CardProps extends HTMLAttributes<HTMLDivElement>, VariantProps<typeof cardVariants> {}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, variant, padding, ...props }, ref) => (
    <div ref={ref} className={cn(cardVariants({ variant, padding }), className)} {...props} />
  ),
)
Card.displayName = 'Card'

/* Header is a 2-column grid so `CardAction` can sit top-right without a
 * wrapper row; without an action it collapses to one column. */
export const CardHeader = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('grid auto-rows-min items-start gap-2 px-6', className)}
      {...props}
    />
  ),
)
CardHeader.displayName = 'CardHeader'

export const CardTitle = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('text-base leading-none font-semibold tracking-tight', className)} {...props} />
  ),
)
CardTitle.displayName = 'CardTitle'

export const CardDescription = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('text-sm text-text-tertiary', className)} {...props} />
  ),
)
CardDescription.displayName = 'CardDescription'

/** Top-right slot inside `CardHeader` (shadcn `CardAction`). */
export const CardAction = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('col-start-2 row-span-2 row-start-1 self-start justify-self-end', className)} {...props} />
  ),
)
CardAction.displayName = 'CardAction'

export const CardBody = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => <div ref={ref} className={cn('px-6', className)} {...props} />,
)
CardBody.displayName = 'CardBody'

export const CardFooter = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('flex items-center px-6', className)} {...props} />
  ),
)
CardFooter.displayName = 'CardFooter'
