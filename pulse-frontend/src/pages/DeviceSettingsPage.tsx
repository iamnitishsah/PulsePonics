import { useCallback, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { ApiKeyDialog } from '../components/ApiKeyDialog'
import { ThresholdEditor } from '../components/ThresholdEditor'
import { Button } from '../components/ui/Button'
import { buttonClass } from '../components/ui/buttonClass'
import { Card, CardHeader } from '../components/ui/Card'
import { EmptyState, Notice } from '../components/ui/Notice'
import { Modal } from '../components/ui/Modal'
import { PageSpinner } from '../components/ui/Spinner'
import { TextField } from '../components/ui/TextField'
import { useQuery } from '../hooks/useQuery'
import { useTitle } from '../hooks/useTitle'
import { ApiError, api, errorMessage } from '../lib/api'
import { formatDate } from '../lib/format'
import type { Device, DeviceWithKey } from '../types/api'

function RenameForm({ device, onRenamed }: { device: Device; onRenamed: (d: Device) => void }) {
  const [name, setName] = useState(device.name)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const updated = await api.renameDevice(device.device_id, name)
      onRenamed({ ...device, name: updated.name })
      setSaved(true)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-wrap items-end gap-3">
      <TextField
        className="min-w-56 flex-1"
        label="Display name"
        value={name}
        onChange={(e) => {
          setName(e.target.value)
          setSaved(false)
        }}
        error={error}
        hint={saved ? 'Saved.' : undefined}
      />
      <Button type="submit" variant="secondary" loading={busy} disabled={!name.trim() || name === device.name}>
        Save name
      </Button>
    </form>
  )
}

function DeleteDevice({ device }: { device: Device }) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const remove = async () => {
    setBusy(true)
    setError(null)
    try {
      await api.deleteDevice(device.device_id)
      navigate('/', { replace: true })
    } catch (err) {
      setError(errorMessage(err))
      setBusy(false)
    }
  }

  return (
    <>
      <Button variant="danger" onClick={() => setOpen(true)}>
        Delete device
      </Button>
      <Modal
        open={open}
        title="Delete this device?"
        onClose={() => setOpen(false)}
        footer={
          <>
            <Button variant="secondary" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button variant="danger" loading={busy} disabled={confirm !== device.device_id} onClick={remove}>
              Delete permanently
            </Button>
          </>
        }
      >
        <div className="space-y-4 text-sm">
          <p className="text-ink-soft">
            All readings, thresholds and alerts for <strong className="font-mono">{device.device_id}</strong> are
            deleted. Telemetry from this ID is ignored until it is registered again.
          </p>
          {error ? <Notice>{error}</Notice> : null}
          <TextField
            label={`Type ${device.device_id} to confirm`}
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="off"
          />
        </div>
      </Modal>
    </>
  )
}

export function DeviceSettingsPage() {
  const { deviceId = '' } = useParams()
  const device = useQuery(useCallback((s: AbortSignal) => api.device(deviceId, s), [deviceId]))
  const thresholds = useQuery(useCallback((s: AbortSignal) => api.thresholds(deviceId, s), [deviceId]))
  const [rotating, setRotating] = useState(false)
  const [rotateError, setRotateError] = useState<string | null>(null)
  const [newKey, setNewKey] = useState<DeviceWithKey | null>(null)
  const [confirmRotate, setConfirmRotate] = useState(false)

  useTitle(device.data ? `${device.data.name} settings` : 'Device settings')

  if (device.error instanceof ApiError && device.error.status === 404) {
    return (
      <EmptyState title="Device not found" action={<Link to="/" className={buttonClass('secondary')}>Back to devices</Link>}>
        <span className="font-mono">{deviceId}</span> isn't registered to your account.
      </EmptyState>
    )
  }
  if (device.error) return <Notice>{errorMessage(device.error)}</Notice>
  if (!device.data) return <PageSpinner label="Loading settings" />

  const d = device.data

  const rotate = async () => {
    setRotating(true)
    setRotateError(null)
    try {
      const updated = await api.rotateDeviceKey(d.device_id)
      setNewKey(updated)
      device.mutate((cur) => (cur ? { ...cur, api_key_prefix: updated.api_key_prefix } : cur))
      setConfirmRotate(false)
    } catch (err) {
      setRotateError(errorMessage(err))
    } finally {
      setRotating(false)
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <nav className="text-sm text-muted" aria-label="Breadcrumb">
          <Link to="/" className="hover:text-forest hover:underline">
            Devices
          </Link>{' '}
          /{' '}
          <Link to={`/devices/${encodeURIComponent(d.device_id)}`} className="hover:text-forest hover:underline">
            {d.name}
          </Link>{' '}
          / Settings
        </nav>
        <h1 className="mt-1 text-2xl font-extrabold">Device settings</h1>
        <p className="mt-1 text-sm text-muted">
          <span className="font-mono">{d.device_id}</span> · registered {formatDate(d.created_at)}
        </p>
      </div>

      {thresholds.error ? (
        <Notice>{errorMessage(thresholds.error)}</Notice>
      ) : thresholds.data ? (
        <ThresholdEditor
          deviceId={d.device_id}
          thresholds={thresholds.data.thresholds}
          onSaved={(ts) => thresholds.mutate(() => ({ thresholds: ts }))}
        />
      ) : (
        <PageSpinner label="Loading thresholds" />
      )}

      <Card>
        <CardHeader eyebrow="Device" title="Name" />
        <div className="p-5">
          <RenameForm device={d} onRenamed={(updated) => device.mutate(() => updated)} />
        </div>
      </Card>

      <Card>
        <CardHeader eyebrow="HTTP ingestion" title="API key" />
        <div className="space-y-3 p-5 text-sm">
          <p className="text-ink-soft">
            Used only for <code className="rounded bg-canvas px-1">POST /api/readings</code> (header{' '}
            <code className="rounded bg-canvas px-1">X-Device-Key</code>). The ESP32 itself publishes over MQTT and does
            not need it.
          </p>
          <p>
            Current key: <code className="rounded bg-canvas px-1.5 py-0.5 font-mono">{d.api_key_prefix}…</code>
          </p>
          {rotateError ? <Notice>{rotateError}</Notice> : null}
          <Button variant="secondary" onClick={() => setConfirmRotate(true)}>
            Generate new key
          </Button>
        </div>
      </Card>

      <Card className="border-danger/30">
        <CardHeader eyebrow="Danger zone" title="Delete device" />
        <div className="flex flex-wrap items-center justify-between gap-3 p-5 text-sm">
          <p className="max-w-md text-ink-soft">Removes the device and its entire history. This cannot be undone.</p>
          <DeleteDevice device={d} />
        </div>
      </Card>

      <Modal
        open={confirmRotate}
        title="Generate a new API key?"
        onClose={() => setConfirmRotate(false)}
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmRotate(false)}>
              Cancel
            </Button>
            <Button loading={rotating} onClick={rotate}>
              Generate key
            </Button>
          </>
        }
      >
        <p className="text-sm text-ink-soft">The current key stops working immediately. MQTT telemetry is not affected.</p>
      </Modal>
      <ApiKeyDialog device={newKey} onClose={() => setNewKey(null)} />
    </div>
  )
}
