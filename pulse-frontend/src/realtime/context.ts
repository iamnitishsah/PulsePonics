import { createContext } from 'react'
import type { RealtimeEvent } from '../types/api'

export type RealtimeStatus = 'connecting' | 'live' | 'offline'

export type RealtimeContextValue = {
  status: RealtimeStatus
  /** Increments every time the socket reconnects after a drop. Pages refetch
   *  when it changes, to fill in anything sent while disconnected. */
  reconnects: number
  subscribe: (handler: (event: RealtimeEvent) => void) => () => void
}

export const RealtimeContext = createContext<RealtimeContextValue | null>(null)
