import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { webSocketUrl } from '../lib/config'
import { getAccessToken } from '../lib/session'
import type { RealtimeEvent } from '../types/api'
import { RealtimeContext, type RealtimeStatus } from './context'

const MAX_BACKOFF_MS = 30_000

/** One WebSocket per signed-in tab, shared by every page. */
export function RealtimeProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<RealtimeStatus>('connecting')
  const [reconnects, setReconnects] = useState(0)
  const handlers = useRef(new Set<(event: RealtimeEvent) => void>())

  useEffect(() => {
    let socket: WebSocket | null = null
    let timer: number | undefined
    let stopped = false
    let attempt = 0
    let openedBefore = false

    const scheduleReconnect = () => {
      if (stopped) return
      setStatus('offline')
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt)
      attempt += 1
      timer = window.setTimeout(() => {
        setStatus('connecting')
        void connect()
      }, delay)
    }

    const connect = async () => {
      let token: string | null
      try {
        token = await getAccessToken()
      } catch {
        scheduleReconnect()
        return
      }
      if (stopped) return
      if (!token) {
        setStatus('offline') // session ended; the auth layer redirects to login
        return
      }

      socket = new WebSocket(webSocketUrl(token))
      socket.onopen = () => {
        attempt = 0
        setStatus('live')
        if (openedBefore) setReconnects((n) => n + 1)
        openedBefore = true
      }
      socket.onmessage = (message) => {
        let event: RealtimeEvent
        try {
          event = JSON.parse(message.data as string) as RealtimeEvent
        } catch {
          return
        }
        handlers.current.forEach((handler) => handler(event))
      }
      socket.onclose = scheduleReconnect
    }

    void connect()

    return () => {
      stopped = true
      window.clearTimeout(timer)
      if (socket) {
        socket.onclose = null
        socket.close()
      }
    }
  }, [])

  const subscribe = useCallback((handler: (event: RealtimeEvent) => void) => {
    handlers.current.add(handler)
    return () => {
      handlers.current.delete(handler)
    }
  }, [])

  const value = useMemo(() => ({ status, reconnects, subscribe }), [status, reconnects, subscribe])
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>
}
