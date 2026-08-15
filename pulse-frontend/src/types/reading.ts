export type Reading = {
  id: number
  device_id: string
  ph?: number | null
  ec_ms_cm?: number | null
  water_temp_c?: number | null
  air_temp_c?: number | null
  humidity_pct?: number | null
  pressure_hpa?: number | null
  recorded_at: string
  created_at: string
}

export type SensorKey =
  | 'ph'
  | 'ec_ms_cm'
  | 'water_temp_c'
  | 'air_temp_c'
  | 'humidity_pct'
  | 'pressure_hpa'

export type SensorConfig = {
  key: SensorKey
  label: string
  shortLabel: string
  unit: string
  precision: number
  range: [number, number]
  ideal?: [number, number]
}
