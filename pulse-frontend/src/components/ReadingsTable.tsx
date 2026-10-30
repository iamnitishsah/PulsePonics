import { formatDateTime } from '../lib/format'
import { METRICS, formatValue, metricState, valueOf } from '../lib/metrics'
import type { Reading, Threshold } from '../types/api'
import { Card, CardHeader } from './ui/Card'

export function ReadingsTable({ readings, thresholds }: { readings: Reading[] | undefined; thresholds: Threshold[] | undefined }) {
  return (
    <Card>
      <CardHeader eyebrow="Latest" title="Recent readings" />
      <div className="overflow-x-auto px-5 pb-5 pt-3">
        <table className="w-full min-w-[720px] border-collapse text-sm">
          <thead>
            <tr className="border-b border-line text-left text-xs font-bold uppercase text-muted">
              <th className="py-2.5 pr-3">Recorded</th>
              {METRICS.map((m) => (
                <th key={m.key} className="px-3 py-2.5 text-right">
                  {m.short}
                  {m.unit ? <span className="ml-1 font-semibold normal-case">({m.unit})</span> : null}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {readings && readings.length > 0 ? (
              readings.map((r) => (
                <tr key={r.id} className="border-b border-line last:border-0">
                  <td className="whitespace-nowrap py-2.5 pr-3 font-medium text-ink">{formatDateTime(r.recorded_at)}</td>
                  {METRICS.map((m) => {
                    const v = valueOf(r, m.key)
                    const state = metricState(v, thresholds?.find((t) => t.metric === m.key))
                    const out = state === 'low' || state === 'high'
                    return (
                      <td
                        key={m.key}
                        className={`whitespace-nowrap px-3 py-2.5 text-right tabular-nums ${out ? 'font-semibold text-danger' : v === null ? 'text-muted' : 'text-ink'}`}
                      >
                        {formatValue(v, m, false)}
                      </td>
                    )
                  })}
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={METRICS.length + 1} className="py-6 text-center text-muted">
                  No readings yet. They appear here as soon as the device publishes.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Card>
  )
}
