import type {
  Alert,
  Device,
  DeviceWithKey,
  History,
  Reading,
  Session,
  Threshold,
  User,
} from '../types/api'
import { apiUrl } from './config'
import { getAccessToken, refreshSession } from './session'

export class ApiError extends Error {
  status: number
  field?: string

  constructor(status: number, message: string, field?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.field = field
  }
}

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
type Options = { auth?: boolean; signal?: AbortSignal }

const SESSION_EXPIRED = 'Your session has expired. Please sign in again.'

async function send(method: Method, path: string, body: unknown, token: string | null, signal?: AbortSignal) {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`
  try {
    return await fetch(apiUrl(path), {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'Cannot reach the PulsePonics server. Check your connection and try again.')
  }
}

async function request<T>(method: Method, path: string, body?: unknown, { auth = true, signal }: Options = {}) {
  let token: string | null = null
  if (auth) {
    token = await getAccessToken().catch(() => {
      throw new ApiError(0, 'Cannot reach the PulsePonics server. Check your connection and try again.')
    })
    if (!token) throw new ApiError(401, SESSION_EXPIRED)
  }

  let res = await send(method, path, body, token, signal)

  // The access token can expire between the check above and the request
  // (or be rejected after a server restart with a new secret): refresh once.
  if (auth && res.status === 401) {
    const session = await refreshSession().catch(() => null)
    if (!session) throw new ApiError(401, SESSION_EXPIRED)
    res = await send(method, path, body, session.access_token, signal)
  }

  if (res.status === 204) return undefined as T

  const data = (await res.json().catch(() => null)) as { error?: string; field?: string } | null
  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? `Request failed (${res.status})`, data?.field)
  }
  return data as T
}

const enc = encodeURIComponent

export type HistoryParams = { from: Date; to?: Date; maxPoints?: number }
export type AlertParams = { deviceId?: string; status?: 'open' | 'resolved' | 'all'; limit?: number }

export const api = {
  // Auth
  signup: (input: { email: string; password: string; name?: string }) =>
    request<Session>('POST', '/api/auth/signup', input, { auth: false }),
  login: (input: { email: string; password: string }) =>
    request<Session>('POST', '/api/auth/login', input, { auth: false }),
  logout: (refreshToken: string) =>
    request<void>('POST', '/api/auth/logout', { refresh_token: refreshToken }, { auth: false }),
  me: (signal?: AbortSignal) => request<User>('GET', '/api/me', undefined, { signal }),
  changePassword: (input: { current_password: string; new_password: string }) =>
    request<Session>('POST', '/api/me/password', input),
  deleteAccount: (password: string) => request<void>('DELETE', '/api/me', { password }),

  // Devices
  devices: (signal?: AbortSignal) => request<Device[]>('GET', '/api/devices', undefined, { signal }),
  device: (deviceId: string, signal?: AbortSignal) =>
    request<Device>('GET', `/api/devices/${enc(deviceId)}`, undefined, { signal }),
  createDevice: (input: { device_id: string; name?: string }) =>
    request<DeviceWithKey>('POST', '/api/devices', input),
  renameDevice: (deviceId: string, name: string) =>
    request<Device>('PATCH', `/api/devices/${enc(deviceId)}`, { name }),
  deleteDevice: (deviceId: string) => request<void>('DELETE', `/api/devices/${enc(deviceId)}`),
  rotateDeviceKey: (deviceId: string) =>
    request<DeviceWithKey>('POST', `/api/devices/${enc(deviceId)}/api-key`),

  // Readings
  history: (deviceId: string, { from, to, maxPoints }: HistoryParams, signal?: AbortSignal) => {
    const q = new URLSearchParams({ from: from.toISOString() })
    if (to) q.set('to', to.toISOString())
    if (maxPoints) q.set('max_points', String(maxPoints))
    return request<History>('GET', `/api/devices/${enc(deviceId)}/readings?${q}`, undefined, { signal })
  },
  recentReadings: (deviceId: string, limit = 10, signal?: AbortSignal) =>
    request<Reading[]>('GET', `/api/devices/${enc(deviceId)}/readings/recent?limit=${limit}`, undefined, {
      signal,
    }),

  // Thresholds and alerts
  thresholds: (deviceId: string, signal?: AbortSignal) =>
    request<{ thresholds: Threshold[] }>('GET', `/api/devices/${enc(deviceId)}/thresholds`, undefined, {
      signal,
    }),
  saveThresholds: (deviceId: string, thresholds: Threshold[]) =>
    request<{ thresholds: Threshold[] }>('PUT', `/api/devices/${enc(deviceId)}/thresholds`, { thresholds }),
  alerts: ({ deviceId, status = 'all', limit }: AlertParams, signal?: AbortSignal) => {
    const q = new URLSearchParams({ status })
    if (deviceId) q.set('device_id', deviceId)
    if (limit) q.set('limit', String(limit))
    return request<Alert[]>('GET', `/api/alerts?${q}`, undefined, { signal })
  },
  acknowledgeAlert: (id: number) => request<Alert>('POST', `/api/alerts/${id}/acknowledge`),
}

export function errorMessage(err: unknown) {
  if (err instanceof Error) return err.message
  return 'Something went wrong'
}
