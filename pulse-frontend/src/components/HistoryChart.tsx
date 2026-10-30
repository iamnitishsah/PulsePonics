import { useCallback, useMemo, useRef, useState } from 'react'
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceArea,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { useQuery } from '../hooks/useQuery'
import { api, errorMessage } from '../lib/api'
import { toChartSeries, yDomain } from '../lib/chart'
import { formatAxisTime, formatDateTime } from '../lib/format'
import { METRICS, METRIC_BY_KEY, formatValue } from '../lib/metrics'
import { useOnReconnect, useRealtimeEvents } from '../realtime/useRealtime'
import type { MetricKey, Point, Threshold } from '../types/api'
import { Card } from './ui/Card'
import { Notice } from './ui/Notice'
import { Spinner } from './ui/Spinner'

const HOUR = 3600_000
const RANGES = [
  { key: '1h', label: '1H', ms: HOUR },
  { key: '6h', label: '6H', ms: 6 * HOUR },
  { key: '24h', label: '24H', ms: 24 * HOUR },
  { key: '7d', label: '7D', ms: 7 * 24 * HOUR },
  { key: '30d', label: '30D', ms: 30 * 24 * HOUR },
] as const
type RangeKey = (typeof RANGES)[number]['key']

const COLORS = { line: '#2f6b57', band: '#8ed4b8', limit: '#8a5a11', grid: 'rgb(23 33 29 / 0.08)', tick: '#6b7b74' }

