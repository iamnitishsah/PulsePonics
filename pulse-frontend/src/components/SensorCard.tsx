import type { Reading, SensorConfig } from '../types/reading'
import { formatSensorValue, getSensorState, getSensorValue } from '../utils/sensors'

type SensorCardProps = {
  config: SensorConfig
  latest: Reading | null
}

export function SensorCard({ config, latest }: SensorCardProps) {
  const value = getSensorValue(latest, config.key)
  const state = getSensorState(value, config)
  const stateStyles =
    state === 'Optimal'
      ? 'bg-[#e2f5ec] text-[#13734e]'
      : state === 'Low' || state === 'High'
        ? 'bg-[#fff1d8] text-[#8a5a11]'
        : 'bg-[#edf2ee] text-[#5b6b64]'

  return (
    <article className="min-h-[148px] rounded-lg border border-[#17211d]/10 bg-white p-4 shadow-[0_18px_50px_rgba(45,57,50,0.08)]">
      <div className="flex min-h-7 items-center justify-between gap-2 text-sm font-extrabold text-[#586a62]">
        <span>{config.label}</span>
        <span className={`shrink-0 rounded-full px-2 py-1 text-[0.72rem] ${stateStyles}`}>
          {state}
        </span>
      </div>
      <strong className="mt-5 block text-[clamp(1.55rem,2.4vw,2.4rem)] font-extrabold leading-none text-[#14201b]">
        {formatSensorValue(value, config)}
      </strong>
      {config.ideal ? (
        <span className="mt-5 block text-sm text-[#75847d]">
          Target {config.ideal[0]}-{config.ideal[1]} {config.unit}
        </span>
      ) : (
        <span className="mt-5 block text-sm text-[#75847d]">Environmental context</span>
      )}
    </article>
  )
}
