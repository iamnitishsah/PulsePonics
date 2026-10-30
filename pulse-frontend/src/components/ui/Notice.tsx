import type { ReactNode } from 'react'

type Tone = 'error' | 'warning' | 'info' | 'success'

const tones: Record<Tone, string> = {
  error: 'border-danger/25 bg-danger-soft text-danger',
  warning: 'border-warn/25 bg-warn-soft text-warn',
  info: 'border-forest/20 bg-mint-soft text-forest-dark',
  success: 'border-ok/25 bg-mint-soft text-ok',
}

export function Notice({
  tone = 'error',
  title,
  children,
  action,
}: {
  tone?: Tone
  title?: string
  children?: ReactNode
  action?: ReactNode
}) {
  return (
    <div
      className={`flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm ${tones[tone]}`}
      role={tone === 'error' ? 'alert' : 'status'}
    >
      <div>
        {title ? <strong className="font-semibold">{title} </strong> : null}
        {children}
      </div>
      {action}
    </div>
  )
}

export function Badge({ tone, children }: { tone: 'ok' | 'warn' | 'danger' | 'neutral'; children: ReactNode }) {
  const styles = {
    ok: 'bg-mint-soft text-ok',
    warn: 'bg-warn-soft text-warn',
    danger: 'bg-danger-soft text-danger',
    neutral: 'bg-ink/5 text-ink-soft',
  }[tone]
  return (
    <span className={`inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-xs font-semibold ${styles}`}>
      {children}
    </span>
  )
}

export function EmptyState({ title, children, action }: { title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="grid place-items-center rounded-xl border border-dashed border-ink/20 bg-surface/60 px-6 py-12 text-center">
      <div className="max-w-md">
        <h3 className="text-base font-bold text-ink">{title}</h3>
        {children ? <div className="mt-2 text-sm text-muted">{children}</div> : null}
        {action ? <div className="mt-5 flex justify-center">{action}</div> : null}
      </div>
    </div>
  )
}
