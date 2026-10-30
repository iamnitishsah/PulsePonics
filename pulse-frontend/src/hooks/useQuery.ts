import { useCallback, useEffect, useState } from 'react'

type Result<T> = {
  fetcher: unknown
  nonce: number
  data?: T
  error?: Error
}

/**
 * Minimal data loader. `fetcher` must be stable (wrap it in useCallback);
 * when it changes (e.g. a different device), stale data is dropped and the
 * request is re-run. reload() re-runs it while keeping the current data on
 * screen. Earlier requests are aborted, so out-of-order responses can't win.
 */
export function useQuery<T>(fetcher: (signal: AbortSignal) => Promise<T>) {
  const [nonce, setNonce] = useState(0)
  const [result, setResult] = useState<Result<T>>({ fetcher: null, nonce: -1 })

  useEffect(() => {
    const controller = new AbortController()
    fetcher(controller.signal).then(
      (data) => setResult({ fetcher, nonce, data }),
      (error: Error) => {
        if (!controller.signal.aborted) {
          setResult((prev) => ({ fetcher, nonce, error, data: prev.fetcher === fetcher ? prev.data : undefined }))
        }
      },
    )
    return () => controller.abort()
  }, [fetcher, nonce])

  const sameSource = result.fetcher === fetcher
  const reload = useCallback(() => setNonce((n) => n + 1), [])

  /** Update the loaded data locally (live updates, optimistic edits). */
  const mutate = useCallback(
    (update: (data: T | undefined) => T | undefined) => {
      setResult((prev) => (prev.fetcher === fetcher ? { ...prev, data: update(prev.data) } : prev))
    },
    [fetcher],
  )

  return {
    data: sameSource ? result.data : undefined,
    error: sameSource && result.nonce === nonce ? result.error : undefined,
    loading: !sameSource || result.nonce !== nonce,
    reload,
    mutate,
  }
}
