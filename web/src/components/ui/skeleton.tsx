import { forwardRef, type HTMLAttributes } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/cn'

/** shadcn skeleton: a flat `animate-pulse` neutral block, rounded-md. */
const skeletonVariants = cva('bg-surface-inset', {
  variants: {
    shape: {
      block: 'rounded-md',
      line: 'rounded-md',
      circle: 'rounded-full',
    },
    animated: {
      true: 'animate-pulse',
      false: '',
    },
  },
  defaultVariants: {
    shape: 'block',
    animated: true,
  },
})

export interface SkeletonProps extends HTMLAttributes<HTMLDivElement>, VariantProps<typeof skeletonVariants> {}

export const Skeleton = forwardRef<HTMLDivElement, SkeletonProps>(
  ({ className, shape, animated, ...props }, ref) => (
    <div ref={ref} aria-hidden className={cn(skeletonVariants({ shape, animated }), className)} {...props} />
  ),
)
Skeleton.displayName = 'Skeleton'

/** A block of text lines with a ragged last line. */
export function SkeletonText({ lines = 3, className }: { lines?: number; className?: string }) {
  return (
    <div className={cn('space-y-2', className)} aria-hidden>
      {Array.from({ length: lines }).map((_, index) => (
        <Skeleton key={index} shape="line" className={cn('h-3.5', index === lines - 1 ? 'w-2/3' : 'w-full')} />
      ))}
    </div>
  )
}

/** Placeholder for the table rows while a list loads. */
export function SkeletonRows({ rows = 5, columns = 4 }: { rows?: number; columns?: number }) {
  return (
    <div className="divide-y divide-border" aria-hidden>
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <div key={rowIndex} className="flex items-center gap-4 px-4 py-3.5">
          {Array.from({ length: columns }).map((_, columnIndex) => (
            <Skeleton
              key={columnIndex}
              shape="line"
              className={cn('h-3.5', columnIndex === 0 ? 'w-40' : 'w-24')}
            />
          ))}
        </div>
      ))}
    </div>
  )
}

export { skeletonVariants }
