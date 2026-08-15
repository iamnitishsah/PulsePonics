import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { getHistory, getLatest, getWebSocketUrl } from '../api/readings'
import type { Reading } from '../types/reading'

type ConnectionState = 'connecting' | 'live' | 'offline'

const MAX_POINTS = 120

function sortAscending(readings: Reading[]) {
  return [...readings].sort(
    (a, b) => new Date(a.recorded_at).getTime() - new Date(b.recorded_at).getTime(),
  )
}

function mergeReading(readings: Reading[], next: Reading) {
  const deduped = readings.filter((reading) => reading.id !== next.id)
  return sortAscending([...deduped, next]).slice(-MAX_POINTS)
}

export function useTelemetry(deviceId: string) {
  const [history, setHistory] = useState<Reading[]>([])
  const [latest, setLatest] = useState<Reading | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting')
  const reconnectTimer = useRef<number | null>(null)

  const loadReadings = useCallback(async () => {
    if (!deviceId.trim()) return

    setLoading(true)
    setError(null)

    try {
      const [historyResult, latestResult] = await Promise.allSettled([
        getHistory({ deviceId, limit: MAX_POINTS }),
        getLatest(deviceId),
      ])

      if (historyResult.status === 'fulfilled') {
        setHistory(sortAscending(historyResult.value))
      } else {
        throw historyResult.reason
      }

      if (latestResult.status === 'fulfilled') {
        setLatest(latestResult.value)
      } else {
        setLatest(null)
      }
    } catch (err) {
      setHistory([])
      setLatest(null)
      setError(err instanceof Error ? err.message : 'Unable to load telemetry')
    } finally {
      setLoading(false)
    }
  }, [deviceId])

  useEffect(() => {
    void Promise.resolve().then(loadReadings)
  }, [loadReadings])

  useEffect(() => {
    if (!deviceId.trim()) return

    let socket: WebSocket | null = null
    let cancelled = false

    const connect = () => {
      setConnectionState('connecting')
      socket = new WebSocket(getWebSocketUrl())

      socket.onopen = () => {
        if (!cancelled) setConnectionState('live')
      }

      socket.onmessage = (event) => {
        try {
          const reading = JSON.parse(event.data) as Reading
          if (reading.device_id !== deviceId) return

          setLatest(reading)
          setHistory((current) => mergeReading(current, reading))
        } catch {
          setError('Received an invalid realtime payload')
        }
      }

      socket.onclose = () => {
        if (cancelled) return
        setConnectionState('offline')
        reconnectTimer.current = window.setTimeout(connect, 3000)
      }

      socket.onerror = () => {
        setConnectionState('offline')
      }
    }

    connect()

    return () => {
      cancelled = true
      if (reconnectTimer.current) window.clearTimeout(reconnectTimer.current)
      socket?.close()
    }
  }, [deviceId])

  const stats = useMemo(
    () => ({
      sampleCount: history.length,
      lastUpdated: latest?.recorded_at ?? null,
    }),
    [history.length, latest?.recorded_at],
  )

  return {
    history,
    latest,
    loading,
    error,
    connectionState,
    refresh: loadReadings,
    stats,
  }
}
