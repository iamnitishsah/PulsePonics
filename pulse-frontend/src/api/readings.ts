import type { Reading } from '../types/reading'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

type HistoryParams = {
  deviceId: string
  limit?: number
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      Accept: 'application/json',
      ...init?.headers,
    },
    ...init,
  })

  if (!response.ok) {
    let message = `Request failed with ${response.status}`
    try {
      const payload = (await response.json()) as { error?: string }
      if (payload.error) message = payload.error
    } catch {
      // Keep the HTTP status fallback.
    }
    throw new Error(message)
  }

  return response.json() as Promise<T>
}

export function getHistory({ deviceId, limit = 100 }: HistoryParams) {
  const params = new URLSearchParams({
    device_id: deviceId,
    limit: String(limit),
  })

  return request<Reading[]>(`/readings?${params.toString()}`)
}

export function getLatest(deviceId: string) {
  const params = new URLSearchParams({ device_id: deviceId })
  return request<Reading>(`/readings/latest?${params.toString()}`)
}

export function getWebSocketUrl() {
  const configuredUrl = import.meta.env.VITE_WS_URL as string | undefined
  if (configuredUrl) return configuredUrl

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}
