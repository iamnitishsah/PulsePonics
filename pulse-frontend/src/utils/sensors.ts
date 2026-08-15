import type { Reading, SensorConfig, SensorKey } from '../types/reading'

export const SENSOR_CONFIGS: SensorConfig[] = [
  {
    key: 'ph',
    label: 'pH',
    shortLabel: 'pH',
    unit: '',
    precision: 2,
    range: [0, 14],
    ideal: [5.5, 6.5],
  },
  {
    key: 'ec_ms_cm',
    label: 'EC',
    shortLabel: 'EC',
    unit: 'mS/cm',
    precision: 2,
    range: [0, 4],
    ideal: [1.2, 2.4],
  },
  {
    key: 'water_temp_c',
    label: 'Water temp',
    shortLabel: 'Water',
    unit: 'C',
    precision: 1,
    range: [10, 35],
    ideal: [18, 24],
  },
  {
    key: 'air_temp_c',
    label: 'Air temp',
    shortLabel: 'Air',
    unit: 'C',
    precision: 1,
    range: [10, 40],
    ideal: [18, 28],
  },
  {
    key: 'humidity_pct',
    label: 'Humidity',
    shortLabel: 'RH',
    unit: '%',
    precision: 0,
    range: [0, 100],
    ideal: [45, 75],
  },
  {
    key: 'pressure_hpa',
    label: 'Pressure',
    shortLabel: 'Pressure',
    unit: 'hPa',
    precision: 0,
    range: [950, 1050],
  },
]

export function getSensorConfig(key: SensorKey) {
  return SENSOR_CONFIGS.find((sensor) => sensor.key === key)!
}

export function getSensorValue(reading: Reading | null | undefined, key: SensorKey) {
  const value = reading?.[key]
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

export function getSensorState(value: number | null, config: SensorConfig) {
  if (value === null) return 'No data'
  if (!config.ideal) return 'Tracking'
  if (value < config.ideal[0]) return 'Low'
  if (value > config.ideal[1]) return 'High'
  return 'Optimal'
}

export function formatSensorValue(value: number | null, config: SensorConfig) {
  if (value === null) return '—'
  const formatted = value.toFixed(config.precision)
  return config.unit ? `${formatted} ${config.unit}` : formatted
}
