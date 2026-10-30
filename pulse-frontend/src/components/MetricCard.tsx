import { METRIC_BY_KEY, formatBand, formatValue, metricState, type MetricState } from '../lib/metrics'
import type { MetricKey, Threshold } from '../types/api'
import { Badge } from './ui/Notice'

const STATE: Record<MetricState, { label: string; tone: 'ok' | 'warn' | 'danger' | 'neutral' }> = {
  ok: { label: 'Optimal', tone: 'ok' },
  low: { label: 'Too low', tone: 'danger' },
  high: { label: 'Too high', tone: 'danger' },
  off: { label: 'Monitoring', tone: 'neutral' },
  nodata: { label: 'No data', tone: 'neutral' },
}

export function MetricCard({
  metric,
  value,
  threshold,
  selected,
  onSelect,
}: {
  metric: MetricKey
  value: number | null
  threshold?: Threshold
  selected: boolean
  onSelect: () => void
}) {
  const meta = METRIC_BY_KEY[metric]
  const state = metricState(value, threshold)
  const s = STATE[state]
  const alarming = state === 'low' || state === 'high'

  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      className={`rounded-xl border bg-surface p-4 text-left shadow-card transition hover:border-forest/40 ${
        selected ? 'ring-2 ring-forest' : ''
      } ${alarming ? 'border-danger/40' : 'border-line'}`}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-semibold text-ink-soft">{meta.label}</span>
        <Badge tone={s.tone}>{s.label}</Badge>
      </div>
      <p className={`mt-3 text-2xl font-extrabold tabular-nums sm:text-3xl ${alarming ? 'text-danger' : 'text-ink'}`}>
        {formatValue(value, meta, false)}
        {meta.unit && value !== null ? <span className="ml-1 text-base font-semibold text-muted">{meta.unit}</span> : null}
      </p>
      <p className="mt-2 text-xs text-muted">{formatBand(threshold, meta)}</p>
      <p className="mt-0.5 text-xs text-muted/80">{meta.source}</p>
    </button>
  )
}
