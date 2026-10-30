import type { MetricKey, MetricValues, Threshold } from '../types/api'

export type MetricMeta = {
  key: MetricKey
  label: string
  short: string
  unit: string
  precision: number
  /** Range the backend accepts (pulse-hardware include/config.h). */
  sensorMin: number
  sensorMax: number
  source: string
}

export const METRICS: MetricMeta[] = [
  { key: 'ph', label: 'pH', short: 'pH', unit: '', precision: 2, sensorMin: 0, sensorMax: 14, source: 'pH probe · ADS1115 A0' },
  { key: 'ec_ms_cm', label: 'EC', short: 'EC', unit: 'mS/cm', precision: 2, sensorMin: 0, sensorMax: 20, source: 'EC probe · ADS1115 A1' },
  { key: 'water_temp_c', label: 'Water temp', short: 'Water', unit: '°C', precision: 1, sensorMin: -10, sensorMax: 60, source: 'DS18B20' },
  { key: 'air_temp_c', label: 'Air temp', short: 'Air', unit: '°C', precision: 1, sensorMin: -20, sensorMax: 60, source: 'BME280' },
  { key: 'humidity_pct', label: 'Humidity', short: 'RH', unit: '%', precision: 0, sensorMin: 0, sensorMax: 100, source: 'BME280' },
  { key: 'pressure_hpa', label: 'Pressure', short: 'Pressure', unit: 'hPa', precision: 0, sensorMin: 300, sensorMax: 1100, source: 'BME280' },
]

export const METRIC_BY_KEY = Object.fromEntries(METRICS.map((m) => [m.key, m])) as Record<MetricKey, MetricMeta>

export type MetricState = 'nodata' | 'off' | 'ok' | 'low' | 'high'

export function metricState(value: number | null | undefined, threshold?: Threshold): MetricState {
  if (value === null || value === undefined || !Number.isFinite(value)) return 'nodata'
  if (!threshold || !threshold.enabled) return 'off'
  if (threshold.min !== null && value < threshold.min) return 'low'
  if (threshold.max !== null && value > threshold.max) return 'high'
  return 'ok'
}

export function formatValue(value: number | null | undefined, meta: MetricMeta, withUnit = true) {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—'
  const text = value.toFixed(meta.precision)
  return withUnit && meta.unit ? `${text} ${meta.unit}` : text
}

export function formatBand(threshold: Threshold | undefined, meta: MetricMeta) {
  if (!threshold || !threshold.enabled) return 'No alert limits'
  const unit = meta.unit ? ` ${meta.unit}` : ''
  const { min, max } = threshold
  if (min !== null && max !== null) return `Target ${min}–${max}${unit}`
  if (min !== null) return `Alert below ${min}${unit}`
  if (max !== null) return `Alert above ${max}${unit}`
  return 'No alert limits'
}

export function valueOf(values: MetricValues | null | undefined, key: MetricKey) {
  const v = values?.[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : null
}
