import { useState } from 'react'
import { DeviceSelector } from './components/DeviceSelector'
import { ReadingTable } from './components/ReadingTable'
import { SensorCard } from './components/SensorCard'
import { StatusPill } from './components/StatusPill'
import { TrendPanel } from './components/TrendPanel'
import { useTelemetry } from './hooks/useTelemetry'
import { SENSOR_CONFIGS } from './utils/sensors'
import { formatDateTime, timeAgo } from './utils/time'

function App() {
  const [deviceId, setDeviceId] = useState('esp32-tank-01')
  const { history, latest, loading, error, connectionState, refresh, stats } = useTelemetry(deviceId)

  return (
    <main className="mx-auto min-h-screen w-full max-w-[1440px] bg-[#f4f7f2] p-3 text-[#17211d] sm:p-6">
      <header className="flex min-h-[78px] flex-col items-start justify-between gap-3 rounded-lg border border-[#17211d]/10 bg-white/85 px-4 py-4 shadow-[0_18px_50px_rgba(45,57,50,0.08)] backdrop-blur md:flex-row md:items-center">
        <div>
          <span className="mb-1 block text-xs font-extrabold uppercase tracking-normal text-[#2f6b57]">
            PulsePonics
          </span>
          <h1 className="text-[clamp(1.35rem,2vw,2rem)] font-extrabold leading-tight text-[#14201b]">
            Hydroponics telemetry
          </h1>
        </div>
        <StatusPill state={connectionState} />
      </header>

      <section className="mt-5 grid items-end gap-6 rounded-lg border border-[#17211d]/10 bg-[#17211d] p-5 text-white/80 shadow-[0_18px_50px_rgba(45,57,50,0.08)] md:grid-cols-[1fr_auto] md:p-7">
        <div>
          <span className="text-xs font-extrabold uppercase tracking-normal text-[#a8d7c2]">
            Realtime control room
          </span>
          <h2 className="mt-2 [overflow-wrap:anywhere] text-[clamp(2rem,5vw,4.5rem)] font-extrabold leading-none text-white">
            {deviceId || 'No device selected'}
          </h2>
          <p className="mt-4 max-w-3xl text-base leading-7">
            Latest sample {stats.lastUpdated ? timeAgo(stats.lastUpdated) : 'not available'}.
            Showing {stats.sampleCount} readings from the backend history window.
          </p>
        </div>
        <DeviceSelector
          deviceId={deviceId}
          onChange={setDeviceId}
          onRefresh={refresh}
          loading={loading}
        />
      </section>

      {error ? (
        <div
          className="mt-4 flex flex-wrap gap-x-3 gap-y-2 rounded-lg border border-[#c24141]/25 bg-[#fff8f6] px-4 py-3 text-[#883431] shadow-[0_18px_50px_rgba(45,57,50,0.08)]"
          role="alert"
        >
          <strong>Backend unavailable</strong>
          <span>{error}</span>
        </div>
      ) : null}

      <section
        className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-6"
        aria-label="Latest sensor values"
      >
        {SENSOR_CONFIGS.map((sensor) => (
          <SensorCard key={sensor.key} config={sensor} latest={latest} />
        ))}
      </section>

      <section className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1.8fr)_minmax(280px,0.7fr)]">
        <TrendPanel readings={history} />
        <aside className="min-h-full rounded-lg border border-[#17211d]/10 bg-white shadow-[0_18px_50px_rgba(45,57,50,0.08)]">
          <div className="flex flex-col items-start justify-between gap-4 px-5 pt-5 md:flex-row md:items-center">
            <div>
              <span className="text-xs font-extrabold uppercase tracking-normal text-[#5a7c6d]">
                Snapshot
              </span>
              <h2 className="text-lg font-extrabold leading-tight text-[#14201b]">Current state</h2>
            </div>
          </div>
          <dl className="grid gap-3 p-5">
            <div className="border-b border-[#17211d]/10 pb-3">
              <dt className="text-xs font-extrabold uppercase text-[#74847d]">Device ID</dt>
              <dd className="mt-1 [overflow-wrap:anywhere] font-extrabold text-[#17211d]">
                {latest?.device_id ?? deviceId}
              </dd>
            </div>
            <div className="border-b border-[#17211d]/10 pb-3">
              <dt className="text-xs font-extrabold uppercase text-[#74847d]">Latest record</dt>
              <dd className="mt-1 font-extrabold text-[#17211d]">
                {formatDateTime(latest?.recorded_at)}
              </dd>
            </div>
            <div className="border-b border-[#17211d]/10 pb-3">
              <dt className="text-xs font-extrabold uppercase text-[#74847d]">Storage record</dt>
              <dd className="mt-1 font-extrabold text-[#17211d]">
                {formatDateTime(latest?.created_at)}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-extrabold uppercase text-[#74847d]">Realtime feed</dt>
              <dd className="mt-1 font-extrabold capitalize text-[#17211d]">{connectionState}</dd>
            </div>
          </dl>
        </aside>
      </section>

      <ReadingTable readings={history} />
    </main>
  )
}

export default App
