import { useCallback, useState } from 'react'
import { Link } from 'react-router'
import { AddDeviceDialog } from '../components/AddDeviceDialog'
import { ApiKeyDialog } from '../components/ApiKeyDialog'
import { DeviceStatus } from '../components/DeviceStatus'
import { Button } from '../components/ui/Button'
import { Badge, EmptyState, Notice } from '../components/ui/Notice'
import { PageSpinner } from '../components/ui/Spinner'
import { useNow } from '../hooks/useNow'
import { useQuery } from '../hooks/useQuery'
import { useTitle } from '../hooks/useTitle'
import { api, errorMessage } from '../lib/api'
import { METRIC_BY_KEY, formatValue, valueOf } from '../lib/metrics'
import { useOnReconnect, useRealtimeEvents } from '../realtime/useRealtime'
import type { Device, DeviceWithKey, MetricKey } from '../types/api'

const PREVIEW: MetricKey[] = ['ph', 'ec_ms_cm', 'water_temp_c', 'air_temp_c']

function DeviceCard({ device, now }: { device: Device; now: number }) {
  return (
    <Link
      to={`/devices/${encodeURIComponent(device.device_id)}`}
      className="group block rounded-xl border border-line bg-surface p-5 shadow-card transition hover:border-forest/40 hover:shadow-lg"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="truncate text-base font-bold text-ink group-hover:text-forest">{device.name}</h2>
          <p className="truncate font-mono text-xs text-muted">{device.device_id}</p>
        </div>
        {device.open_alerts > 0 ? (
          <Badge tone="danger">
            {device.open_alerts} alert{device.open_alerts > 1 ? 's' : ''}
          </Badge>
        ) : (
          <Badge tone="ok">No alerts</Badge>
        )}
      </div>
      <dl className="mt-5 grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-4">
        {PREVIEW.map((key) => {
          const meta = METRIC_BY_KEY[key]
          return (
            <div key={key}>
              <dt className="text-xs font-semibold text-muted">{meta.short}</dt>
              <dd className="mt-0.5 text-lg font-bold tabular-nums text-ink">
                {formatValue(valueOf(device.latest, key), meta, false)}
                {meta.unit && valueOf(device.latest, key) !== null ? (
                  <span className="ml-0.5 text-xs font-semibold text-muted">{meta.unit}</span>
                ) : null}
              </dd>
            </div>
          )
        })}
      </dl>
      <div className="mt-5 border-t border-line pt-3">
        <DeviceStatus lastSeen={device.latest?.recorded_at} now={now} />
      </div>
    </Link>
  )
}

export function DevicesPage() {
  useTitle('Devices')
  const now = useNow()
  const fetchDevices = useCallback((signal: AbortSignal) => api.devices(signal), [])
  const { data: devices, error, loading, reload, mutate } = useQuery(fetchDevices)
  const [adding, setAdding] = useState(false)
  const [created, setCreated] = useState<DeviceWithKey | null>(null)

  useRealtimeEvents((event) => {
    if (event.type === 'reading') {
      mutate((list) => list?.map((d) => (d.device_id === event.data.device_id ? { ...d, latest: event.data } : d)))
    } else {
      reload() // alert counts changed
    }
  })

  useOnReconnect(reload)

  return (
    <div>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-bold uppercase tracking-wide text-forest">Overview</p>
          <h1 className="text-2xl font-extrabold">Your devices</h1>
        </div>
        <Button onClick={() => setAdding(true)}>+ Register device</Button>
      </div>

      <div className="mt-6">
        {error ? (
          <Notice action={<Button variant="secondary" size="sm" onClick={reload}>Retry</Button>}>
            {errorMessage(error)}
          </Notice>
        ) : loading && !devices ? (
          <PageSpinner label="Loading devices" />
        ) : devices && devices.length === 0 ? (
          <EmptyState
            title="No devices yet"
            action={<Button onClick={() => setAdding(true)}>Register your first device</Button>}
          >
            Register your ESP32 with the same <code className="rounded bg-canvas px-1">DEVICE_ID</code> it uses in
            its firmware. Its MQTT telemetry is ignored until then.
          </EmptyState>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {devices?.map((device) => <DeviceCard key={device.device_id} device={device} now={now} />)}
          </div>
        )}
      </div>

      <AddDeviceDialog
        open={adding}
        onClose={() => setAdding(false)}
        onCreated={(device) => {
          setAdding(false)
          setCreated(device)
          reload()
        }}
      />
      <ApiKeyDialog device={created} onClose={() => setCreated(null)} />
    </div>
  )
}
