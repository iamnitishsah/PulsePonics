import type { Reading, SensorConfig } from '../types/reading'
import { formatTime } from '../utils/time'

type LineChartProps = {
  readings: Reading[]
  config: SensorConfig
}

const WIDTH = 640
const HEIGHT = 220
const PADDING = 28

function getPoints(readings: Reading[], config: SensorConfig) {
  return readings
    .map((reading) => ({
      time: new Date(reading.recorded_at).getTime(),
      value: reading[config.key],
      label: reading.recorded_at,
    }))
    .filter((point): point is { time: number; value: number; label: string } => {
      return Number.isFinite(point.time) && typeof point.value === 'number'
    })
}

export function LineChart({ readings, config }: LineChartProps) {
  const points = getPoints(readings, config)

  if (points.length < 2) {
    return (
      <div className="m-5 grid min-h-[260px] place-items-center rounded-lg border border-dashed border-[#17211d]/20 bg-[#f8faf7] text-center text-[#75847d] md:min-h-[300px]">
        <span>No chartable {config.label.toLowerCase()} trend yet</span>
      </div>
    )
  }

  const minTime = Math.min(...points.map((point) => point.time))
  const maxTime = Math.max(...points.map((point) => point.time))
  const rawMin = Math.min(...points.map((point) => point.value), config.range[0])
  const rawMax = Math.max(...points.map((point) => point.value), config.range[1])
  const valuePadding = (rawMax - rawMin || 1) * 0.12
  const minValue = rawMin - valuePadding
  const maxValue = rawMax + valuePadding

  const xFor = (time: number) => {
    if (maxTime === minTime) return WIDTH / 2
    return PADDING + ((time - minTime) / (maxTime - minTime)) * (WIDTH - PADDING * 2)
  }

  const yFor = (value: number) => {
    return HEIGHT - PADDING - ((value - minValue) / (maxValue - minValue)) * (HEIGHT - PADDING * 2)
  }

  const path = points
    .map((point, index) => `${index === 0 ? 'M' : 'L'} ${xFor(point.time)} ${yFor(point.value)}`)
    .join(' ')

  const latestPoint = points[points.length - 1]
  const firstPoint = points[0]

  return (
    <div className="px-5 pb-5 pt-2">
      <svg
        className="block h-60 w-full overflow-visible md:h-[300px]"
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        role="img"
      >
        <title>{config.label} trend</title>
        <line
          className="stroke-[#17211d]/20"
          x1={PADDING}
          x2={WIDTH - PADDING}
          y1={HEIGHT - PADDING}
          y2={HEIGHT - PADDING}
        />
        <line
          className="stroke-[#17211d]/20"
          x1={PADDING}
          x2={PADDING}
          y1={PADDING}
          y2={HEIGHT - PADDING}
        />
        {config.ideal ? (
          <rect
            className="fill-[#8ed4b8]/25"
            x={PADDING}
            y={yFor(config.ideal[1])}
            width={WIDTH - PADDING * 2}
            height={Math.max(2, yFor(config.ideal[0]) - yFor(config.ideal[1]))}
          />
        ) : null}
        <path
          className="fill-none stroke-[#2f6b57] stroke-4 [stroke-linecap:round] [stroke-linejoin:round]"
          d={path}
        />
        <circle
          className="fill-[#17211d] stroke-white stroke-[3px]"
          cx={xFor(latestPoint.time)}
          cy={yFor(latestPoint.value)}
          r="5"
        />
      </svg>
      <div className="flex justify-between text-xs font-bold text-[#78887f]">
        <span>{formatTime(firstPoint.label)}</span>
        <span>{formatTime(latestPoint.label)}</span>
      </div>
    </div>
  )
}
