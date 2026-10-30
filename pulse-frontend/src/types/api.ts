// Shapes returned by pulse-backend (see pulse-backend/API_DOCUMENTATION.md).

export type MetricKey =
  | 'ph'
  | 'ec_ms_cm'
  | 'water_temp_c'
  | 'air_temp_c'
  | 'humidity_pct'
  | 'pressure_hpa'

export type MetricValues = Record<MetricKey, number | null>

export type User = {
  id: number
  email: string
  name: string
  created_at: string
}

export type Session = {
  access_token: string
  access_token_expires_at: string
  refresh_token: string
  refresh_token_expires_at: string
  user: User
}

export type Reading = MetricValues & {
  id: number
  device_id: string
  recorded_at: string
  created_at: string
}

export type Point = MetricValues & {
  recorded_at: string
}

export type History = {
  device_id: string
  from: string
  to: string
  /** 0 = raw readings, otherwise each point averages this many seconds. */
  bucket_seconds: number
  points: Point[]
}

export type Device = {
  device_id: string
  name: string
  api_key_prefix: string
  created_at: string
  latest: Reading | null
  open_alerts: number
}

export type DeviceWithKey = Omit<Device, 'latest' | 'open_alerts'> & {
  api_key: string
}

export type Threshold = {
  metric: MetricKey
  min: number | null
  max: number | null
  enabled: boolean
}

export type Alert = {
  id: number
  device_id: string
  metric: MetricKey
  kind: 'low' | 'high'
  threshold_value: number
  trigger_value: number
  latest_value: number
  opened_at: string
  resolved_at: string | null
  acknowledged_at: string | null
  status: 'open' | 'resolved'
}

export type RealtimeEvent =
  | { type: 'reading'; data: Reading }
  | { type: 'alert'; data: Alert }
