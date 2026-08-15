type DeviceSelectorProps = {
  deviceId: string
  onChange: (deviceId: string) => void
  onRefresh: () => void
  loading: boolean
}

export function DeviceSelector({ deviceId, onChange, onRefresh, loading }: DeviceSelectorProps) {
  return (
    <form
      className="grid w-full grid-cols-[1fr_44px] items-end gap-2 md:w-auto md:grid-cols-[minmax(190px,280px)_44px]"
      onSubmit={(event) => {
        event.preventDefault()
        onRefresh()
      }}
    >
      <label
        className="col-span-full text-xs font-extrabold text-white/75"
        htmlFor="device-id"
      >
        Device
      </label>
      <input
        id="device-id"
        className="min-h-11 w-full rounded-lg border border-white/20 bg-white/10 px-3 text-white outline-none placeholder:text-white/35 focus:border-[#8ed4b8] focus:ring-4 focus:ring-[#8ed4b8]/20"
        value={deviceId}
        onChange={(event) => onChange(event.target.value)}
        placeholder="esp32-tank-01"
      />
      <button
        className="grid h-11 w-11 place-items-center rounded-lg bg-[#8ed4b8] text-xl font-extrabold text-[#10231b] transition hover:bg-[#9ddfc6] disabled:cursor-wait disabled:opacity-60"
        type="submit"
        disabled={loading}
        title="Refresh readings"
      >
        ↻
      </button>
    </form>
  )
}
