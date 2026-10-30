import { useContext, useEffect, useRef } from 'react'
import type { RealtimeEvent } from '../types/api'
import { RealtimeContext } from './context'

export function useRealtime() {
  const ctx = useContext(RealtimeContext)
  if (!ctx) throw new Error('useRealtime must be used inside <RealtimeProvider>')
  return ctx
}

/** Calls `handler` for every realtime event. The latest handler is always
 *  used, so it can read fresh props/state without resubscribing. */
export function useRealtimeEvents(handler: (event: RealtimeEvent) => void) {
  const { subscribe } = useRealtime()
  const latest = useRef(handler)
  useEffect(() => {
    latest.current = handler
  })
  useEffect(() => subscribe((event) => latest.current(event)), [subscribe])
}

/** Runs `callback` each time the socket comes back after a drop, so a page
 *  can refetch whatever was published while it was disconnected. */
export function useOnReconnect(callback: () => void) {
  const { reconnects } = useRealtime()
  const latest = useRef(callback)
  const seen = useRef(reconnects)
  useEffect(() => {
    latest.current = callback
  })
  useEffect(() => {
    if (reconnects !== seen.current) {
      seen.current = reconnects
      latest.current()
    }
  }, [reconnects])
}
