import type { MetricKey, Point } from '../types/api'

export type ChartPoint = { t: number; v: number | null }

/**
 * Converts API points into chart points for one metric and inserts a null
 * wherever the device went quiet, so the line breaks instead of drawing a
 * straight segment across an outage. A gap is anything longer than three
 * times the usual spacing between samples (or the aggregation bucket).
 */
export function toChartSeries(points: Point[], metric: MetricKey, bucketSeconds = 0): ChartPoint[] {
  const series = points
    .map((p) => ({ t: Date.parse(p.recorded_at), v: p[metric] }))
    .filter((p) => Number.isFinite(p.t))
    .sort((a, b) => a.t - b.t)

  if (series.length < 2) return series

  const spacing = Math.max(medianSpacing(series), bucketSeconds * 1000)
  const gap = spacing * 3

  const out: ChartPoint[] = [series[0]]
  for (let i = 1; i < series.length; i++) {
    if (series[i].t - series[i - 1].t > gap) {
      out.push({ t: series[i - 1].t + 1, v: null })
    }
    out.push(series[i])
  }
  return out
}

function medianSpacing(series: ChartPoint[]) {
  const diffs: number[] = []
  for (let i = 1; i < series.length; i++) diffs.push(series[i].t - series[i - 1].t)
  diffs.sort((a, b) => a - b)
  return diffs[Math.floor(diffs.length / 2)]
}

/** Y-axis domain covering the data and the threshold band, with padding. */
export function yDomain(
  values: number[],
  band: { min: number | null; max: number | null } | null,
): [number, number] | null {
  const all = [...values]
  if (band?.min !== null && band?.min !== undefined) all.push(band.min)
  if (band?.max !== null && band?.max !== undefined) all.push(band.max)
  if (all.length === 0) return null
  const lo = Math.min(...all)
  const hi = Math.max(...all)
  const pad = (hi - lo || Math.abs(hi) || 1) * 0.1
  return [round(lo - pad), round(hi + pad)]
}

function round(n: number) {
  return Math.round(n * 100) / 100
}
