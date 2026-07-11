import * as React from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'

export const Checkbox = React.forwardRef<
  HTMLButtonElement,
  Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> & {
    checked?: boolean
    onCheckedChange?: (checked: boolean) => void
  }
>(({ className, checked = false, onCheckedChange, disabled, ...props }, ref) => (
  <button
    type="button"
    role="checkbox"
    aria-checked={checked}
    ref={ref}
    disabled={disabled}
    className={cn(
      'peer inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--primary))] disabled:cursor-not-allowed disabled:opacity-50',
      checked && 'border-[hsl(var(--primary))] bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))]',
      className,
    )}
    onClick={() => onCheckedChange?.(!checked)}
    {...props}
  >
    {checked ? <Check className="h-3 w-3" /> : null}
  </button>
))
Checkbox.displayName = 'Checkbox'
