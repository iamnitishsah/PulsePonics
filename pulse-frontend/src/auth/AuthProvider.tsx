import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api } from '../lib/api'
import {
  REFRESH_STORAGE_KEY,
  clearSession,
  onSessionChange,
  readRefreshToken,
  refreshSession,
  setSession,
} from '../lib/session'
import type { Session, User } from '../types/api'
import { AuthContext, type AuthStatus } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>(() => (readRefreshToken() ? 'loading' : 'anonymous'))
  const [user, setUser] = useState<User | null>(null)
  const [attempt, setAttempt] = useState(0)

  // Mirror token-store changes (refresh failures, other tabs) into React state.
  useEffect(
    () =>
      onSessionChange((next) => {
        setUser(next)
        setStatus(next ? 'authenticated' : 'anonymous')
      }),
    [],
  )

  // Restore the session from the stored refresh token on load.
  useEffect(() => {
    if (!readRefreshToken()) return
    refreshSession().catch(() => setStatus('unreachable'))
  }, [attempt])

  // Signing out in one tab signs out the others.
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === REFRESH_STORAGE_KEY && e.newValue === null) clearSession({ keepStorage: true })
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    setSession(await api.login({ email, password }))
  }, [])

  const signup = useCallback(async (email: string, password: string, name: string) => {
    setSession(await api.signup({ email, password, name }))
  }, [])

  const logout = useCallback(async () => {
    const token = readRefreshToken()
    clearSession()
    if (token) await api.logout(token).catch(() => undefined)
  }, [])

  const adoptSession = useCallback((session: Session) => setSession(session), [])
  const forget = useCallback(() => clearSession(), [])

  const retry = useCallback(() => {
    setStatus('loading')
    setAttempt((n) => n + 1)
  }, [])

  const value = useMemo(
    () => ({ status, user, login, signup, logout, adoptSession, forget, retry }),
    [status, user, login, signup, logout, adoptSession, forget, retry],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
