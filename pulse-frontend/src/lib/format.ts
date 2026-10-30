const dateTime = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
})
const dateOnly = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'long', year: 'numeric' })
const timeOnly = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' })
const dayAndTime = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
const dayOnly = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' })

function toDate(value: string | number | Date | null | undefined) {
  if (value === null || value === undefined) return null
  const d = value instanceof Date ? value : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDateTime(value: string | number | Date | null | undefined) {
  const d = toDate(value)
  return d ? dateTime.format(d) : '—'
}

export function formatDate(value: string | null | undefined) {
  const d = toDate(value)
  return d ? dateOnly.format(d) : '—'
}

/** Chart axis label; precision depends on how long the visible range is. */
export function formatAxisTime(ms: number, rangeMs: number) {
  if (rangeMs <= 24 * 3600_000) return timeOnly.format(ms)
  if (rangeMs <= 7 * 24 * 3600_000) return dayAndTime.format(ms)
  return dayOnly.format(ms)
}

export function timeAgo(value: string | null | undefined, now = Date.now()) {
  const d = toDate(value)
  if (!d) return 'never'
  const seconds = Math.max(0, Math.round((now - d.getTime()) / 1000))
  if (seconds < 10) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

export function isFresh(value: string | null | undefined, staleAfterMs: number, now = Date.now()) {
  const d = toDate(value)
  return d !== null && now - d.getTime() <= staleAfterMs
}
