import { type ReactNode } from 'react'
import * as AlertDialogPrimitive from '@radix-ui/react-alert-dialog'
import { Loader2 } from 'lucide-react'
import { Button } from './button'
import { cn } from '@/lib/cn'

/**
 * AlertDialog wrapper for destructive confirmations. The caller owns the open
 * state and the async action so the dialog can stay up while the request runs
 * and surface the failure inline instead of closing silently.
 */
export interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  confirmLabel: ReactNode
  cancelLabel: ReactNode
  onConfirm: () => void
  /** While true the confirm button shows a spinner and both buttons lock. */
  pending?: boolean
  /** Request failure, rendered above the actions. */
  error?: ReactNode
  /** Confirm button label while pending. */
  pendingLabel?: ReactNode
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  onConfirm,
  pending = false,
  error,
  pendingLabel,
}: ConfirmDialogProps) {
  return (
    <AlertDialogPrimitive.Root open={open} onOpenChange={(next) => !pending && onOpenChange(next)}>
      <AlertDialogPrimitive.Portal>
        <AlertDialogPrimitive.Overlay
          className={cn(
            'fixed inset-0 z-50 bg-text-primary/45 backdrop-blur-[1px]',
            'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
          )}
        />
        <AlertDialogPrimitive.Content
          className={cn(
            'fixed top-1/2 left-1/2 z-50 grid w-[calc(100vw-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 gap-4',
            'rounded-xl border border-border bg-surface-overlay p-6 shadow-overlay',
            'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
          )}
        >
          <div className="flex flex-col gap-1.5">
            <AlertDialogPrimitive.Title className="text-base font-semibold text-text-primary">
              {title}
            </AlertDialogPrimitive.Title>
            {description && (
              <AlertDialogPrimitive.Description className="text-sm text-text-tertiary">
                {description}
              </AlertDialogPrimitive.Description>
            )}
          </div>

          {error && (
            <p role="alert" className="rounded-md border border-danger-border bg-danger-bg px-2.5 py-2 text-xs text-danger-fg">
              {error}
            </p>
          )}

          <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <AlertDialogPrimitive.Cancel asChild>
              <Button variant="outline" size="sm" disabled={pending}>
                {cancelLabel}
              </Button>
            </AlertDialogPrimitive.Cancel>
            <Button variant="danger" size="sm" onClick={onConfirm} disabled={pending}>
              {pending && <Loader2 className="animate-spin" />}
              {pending ? pendingLabel || confirmLabel : confirmLabel}
            </Button>
          </div>
        </AlertDialogPrimitive.Content>
      </AlertDialogPrimitive.Portal>
    </AlertDialogPrimitive.Root>
  )
}

export default ConfirmDialog
