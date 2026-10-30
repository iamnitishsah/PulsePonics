import { STALE_AFTER_MS } from '../lib/config'
import { isFresh, timeAgo } from '../lib/format'

/** "Live" while readings keep arriving, otherwise when the device was last heard from. */
export function DeviceStatus({ lastSeen, now }: { lastSeen: string | null | undefined; now: number }) {
  if (!lastSeen) {
    return (
      <span className="inline-flex items-center gap-1.5 text-sm text-muted">
        <span className="size-2 rounded-full bg-ink/25" />
        No data yet
      </span>
    )
  }
  const fresh = isFresh(lastSeen, STALE_AFTER_MS, now)
  return (
    <span className={`inline-flex items-center gap-1.5 text-sm ${fresh ? 'text-ok' : 'text-warn'}`}>
      <span className={`size-2 rounded-full ${fresh ? 'bg-ok' : 'bg-warn'}`} />
      {fresh ? `Live · ${timeAgo(lastSeen, now)}` : `Offline · last seen ${timeAgo(lastSeen, now)}`}
    </span>
  )
}