function Segmented<T extends string>({
  label,
  options,
  value,
  onChange,
}: {
  label: string
  options: { key: T; label: string }[]
  value: T
  onChange: (key: T) => void
}) {
  return (
    <div role="group" aria-label={label} className="inline-flex max-w-full overflow-x-auto rounded-lg border border-line bg-canvas p-1">
      {options.map((o) => (
        <button
          key={o.key}
          type="button"
          aria-pressed={o.key === value}
          onClick={() => onChange(o.key)}
          className={`shrink-0 rounded-md px-3 py-1.5 text-sm font-semibold transition-colors ${
            o.key === value ? 'bg-ink text-white' : 'text-ink-soft hover:bg-surface'
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}

export function HistoryChart({
  deviceId,
  metric,
  onMetricChange,
  thresholds,
}: {
  deviceId: string
  metric: MetricKey
  onMetricChange: (m: MetricKey) => void
  thresholds: Threshold[] | undefined
}) {
  const [range, setRange] = useState<RangeKey>('1h')
  const rangeMs = RANGES.find((r) => r.key === range)!.ms
  const meta = METRIC_BY_KEY[metric]
  const threshold = thresholds?.find((t) => t.metric === metric)

  const fetchHistory = useCallback(
    (signal: AbortSignal) => {
      const to = new Date()
      return api.history(deviceId, { from: new Date(to.getTime() - rangeMs), to, maxPoints: 500 }, signal)
    },
    [deviceId, rangeMs],
  )
  const { data, error, loading, reload, mutate } = useQuery(fetchHistory)
  const lastReload = useRef(0)

  // Raw ranges get each new reading appended live. Aggregated ranges are
  // refetched, at most once per bucket, since a new reading only moves an average.
  useRealtimeEvents((event) => {
    if (event.type !== 'reading' || event.data.device_id !== deviceId || !data) return
    if (data.bucket_seconds === 0) {
      const r = event.data
      const point: Point = {
        recorded_at: r.recorded_at,
        ph: r.ph,
        ec_ms_cm: r.ec_ms_cm,
        water_temp_c: r.water_temp_c,
        air_temp_c: r.air_temp_c,
        humidity_pct: r.humidity_pct,
        pressure_hpa: r.pressure_hpa,
      }
      const cutoff = Date.now() - rangeMs
      mutate((h) =>
        h
          ? {
              ...h,
              points: [...h.points.filter((p) => p.recorded_at !== point.recorded_at && Date.parse(p.recorded_at) >= cutoff), point],
            }
          : h,
      )
    } else if (Date.now() - lastReload.current > data.bucket_seconds * 1000) {
      lastReload.current = Date.now()
      reload()
    }
  })
  useOnReconnect(reload)

  const series = useMemo(
    () => (data ? toChartSeries(data.points, metric, data.bucket_seconds) : []),
    [data, metric],
  )
  const values = series.flatMap((p) => (p.v === null ? [] : [p.v]))
  const band = threshold?.enabled ? { min: threshold.min, max: threshold.max } : null
  const domain = yDomain(values, band)

  // Live appends can run past the fetch's `to`, so extend the axis to the newest point.
  const fromMs = data ? Date.parse(data.from) : 0
  const toMs = Math.max(data ? Date.parse(data.to) : 0, series.at(-1)?.t ?? 0)

  const stats =
    values.length > 0
      ? {
          min: Math.min(...values),
          max: Math.max(...values),
          avg: values.reduce((a, b) => a + b, 0) / values.length,
        }
      : null

  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 pt-5">
        <div>
          <p className="text-xs font-bold uppercase tracking-wide text-forest">History</p>
          <h2 className="text-lg font-bold">{meta.label}</h2>
        </div>
        <Segmented label="Time range" options={[...RANGES]} value={range} onChange={setRange} />
      </div>
      <div className="px-5 pt-3">
        <Segmented
          label="Metric"
          options={METRICS.map((m) => ({ key: m.key, label: m.short }))}
          value={metric}
          onChange={onMetricChange}
        />
      </div>

      <div className="relative px-2 pb-2 pt-4 sm:px-4">
        {error ? (
          <div className="px-3 pb-4">
            <Notice>{errorMessage(error)}</Notice>
          </div>
        ) : values.length === 0 && !loading ? (
          <div className="mx-3 mb-4 grid h-[280px] place-items-center rounded-lg border border-dashed border-ink/20 text-center text-sm text-muted">
            <p className="max-w-xs">
              No {meta.label.toLowerCase()} readings in the last {RANGES.find((r) => r.key === range)!.label.toLowerCase()}.
              {' '}The {meta.source} sensor may not be connected yet.
            </p>
          </div>
        ) : (
          <div className="h-[280px] sm:h-[320px]" aria-label={`${meta.label} chart`} role="img">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={series} margin={{ top: 8, right: 16, bottom: 4, left: 0 }}>
                <CartesianGrid stroke={COLORS.grid} vertical={false} />
                <XAxis
                  dataKey="t"
                  type="number"
                  scale="time"
                  domain={[fromMs, toMs]}
                  tickFormatter={(ms: number) => formatAxisTime(ms, rangeMs)}
                  tick={{ fontSize: 12, fill: COLORS.tick }}
                  tickLine={false}
                  axisLine={{ stroke: COLORS.grid }}
                  minTickGap={48}
                />
                <YAxis
                  domain={domain ?? ['auto', 'auto']}
                  width={52}
                  tick={{ fontSize: 12, fill: COLORS.tick }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v: number) => v.toFixed(meta.precision > 1 ? 1 : 0)}
                />
                {band && band.min !== null && band.max !== null ? (
                  <ReferenceArea y1={band.min} y2={band.max} fill={COLORS.band} fillOpacity={0.22} />
                ) : null}
                {band && band.min !== null ? <ReferenceLine y={band.min} stroke={COLORS.limit} strokeDasharray="4 4" /> : null}
                {band && band.max !== null ? <ReferenceLine y={band.max} stroke={COLORS.limit} strokeDasharray="4 4" /> : null}
                <Tooltip
                  labelFormatter={(ms) => formatDateTime(Number(ms))}
                  formatter={(v) => [formatValue(Number(v), meta), data?.bucket_seconds ? `${meta.label} (avg)` : meta.label]}
                  contentStyle={{ borderRadius: 8, border: '1px solid rgb(23 33 29 / 0.1)', fontSize: 13 }}
                />
                <Line
                  type="monotone"
                  dataKey="v"
                  stroke={COLORS.line}
                  strokeWidth={2}
                  dot={values.length <= 40 ? { r: 2.5, fill: COLORS.line } : false}
                  activeDot={{ r: 4 }}
                  connectNulls={false}
                  isAnimationActive={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
        {loading ? (
          <div className="absolute right-6 top-6 text-muted" aria-label="Loading">
            <Spinner className="size-4" />
          </div>
        ) : null}
      </div>

      <div className="flex flex-wrap gap-x-6 gap-y-1 border-t border-line px-5 py-3 text-xs text-muted">
        {stats ? (
          <>
            <span>
              Min <strong className="text-ink">{formatValue(stats.min, meta)}</strong>
            </span>
            <span>
              Avg <strong className="text-ink">{formatValue(stats.avg, meta)}</strong>
            </span>
            <span>
              Max <strong className="text-ink">{formatValue(stats.max, meta)}</strong>
            </span>
          </>
        ) : null}
        <span className="ml-auto">
          {data?.bucket_seconds
            ? `${data.points.length} points · averaged per ${Math.round(data.bucket_seconds / 60) || 1} min`
            : `${data?.points.length ?? 0} readings · live`}
        </span>
      </div>
    </Card>
  )
}
