type StatusPillProps = {
  state: 'connecting' | 'live' | 'offline'
}

const LABELS = {
  connecting: 'Connecting',
  live: 'Live',
  offline: 'Offline',
}

export function StatusPill({ state }: StatusPillProps) {
  const dotClass =
    state === 'live'
      ? 'bg-[#1d9d68] shadow-[0_0_0_5px_rgba(29,157,104,0.14)]'
      : state === 'offline'
        ? 'bg-[#c24141]'
        : 'bg-[#9aa79f]'

  return (
    <span className="inline-flex min-h-9 min-w-[108px] items-center justify-center gap-2 rounded-full border border-[#17211d]/10 bg-white px-3 py-2 text-sm font-bold text-[#45564f]">
      <span className={`h-2.5 w-2.5 rounded-full ${dotClass}`} />
      {LABELS[state]}
    </span>
  )
}
