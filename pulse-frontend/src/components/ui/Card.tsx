import type { HTMLAttributes, ReactNode } from 'react'

export function Card({ className = '', children, ...rest }: HTMLAttributes<HTMLElement>) {
  return (
    <section className={`rounded-xl border border-line bg-surface shadow-card ${className}`} {...rest}>
      {children}
    </section>
  )
}

export function CardHeader({
  eyebrow,
  title,
  actions,
}: {
  eyebrow?: string
  title: ReactNode
  actions?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-3 px-5 pt-5">
      <div className="min-w-0">
        {eyebrow ? (
          <p className="text-xs font-bold uppercase tracking-wide text-forest">{eyebrow}</p>
        ) : null}
        <h2 className="text-lg font-bold leading-tight text-ink">{title}</h2>
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}
