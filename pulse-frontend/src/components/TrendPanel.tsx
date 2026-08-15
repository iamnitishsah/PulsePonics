import { useState } from 'react'
import type { Reading, SensorKey } from '../types/reading'
import { LineChart } from './LineChart'
import { SENSOR_CONFIGS, getSensorConfig } from '../utils/sensors'

type TrendPanelProps = {
  readings: Reading[]
}

export function TrendPanel({ readings }: TrendPanelProps) {
  const [activeSensor, setActiveSensor] = useState<SensorKey>('ph')
  const config = getSensorConfig(activeSensor)

  return (
    <section className="rounded-lg border border-[#17211d]/10 bg-white shadow-[0_18px_50px_rgba(45,57,50,0.08)]">
      <div className="flex flex-col items-start justify-between gap-4 px-5 pt-5 md:flex-row md:items-center">
        <div>
          <span className="text-xs font-extrabold uppercase tracking-normal text-[#5a7c6d]">
            Trend
          </span>
          <h2 className="text-lg font-extrabold leading-tight text-[#14201b]">{config.label}</h2>
        </div>
        <div
          className="inline-flex min-h-10 w-full overflow-x-auto rounded-lg border border-[#17211d]/10 bg-[#f3f6f1] p-1 md:w-auto"
          aria-label="Select chart sensor"
        >
          {SENSOR_CONFIGS.slice(0, 5).map((sensor) => (
            <button
              key={sensor.key}
              className={`min-w-14 flex-1 rounded-md px-3 text-sm font-extrabold transition md:flex-none ${
                sensor.key === activeSensor
                  ? 'bg-[#17211d] text-white'
                  : 'bg-transparent text-[#586a62] hover:bg-white'
              }`}
              type="button"
              onClick={() => setActiveSensor(sensor.key)}
            >
              {sensor.shortLabel}
            </button>
          ))}
        </div>
      </div>
      <LineChart readings={readings} config={config} />
    </section>
  )
}
