// Backend origin without a trailing slash. Empty means "same origin", which is
// what the Vite dev server proxy expects (see vite.config.ts).
export const API_URL = (import.meta.env.VITE_API_URL ?? '').trim().replace(/\/+$/, '')

export function apiUrl(path: string) {
  return `${API_URL}${path}`
}

// Browsers can't set headers on WebSocket requests, so the short-lived access
// token goes in the query string.
export function webSocketUrl(token: string) {
  const url = new URL('/api/ws', API_URL || window.location.origin)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('token', token)
  return url.toString()
}

// A device counts as live if its latest reading is newer than this. The
// firmware publishes every 10 s in development and every 60 s when deployed.
export const STALE_AFTER_MS = 3 * 60 * 1000
