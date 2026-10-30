import { describe, expect, it } from 'vitest'
import type { Point, Threshold } from '../types/api'
import { toChartSeries, yDomain } from './chart'
import { isFresh, timeAgo } from './format'
import { formatBand, formatValue, METRIC_BY_KEY, metricState } from './metrics'

const ph: Threshold = { metric: 'ph', min: 5.5, max: 6.5, enabled: true }

function point(iso: string, value: number | null): Point {
  return {
    recorded_at: iso,
    ph: value,
    ec_ms_cm: null,
    water_temp_c: null,
    air_temp_c: null,
    humidity_pct: null,
    pressure_hpa: null,
  }
}

describe('metricState', () => {
  it('classifies against the threshold band', () => {
    expect(metricState(6, ph)).toBe('ok')
    expect(metricState(5.4, ph)).toBe('low')
    expect(metricState(6.6, ph)).toBe('high')
    expect(metricState(null, ph)).toBe('nodata')
    expect(metricState(9, { ...ph, enabled: false })).toBe('off')
    expect(metricState(30, { metric: 'water_temp_c', min: null, max: 24, enabled: true })).toBe('high')
  })
})

describe('formatting', () => {
  it('formats values with precision and unit', () => {
    expect(formatValue(28.437, METRIC_BY_KEY.water_temp_c)).toBe('28.4 °C')
    expect(formatValue(6.2, METRIC_BY_KEY.ph)).toBe('6.20')
    expect(formatValue(null, METRIC_BY_KEY.ph)).toBe('—')
  })

  it('describes threshold bands', () => {
    expect(formatBand(ph, METRIC_BY_KEY.ph)).toBe('Target 5.5–6.5')
    expect(formatBand({ ...ph, min: null }, METRIC_BY_KEY.ph)).toBe('Alert above 6.5')
    expect(formatBand({ ...ph, enabled: false }, METRIC_BY_KEY.ph)).toBe('No alert limits')
  })

  it('describes elapsed time', () => {
    const now = Date.parse('2026-10-24T10:00:00Z')
    expect(timeAgo('2026-10-24T09:59:55Z', now)).toBe('just now')
    expect(timeAgo('2026-10-24T09:58:00Z', now)).toBe('2m ago')
    expect(timeAgo('2026-10-24T07:00:00Z', now)).toBe('3h ago')
    expect(timeAgo(null, now)).toBe('never')
    expect(isFresh('2026-10-24T09:58:00Z', 180_000, now)).toBe(true)
    expect(isFresh('2026-10-24T09:50:00Z', 180_000, now)).toBe(false)
  })
})

describe('toChartSeries', () => {
  it('breaks the line across an outage', () => {
    const series = toChartSeries(
      [
        point('2026-10-24T10:00:00Z', 6),
        point('2026-10-24T10:00:10Z', 6.1),
        point('2026-10-24T10:00:20Z', 6.2),
        point('2026-10-24T10:30:00Z', 6.3), // 30 min silence
        point('2026-10-24T10:30:10Z', 6.2),
      ],
      'ph',
    )
    expect(series).toHaveLength(6)
    expect(series[3].v).toBeNull()
  })

  it('keeps evenly spaced data intact', () => {
    const series = toChartSeries(
      [point('2026-10-24T10:00:00Z', 6), point('2026-10-24T10:00:10Z', 6.1), point('2026-10-24T10:00:20Z', 6.2)],
      'ph',
    )
    expect(series.every((p) => p.v !== null)).toBe(true)
  })
})

describe('yDomain', () => {
  it('includes the threshold band', () => {
    const [lo, hi] = yDomain([6.0, 6.1], { min: 5.5, max: 6.5 })!
    expect(lo).toBeLessThan(5.5)
    expect(hi).toBeGreaterThan(6.5)
  })
  it('returns null without data', () => {
    expect(yDomain([], null)).toBeNull()
  })
})
