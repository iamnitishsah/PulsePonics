import { useCallback, useState } from 'react'
import { Link, useParams } from 'react-router'
import { AlertsPanel } from '../components/AlertsPanel'
import { DeviceStatus } from '../components/DeviceStatus'
import { HistoryChart } from '../components/HistoryChart'
import { MetricCard } from '../components/MetricCard'
import { ReadingsTable } from '../components/ReadingsTable'
import { Button } from '../components/ui/Button'
import { buttonClass } from '../components/ui/buttonClass'
import { EmptyState, Notice } from '../components/ui/Notice'
import { PageSpinner } from '../components/ui/Spinner'
import { useNow } from '../hooks/useNow'
import { useQuery } from '../hooks/useQuery'
import { useTitle } from '../hooks/useTitle'
import { ApiError, api, errorMessage } from '../lib/api'
import { METRICS, valueOf } from '../lib/metrics'
import { useOnReconnect, useRealtimeEvents } from '../realtime/useRealtime'
import type { MetricKey, Reading } from '../types/api'

const RECENT_LIMIT = 10

/** Chart the first sensor that actually reports, e.g. water temperature on
 *  the current DS18B20-only hardware build. */
function firstMetricWithData(latest: Reading | null | undefined): MetricKey | null {
  return METRICS.find((m) => valueOf(latest, m.key) !== null)?.key ?? null
}

export function DevicePage() {
  const { deviceId = '' } = useParams()
  const now = useNow()

  const device = useQuery(useCallback((s: AbortSignal) => api.device(deviceId, s), [deviceId]))
  const thresholds = useQuery(useCallback((s: AbortSignal) => api.thresholds(deviceId, s), [deviceId]))
  const recent = useQuery(useCallback((s: AbortSignal) => api.recentReadings(deviceId, RECENT_LIMIT, s), [deviceId]))
  const alerts = useQuery(useCallback((s: AbortSignal) => api.alerts({ deviceId, status: 'all', limit: 30 }, s), [deviceId]))

  const [picked, setPicked] = useState<MetricKey | null>(null)
  const metric = picked ?? firstMetricWithData(device.data?.latest) ?? 'ph'

  useTitle(device.data?.name ?? deviceId)

  useRealtimeEvents((event) => {
    if (event.data.device_id !== deviceId) return
    if (event.type === 'reading') {
      const reading = event.data
      device.mutate((d) => (d ? { ...d, latest: reading } : d))
      recent.mutate((list) => [reading, ...(list ?? []).filter((r) => r.id !== reading.id)].slice(0, RECENT_LIMIT))
    } else {
      alerts.reload()
      device.reload()
    }
  })

  useOnReconnect(() => {
    device.reload()
    recent.reload()
    alerts.reload()
  })

  if (device.error instanceof ApiError && device.error.status === 404) {
    return (
      <EmptyState title="Device not found" action={<Link to="/" className={buttonClass('secondary')}>Back to devices</Link>}>
        <span className="font-mono">{deviceId}</span> isn't registered to your account.
      </EmptyState>
    )
  }
  if (device.error && !device.data) {
    return (
      <Notice action={<Button variant="secondary" size="sm" onClick={device.reload}>Retry</Button>}>
        {errorMessage(device.error)}
      </Notice>
    )
  }
  if (!device.data) return <PageSpinner label="Loading device" />

  const d = device.data
  const ts = thresholds.data?.thresholds

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="min-w-0">
          <nav className="text-sm text-muted" aria-label="Breadcrumb">
            <Link to="/" className="hover:text-forest hover:underline">
              Devices
            </Link>{' '}
            / <span className="font-mono">{d.device_id}</span>
          </nav>
          <h1 className="mt-1 truncate text-2xl font-extrabold sm:text-3xl">{d.name}</h1>
          <div className="mt-1">
            <DeviceStatus lastSeen={d.latest?.recorded_at} now={now} />
          </div>
        </div>
        <Link to={`/devices/${encodeURIComponent(d.device_id)}/settings`} className={buttonClass('secondary')}>
          Thresholds &amp; settings
        </Link>
      </div>

      {thresholds.error ? <Notice>Couldn't load thresholds: {errorMessage(thresholds.error)}</Notice> : null}

      <section aria-label="Latest values" className="grid grid-cols-2 gap-3 md:grid-cols-3 2xl:grid-cols-6">
        {METRICS.map((m) => (
          <MetricCard
            key={m.key}
            metric={m.key}
            value={valueOf(d.latest, m.key)}
            threshold={ts?.find((t) => t.metric === m.key)}
            selected={m.key === metric}
            onSelect={() => setPicked(m.key)}
          />
        ))}
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
        <HistoryChart deviceId={d.device_id} metric={metric} onMetricChange={setPicked} thresholds={ts} />
        <AlertsPanel
          alerts={alerts.data}
          error={alerts.error}
          now={now}
          onAcknowledged={(updated) => alerts.mutate((list) => list?.map((a) => (a.id === updated.id ? updated : a)))}
        />
      </div>

      <ReadingsTable readings={recent.data} thresholds={ts} />
    </div>
  )
}
