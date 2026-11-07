import {
  cloneElement,
  isValidElement,
  useId,
  type AriaAttributes,
  type ReactElement,
  type ReactNode,
} from 'react'
import { cn } from '@/lib/cn'

/**
 * Shared control styling (shadcn `Input`): `h-9`, transparent ground so the
 * surface shows through, `shadow-xs`, focus = border + 2px ring/50 with no
 * offset. `aria-invalid` swaps to the destructive ring.
 */
export const controlClassName = cn(
  'h-9 w-full min-w-0 rounded-md border border-input bg-transparent px-3 py-1 text-sm text-text-primary shadow-xs',
  'transition-[color,box-shadow] outline-none',
  'file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium',
  'placeholder:text-text-tertiary',
  'focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/50',
  'disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50',
  'dark:bg-input/30',
  'aria-[invalid=true]:border-destructive aria-[invalid=true]:ring-destructive/20 dark:aria-[invalid=true]:ring-destructive/40',
)

export interface FieldControlProps extends AriaAttributes {
  id: string
  'aria-describedby'?: string
  'aria-invalid'?: boolean
  /** Index signature so the props can be spread onto any control. */
  [key: string]: unknown
}

export interface FormFieldProps {
  label: ReactNode
  /** Rendered under the control in muted text; replaced by `error` when set. */
  hint?: ReactNode
  error?: ReactNode
  required?: boolean
  /** Explicit id; generated when omitted. */
  id?: string
  /** Mark the field read-only visually (does not disable the control). */
  disabled?: boolean
  className?: string
  /**
   * Either a render function receiving the wiring props, or one element that
   * receives them via cloneElement.
   */
  children: ReactNode | ((props: FieldControlProps) => ReactNode)
  /** Right-aligned label suffix, e.g. a "unit" or a help link. */
  labelAction?: ReactNode
}

/**
 * label + control + hint/error with the ARIA wiring already attached.
 * Control state is the caller's; this only owns presentation and ids.
 */
export function FormField({
  label,
  hint,
  error,
  required,
  id,
  disabled,
  className,
  children,
  labelAction,
}: FormFieldProps) {
  const generatedId = useId()
  const fieldId = id ?? `field-${generatedId}`
  const hintId = `${fieldId}-hint`
  const errorId = `${fieldId}-error`
  // `error` replaces `hint`, so only ever point at the node actually rendered.
  const describedBy = error ? errorId : hint ? hintId : undefined

  const controlProps: FieldControlProps = {
    id: fieldId,
    'aria-describedby': describedBy,
    'aria-invalid': error ? true : undefined,
  }

  const control =
    typeof children === 'function'
      ? children(controlProps)
      : isValidElement(children)
        ? cloneElement(children as ReactElement<Record<string, unknown>>, controlProps)
        : children

  return (
    <div className={cn('space-y-2', disabled && 'opacity-60', className)}>
      <div className="flex items-baseline justify-between gap-2">
        <label htmlFor={fieldId} className="text-sm leading-none font-medium text-text-primary peer-disabled:opacity-70">
          {label}
          {required && (
            <span aria-hidden className="ml-0.5 text-danger">
              *
            </span>
          )}
        </label>
        {labelAction}
      </div>
      {control}
      {error ? (
        <p id={errorId} className="text-xs font-medium text-danger-fg">
          {error}
        </p>
      ) : (
        hint && (
          <p id={hintId} className="text-xs text-text-tertiary">
            {hint}
          </p>
        )
      )}
    </div>
  )
}

export default FormField
