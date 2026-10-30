import { useState, type FormEvent } from 'react'
import { ApiError, api, errorMessage } from '../lib/api'
import { METRICS, METRIC_BY_KEY } from '../lib/metrics'
import type { MetricKey, Threshold } from '../types/api'
import { Button } from './ui/Button'
import { Card, CardHeader } from './ui/Card'
import { Notice } from './ui/Notice'

type Row = { metric: MetricKey; min: string; max: string; enabled: boolean }

const toRow = (t: Threshold): Row => ({
  metric: t.metric,
  min: t.min === null ? '' : String(t.min),
  max: t.max === null ? '' : String(t.max),
  enabled: t.enabled,
})

function parse(value: string): number | null | 'invalid' {
  const trimmed = value.trim()
  if (trimmed === '') return null
  const n = Number(trimmed)
  return Number.isFinite(n) ? n : 'invalid'
}

/** Client-side check mirroring the backend rules, for instant feedback. */
function validate(row: Row): string | null {
  const meta = METRIC_BY_KEY[row.metric]
  const min = parse(row.min)
  const max = parse(row.max)
  if (min === 'invalid' || max === 'invalid') return 'Enter a number'
  if (row.enabled && min === null && max === null) return 'Set a minimum, a maximum, or both'
  for (const v of [min, max]) {
    if (v !== null && (v < meta.sensorMin || v > meta.sensorMax)) {
      return `Must be between ${meta.sensorMin} and ${meta.sensorMax}`
    }
  }
  if (min !== null && max !== null && min >= max) return 'Minimum must be lower than maximum'
  return null
}

export function ThresholdEditor({
  deviceId,
  thresholds,
  onSaved,
}: {
  deviceId: string
  thresholds: Threshold[]
  onSaved: (t: Threshold[]) => void
}) {
  const [rows, setRows] = useState<Row[]>(() => thresholds.map(toRow))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [saved, setSaved] = useState(false)

  const errors = Object.fromEntries(rows.map((r) => [r.metric, validate(r)])) as Record<MetricKey, string | null>
  const hasErrors = Object.values(errors).some(Boolean)
  const dirty = JSON.stringify(rows) !== JSON.stringify(thresholds.map(toRow))

  const update = (metric: MetricKey, patch: Partial<Row>) => {
    setSaved(false)
    setRows((current) => current.map((r) => (r.metric === metric ? { ...r, ...patch } : r)))
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (hasErrors) return
    setSaving(true)
    setError(null)
    try {
      const payload: Threshold[] = rows.map((r) => ({
        metric: r.metric,
        min: parse(r.min) as number | null,
        max: parse(r.max) as number | null,
        enabled: r.enabled,
      }))
      const res = await api.saveThresholds(deviceId, payload)
      setRows(res.thresholds.map(toRow))
      onSaved(res.thresholds)
      setSaved(true)
    } catch (err) {
      setError(err)
    } finally {
      setSaving(false)
    }
  }

  const serverField = error instanceof ApiError ? error.field : undefined

  return (
    <Card>
      <CardHeader eyebrow="Alerts" title="Alert thresholds" />
      <form onSubmit={submit} className="p-5" noValidate>
        <p className="mb-4 text-sm text-muted">
          An alert opens when a reading goes outside these limits and resolves once it is comfortably back inside.
          Changes apply to the latest reading immediately. Leave a side empty for no limit on it.
        </p>
        {error ? <div className="mb-4"><Notice>{errorMessage(error)}</Notice></div> : null}
        {saved && !dirty ? <div className="mb-4"><Notice tone="success">Thresholds saved.</Notice></div> : null}

        <div className="divide-y divide-line rounded-lg border border-line">
          {METRICS.map((meta) => {
            const row = rows.find((r) => r.metric === meta.key)
            if (!row) return null
            const rowError = errors[meta.key] ?? (serverField === meta.key ? errorMessage(error) : null)
            const inputClass = `h-10 w-full rounded-lg border bg-surface px-3 text-sm tabular-nums outline-none focus:ring-4 disabled:bg-canvas disabled:text-muted ${
              rowError ? 'border-danger focus:ring-danger/15' : 'border-line focus:border-forest focus:ring-forest/15'
            }`
            return (
              <div key={meta.key} className="grid gap-3 p-4 sm:grid-cols-[minmax(160px,1fr)_120px_120px] sm:items-center">
                <label className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    className="size-4 accent-forest"
                    checked={row.enabled}
                    onChange={(e) => update(meta.key, { enabled: e.target.checked })}
                  />
                  <span>
                    <span className="block font-semibold text-ink">{meta.label}</span>
                    <span className="block text-xs text-muted">
                      {meta.unit ? `${meta.unit} · ` : ''}sensor range {meta.sensorMin} to {meta.sensorMax}
                    </span>
                  </span>
                </label>
                <label className="text-xs font-semibold text-muted">
                  Min
                  <input
                    className={`mt-1 ${inputClass}`}
                    inputMode="decimal"
                    value={row.min}
                    disabled={!row.enabled}
                    onChange={(e) => update(meta.key, { min: e.target.value })}
                    aria-invalid={rowError ? true : undefined}
                    placeholder="none"
                  />
                </label>
                <label className="text-xs font-semibold text-muted">
                  Max
                  <input
                    className={`mt-1 ${inputClass}`}
                    inputMode="decimal"
                    value={row.max}
                    disabled={!row.enabled}
                    onChange={(e) => update(meta.key, { max: e.target.value })}
                    aria-invalid={rowError ? true : undefined}
                    placeholder="none"
                  />
                </label>
                {rowError ? <p className="text-sm text-danger sm:col-span-3">{rowError}</p> : null}
              </div>
            )
          })}
        </div>

        <div className="mt-4 flex flex-wrap justify-end gap-2">
          <Button
            variant="secondary"
            disabled={!dirty || saving}
            onClick={() => {
              setRows(thresholds.map(toRow))
              setError(null)
            }}
          >
            Discard changes
          </Button>
          <Button type="submit" loading={saving} disabled={!dirty || hasErrors}>
            Save thresholds
          </Button>
        </div>
      </form>
    </Card>
  )
}
