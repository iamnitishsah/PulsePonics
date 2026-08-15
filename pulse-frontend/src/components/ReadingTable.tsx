import type { Reading } from '../types/reading'
import { SENSOR_CONFIGS, formatSensorValue, getSensorValue } from '../utils/sensors'
import { formatDateTime } from '../utils/time'

type ReadingTableProps = {
  readings: Reading[]
}

export function ReadingTable({ readings }: ReadingTableProps) {
  const rows = [...readings].reverse().slice(0, 8)

  return (
    <section className="mt-4 rounded-lg border border-[#17211d]/10 bg-white shadow-[0_18px_50px_rgba(45,57,50,0.08)]">
      <div className="flex flex-col items-start justify-between gap-4 px-5 pt-5 md:flex-row md:items-center">
        <div>
          <span className="text-xs font-extrabold uppercase tracking-normal text-[#5a7c6d]">
            Recent
          </span>
          <h2 className="text-lg font-extrabold leading-tight text-[#14201b]">Sensor readings</h2>
        </div>
      </div>
      <div className="w-full overflow-x-auto px-5 pb-5 pt-3">
        <table className="w-full min-w-[760px] border-collapse">
          <thead>
            <tr>
              <th className="border-b border-[#17211d]/10 px-3 py-3 text-left text-xs font-black uppercase text-[#65756e]">
                Recorded
              </th>
              {SENSOR_CONFIGS.slice(0, 5).map((sensor) => (
                <th
                  key={sensor.key}
                  className="border-b border-[#17211d]/10 px-3 py-3 text-left text-xs font-black uppercase text-[#65756e]"
                >
                  {sensor.shortLabel}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.length ? (
              rows.map((reading) => (
                <tr key={reading.id}>
                  <td className="whitespace-nowrap border-b border-[#17211d]/10 px-3 py-3 text-sm font-bold text-[#27342f]">
                    {formatDateTime(reading.recorded_at)}
                  </td>
                  {SENSOR_CONFIGS.slice(0, 5).map((sensor) => (
                    <td
                      key={sensor.key}
                      className="whitespace-nowrap border-b border-[#17211d]/10 px-3 py-3 text-sm font-bold text-[#27342f]"
                    >
                      {formatSensorValue(getSensorValue(reading, sensor.key), sensor)}
                    </td>
                  ))}
                </tr>
              ))
            ) : (
              <tr>
                <td
                  className="border-b border-[#17211d]/10 px-3 py-6 text-sm font-bold text-[#74847d]"
                  colSpan={6}
                >
                  No readings found for this device.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  )
}
