import { createContext } from 'react'
import type { Session, User } from '../types/api'

export type AuthStatus = 'loading' | 'authenticated' | 'anonymous' | 'unreachable'

export type AuthContextValue = {
  status: AuthStatus
  user: User | null
  login: (email: string, password: string) => Promise<void>
  signup: (email: string, password: string, name: string) => Promise<void>
  logout: () => Promise<void>
  /** Install a session returned by another call (e.g. password change). */
  adoptSession: (session: Session) => void
  /** Forget the session locally (e.g. after deleting the account). */
  forget: () => void
  retry: () => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)
