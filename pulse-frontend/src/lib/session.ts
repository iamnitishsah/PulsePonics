// Token storage and refresh.
//
// - The access token (15 min) lives only in memory.
// - The refresh token (7 days) lives in localStorage so a reload keeps you
//   signed in. The frontend (Vercel) and API (EC2) are on different sites,
//   where third-party cookies are unreliable, which rules out an httpOnly cookie.
// - The backend rotates refresh tokens and treats reuse of a rotated token as
//   theft (revoking every session). Tabs share localStorage, so refreshes are
//   serialised across tabs with the Web Locks API, and the token is always
//   re-read from storage inside the lock.

import type { Session, User } from '../types/api'
import { apiUrl } from './config'

const REFRESH_KEY = 'pulseponics.refresh_token'
const EXPIRY_MARGIN_MS = 30_000

type Listener = (user: User | null) => void

let access: { token: string; expiresAt: number } | null = null
let inflight: Promise<Session | null> | null = null
const listeners = new Set<Listener>()

export function readRefreshToken(): string | null {
  try {
    return localStorage.getItem(REFRESH_KEY)
  } catch {
    return null
  }
}

function writeRefreshToken(token: string | null) {
  try {
    if (token) localStorage.setItem(REFRESH_KEY, token)
    else localStorage.removeItem(REFRESH_KEY)
  } catch {
    // Storage blocked (private mode): the session lasts until the tab closes.
  }
}

export function onSessionChange(listener: Listener) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

function emit(user: User | null) {
  listeners.forEach((listener) => listener(user))
}

export function setSession(session: Session) {
  access = { token: session.access_token, expiresAt: Date.parse(session.access_token_expires_at) }
  writeRefreshToken(session.refresh_token)
  emit(session.user)
}

export function clearSession({ keepStorage = false } = {}) {
  access = null
  if (!keepStorage) writeRefreshToken(null)
  emit(null)
}

export const REFRESH_STORAGE_KEY = REFRESH_KEY

function withRefreshLock<T>(fn: () => Promise<T>): Promise<T> {
  if (typeof navigator !== 'undefined' && navigator.locks) {
    return navigator.locks.request('pulseponics-refresh', fn)
  }
  return fn()
}

/**
 * Exchanges the stored refresh token for a new session. Concurrent callers
 * share one request. Resolves to null when the session is gone (401) and
 * throws when the server can't be reached, so callers can tell the two apart.
 */
export function refreshSession(): Promise<Session | null> {
  if (!inflight) {
    inflight = withRefreshLock(async () => {
      const token = readRefreshToken()
      if (!token) {
        clearSession()
        return null
      }
      const res = await fetch(apiUrl('/api/auth/refresh'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ refresh_token: token }),
      })
      if (res.status === 401) {
        // Another tab may have rotated the token after we read it: only sign
        // out if storage still holds the token that was rejected.
        if (readRefreshToken() === token) clearSession()
        return null
      }
      if (!res.ok) throw new Error(`Session refresh failed (${res.status})`)
      const session = (await res.json()) as Session
      setSession(session)
      return session
    }).finally(() => {
      inflight = null
    })
  }
  return inflight
}

/** A valid access token, refreshing first if it expires within 30 s. */
export async function getAccessToken(): Promise<string | null> {
  if (access && access.expiresAt - Date.now() > EXPIRY_MARGIN_MS) return access.token
  const session = await refreshSession()
  return session?.access_token ?? null
}
