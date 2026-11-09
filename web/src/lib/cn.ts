import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * Merge conditional class names and resolve Tailwind conflicts.
 *
 * Always use this instead of template-string class concatenation so a
 * later utility can override an earlier one (`px-6` beating `px-3`).
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
