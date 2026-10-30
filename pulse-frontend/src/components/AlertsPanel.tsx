import { useState } from 'react'
import { api, errorMessage } from '../lib/api'
import { formatDateTime, timeAgo } from '../lib/format'
import { METRIC_BY_KEY, formatValue } from '../lib/metrics'
import type { Alert } from '../types/api'
import { Button } from './ui/Button'
import { Card, CardHeader } from './ui/Card'
import { Badge, Notice } from './ui/Notice'

function describe(alert: Alert) {
  const meta = METRIC_BY_KEY[alert.metric]
  const direction = alert.kind === 'high' ? 'above' : 'below'
  return `${meta.label} ${direction} ${formatValue(alert.threshold_value, meta)}`
}

function AlertRow({ alert, now, onAcknowledged }: { alert: Alert; now: number; onAcknowledged: (a: Alert) => void }) {
  const meta = METRIC_BY_KEY[alert.metric]
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const open = alert.status === 'open'

  const acknowledge = async () => {
    setBusy(true)
    setError(null)
    try {
      onAcknowledged(await api.acknowledgeAlert(alert.id))
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className={`rounded-lg border px-4 py-3 ${open ? 'border-danger/30 bg-danger-soft/60' : 'border-line'}`}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className={`font-semibold ${open ? 'text-danger' : 'text-ink'}`}>{describe(alert)}</p>
          <p className="mt-0.5 text-xs text-muted">
            {open
              ? `Started ${timeAgo(alert.opened_at, now)} at ${formatValue(alert.trigger_value, meta)} · now ${formatValue(alert.latest_value, meta)}`
              : `${formatDateTime(alert.opened_at)} → ${formatDateTime(alert.resolved_at)}`}
          </p>
        </div>
        {open ? (
          alert.acknowledged_at ? (
            <Badge tone="neutral">Acknowledged</Badge>
          ) : (
            <Button size="sm" variant="secondary" loading={busy} onClick={acknowledge}>
              Acknowledge
            </Button>
          )
        ) : (
          <Badge tone="ok">Resolved</Badge>
        )}
      </div>
      {error ? <p className="mt-2 text-xs text-danger">{error}</p> : null}
    </li>
  )
}

export function AlertsPanel({
  alerts,
  error,
  now,
  onAcknowledged,
}: {
  alerts: Alert[] | undefined
  error?: Error
  now: number
  onAcknowledged: (a: Alert) => void
}) {
  const open = alerts?.filter((a) => a.status === 'open') ?? []
  const resolved = alerts?.filter((a) => a.status === 'resolved').slice(0, 8) ?? []

  return (
    <Card className="flex flex-col">
      <CardHeader
        eyebrow="Alerts"
        title={open.length ? `${open.length} active` : 'All clear'}
        actions={open.length ? <Badge tone="danger">Action needed</Badge> : <Badge tone="ok">In range</Badge>}
      />
      <div className="space-y-4 p-5">
        {error ? <Notice>{errorMessage(error)}</Notice> : null}
        {open.length > 0 ? (
          <ul className="space-y-2">
            {open.map((a) => (
              <AlertRow key={a.id} alert={a} now={now} onAcknowledged={onAcknowledged} />
            ))}
          </ul>
        ) : (
          <p className="text-sm text-muted">Every monitored value is inside its limits.</p>
        )}
        {resolved.length > 0 ? (
          <div>
            <p className="mb-2 text-xs font-bold uppercase tracking-wide text-muted">Recently resolved</p>
            <ul className="space-y-2">
              {resolved.map((a) => (
                <AlertRow key={a.id} alert={a} now={now} onAcknowledged={onAcknowledged} />
              ))}
            </ul>
          </div>
        ) : null}
      </div>
    </Card>
  )
}
