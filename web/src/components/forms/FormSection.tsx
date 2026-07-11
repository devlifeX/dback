import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function FormSection({
  title,
  description,
  children,
  className,
}: {
  title: string
  description?: string
  children: ReactNode
  className?: string
}) {
  return (
    <section className={cn('space-y-4', className)}>
      <div>
        <h3 className="text-sm font-medium">{title}</h3>
        {description ? <p className="text-xs text-[hsl(var(--muted-foreground))]">{description}</p> : null}
      </div>
      <div className="grid gap-4 sm:grid-cols-2">{children}</div>
    </section>
  )
}
